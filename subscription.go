package junglebus

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/GorillaPool/go-junglebus/models"
	"github.com/centrifugal/centrifuge-go"
	"google.golang.org/protobuf/proto"
)

// Subscription represents an active subscription to the JungleBus service.
// It manages the WebSocket connection, event processing, and position tracking.
type Subscription struct {
	// Public fields (preserved for API compatibility)
	SubscriptionID string
	FromBlock      uint64
	EventHandler   EventHandler

	// Internal state management
	mu      sync.RWMutex
	state   subscriptionState
	client  *Client
	options *SubscribeOptions

	// Position tracking (per-subscription, replaces global vars)
	position *subscriptionPosition

	// Connection management
	centrifugeClient *centrifuge.Client
	channels         *channelManager
	mainChannelName  string // Track current main channel name for reconnect updates
	hasConnected     bool   // Track if we've ever successfully connected

	// Event processing
	eventQueue *eventQueue

	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// pubEvent represents an event received from a subscription channel
type pubEvent struct {
	Channel string
	Data    []byte
}

// SubscribeOptions configures subscription behavior
type SubscribeOptions struct {
	QueueSize uint32
	LiteMode  bool
}

// Unsubscribe closes the subscription and releases all resources.
// It's safe to call multiple times.
func (s *Subscription) Unsubscribe() error {
	s.mu.Lock()
	if s.state == stateClosed {
		s.mu.Unlock()
		return nil
	}
	s.state = stateClosed
	s.mu.Unlock()

	// Cancel context first - signals all goroutines to stop
	if s.cancel != nil {
		s.cancel()
	}

	var errs []error

	// Unsubscribe from all channels
	if s.channels != nil {
		if err := s.channels.UnsubscribeAll(); err != nil {
			errs = append(errs, err)
		}
	}

	// Close the centrifuge client
	if s.centrifugeClient != nil {
		if err := s.centrifugeClient.Disconnect(); err != nil {
			errs = append(errs, fmt.Errorf("disconnect: %w", err))
		}
	}

	// Close event queue and wait for processing to complete
	if s.eventQueue != nil {
		s.eventQueue.Close()
		s.eventQueue.Wait()
	}

	// Signal completion
	if s.done != nil {
		close(s.done)
	}

	return errors.Join(errs...)
}

// Done returns a channel that's closed when the subscription is fully closed
func (s *Subscription) Done() <-chan struct{} {
	return s.done
}

// State returns the current subscription state
func (s *Subscription) State() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.String()
}

// Position returns the current block and page position
func (s *Subscription) Position() (block uint32, page uint64) {
	if s.position == nil {
		return 0, 0
	}
	return s.position.Get()
}

// setState updates the subscription state (internal use)
func (s *Subscription) setState(state subscriptionState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = state
}

// getState returns the current state (internal use)
func (s *Subscription) getState() subscriptionState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// addToQueue safely adds an event to the processing queue
func (s *Subscription) addToQueue(event *pubEvent) {
	if !s.eventQueue.SendBlocking(event) {
		// Queue is closed, subscription is shutting down
		return
	}
}

// handleEvents processes events from the queue (runs in a goroutine)
func (s *Subscription) handleEvents() {
	for event := range s.eventQueue.Channel() {
		s.processEvent(event)
		s.eventQueue.Done()
	}
}

// processEvent handles a single event with proper error handling
func (s *Subscription) processEvent(event *pubEvent) {
	// Recover from panics in event handlers
	defer func() {
		if r := recover(); r != nil {
			if s.EventHandler.OnError != nil {
				s.EventHandler.OnError(fmt.Errorf("panic in event handler: %v", r))
			}
		}
	}()

	switch event.Channel {
	case "control":
		s.handleControlEvent(event.Data)
	case "main":
		s.handleTransactionEvent(event.Data)
	case "mempool":
		s.handleMempoolEvent(event.Data)
	}
}

// handleControlEvent processes control/status messages
func (s *Subscription) handleControlEvent(data []byte) {
	status := &models.ControlResponse{}
	if err := proto.Unmarshal(data, status); err != nil {
		if s.EventHandler.OnError != nil {
			s.EventHandler.OnError(fmt.Errorf("unmarshal control: %w", err))
		}
		return
	}

	// Update position based on status
	switch StatusCode(status.StatusCode) {
	case SubscriptionBlockDone:
		s.position.AdvanceBlock(status.Block + 1)
	case SubscriptionPageDone:
		s.position.AdvancePage(status.Block, status.Transactions+1)
	}

	if s.EventHandler.OnStatus != nil {
		s.EventHandler.OnStatus(status)
	}
}

// handleTransactionEvent processes block transaction messages
func (s *Subscription) handleTransactionEvent(data []byte) {
	tx := &models.TransactionResponse{}
	if err := proto.Unmarshal(data, tx); err != nil {
		if s.EventHandler.OnError != nil {
			s.EventHandler.OnError(fmt.Errorf("unmarshal transaction: %w", err))
		}
		return
	}

	// Fetch full transaction data if needed
	if len(tx.Transaction) == 0 && !s.options.LiteMode {
		txData, err := s.client.GetTransaction(s.ctx, tx.Id)
		if err != nil {
			if s.EventHandler.OnError != nil {
				s.EventHandler.OnError(fmt.Errorf("fetch transaction %s: %w", tx.Id, err))
			}
			return
		}
		tx.Transaction = txData.Transaction
	}

	// Update position
	s.position.SetBlock(tx.BlockHeight)

	if s.EventHandler.OnTransaction != nil {
		s.EventHandler.OnTransaction(tx)
	}
}

// handleMempoolEvent processes mempool transaction messages
func (s *Subscription) handleMempoolEvent(data []byte) {
	tx := &models.TransactionResponse{}
	if err := proto.Unmarshal(data, tx); err != nil {
		if s.EventHandler.OnError != nil {
			s.EventHandler.OnError(fmt.Errorf("unmarshal mempool tx: %w", err))
		}
		return
	}

	// Fetch full transaction data if needed
	if len(tx.Transaction) == 0 && !s.options.LiteMode {
		txData, err := s.client.GetTransaction(s.ctx, tx.Id)
		if err != nil {
			if s.EventHandler.OnError != nil {
				s.EventHandler.OnError(fmt.Errorf("fetch mempool tx %s: %w", tx.Id, err))
			}
			return
		}
		tx.Transaction = txData.Transaction
	}

	if s.EventHandler.OnMempool != nil {
		s.EventHandler.OnMempool(tx)
	}
}

// setupCentrifugeHandlers configures all event handlers for the centrifuge client
func (s *Subscription) setupCentrifugeHandlers() {
	c := s.centrifugeClient

	c.OnConnecting(func(e centrifuge.ConnectingEvent) {
		s.setState(stateConnecting)

		status := "connecting"
		message := "Connecting to server"

		// Check if this is a reconnection
		if s.getState() != stateConnecting || s.position.GetBlock() > uint32(s.FromBlock) {
			status = "reconnecting"
			block, page := s.position.Get()
			message = fmt.Sprintf("Reconnecting to server at block %d, page %d", block, page)
		}

		if s.EventHandler.OnStatus != nil {
			s.EventHandler.OnStatus(&models.ControlResponse{
				StatusCode: uint32(StatusConnecting),
				Status:     status,
				Message:    message,
			})
		}
	})

	c.OnConnected(func(e centrifuge.ConnectedEvent) {
		s.setState(stateConnected)

		// Check if this is a reconnect (we've connected before)
		s.mu.Lock()
		isReconnect := s.hasConnected
		s.hasConnected = true
		s.mu.Unlock()

		// On reconnect, update the main channel to use current position
		if isReconnect && s.EventHandler.OnTransaction != nil && s.mainChannelName != "" {
			if err := s.updateMainChannelPosition(); err != nil {
				log.Printf("Failed to update main channel on reconnect: %v", err)
				if s.EventHandler.OnError != nil {
					s.EventHandler.OnError(fmt.Errorf("reconnect channel update: %w", err))
				}
			}
		}

		if s.EventHandler.OnStatus != nil {
			s.EventHandler.OnStatus(&models.ControlResponse{
				StatusCode: uint32(StatusConnected),
				Status:     "connected",
				Message:    "Connected to server",
			})
		}
	})

	c.OnDisconnected(func(e centrifuge.DisconnectedEvent) {
		// Don't change state if we're closing
		if s.getState() != stateClosed {
			s.setState(stateDisconnected)
		}

		if s.EventHandler.OnStatus != nil {
			s.EventHandler.OnStatus(&models.ControlResponse{
				StatusCode: uint32(StatusDisconnected),
				Status:     "disconnected",
				Message:    "Disconnected from server",
			})
		}
	})

	c.OnError(func(e centrifuge.ErrorEvent) {
		if s.EventHandler.OnStatus != nil {
			s.EventHandler.OnStatus(&models.ControlResponse{
				StatusCode: uint32(StatusError),
				Status:     "error",
				Message:    e.Error.Error(),
			})
		}
	})

	c.OnMessage(func(e centrifuge.MessageEvent) {
		log.Printf("Message from server: %s", string(e.Data))
	})

	c.OnSubscribed(func(e centrifuge.ServerSubscribedEvent) {
		if s.EventHandler.OnStatus != nil {
			s.EventHandler.OnStatus(&models.ControlResponse{
				StatusCode: uint32(StatusSubscribed),
				Status:     "subscribed",
				Message:    "Subscribed to " + e.Channel,
			})
		}
	})

	c.OnSubscribing(func(e centrifuge.ServerSubscribingEvent) {
		s.setState(stateSubscribing)

		if s.EventHandler.OnStatus != nil {
			s.EventHandler.OnStatus(&models.ControlResponse{
				StatusCode: uint32(StatusSubscribing),
				Status:     "subscribing",
				Message:    "Subscribing to " + e.Channel,
			})
		}
	})

	c.OnUnsubscribed(func(e centrifuge.ServerUnsubscribedEvent) {
		if s.EventHandler.OnStatus != nil {
			s.EventHandler.OnStatus(&models.ControlResponse{
				StatusCode: uint32(StatusUnsubscribed),
				Status:     "unsubscribed",
				Message:    "Unsubscribed from " + e.Channel,
			})
		}
	})

	c.OnPublication(func(e centrifuge.ServerPublicationEvent) {
		log.Printf("Publication from server-side channel %s: %s (offset %d)", e.Channel, e.Data, e.Offset)
		if strings.Contains(e.Channel, ":control") {
			s.addToQueue(&pubEvent{Channel: "control", Data: e.Data})
		} else if strings.Contains(e.Channel, ":mempool") {
			s.addToQueue(&pubEvent{Channel: "mempool", Data: e.Data})
		} else {
			s.addToQueue(&pubEvent{Channel: "main", Data: e.Data})
		}
	})

	c.OnJoin(func(e centrifuge.ServerJoinEvent) {
		if s.EventHandler.OnStatus != nil {
			s.EventHandler.OnStatus(&models.ControlResponse{
				StatusCode: uint32(StatusJoin),
				Status:     "join",
				Message:    "Joined " + e.Channel,
			})
		}
	})

	c.OnLeave(func(e centrifuge.ServerLeaveEvent) {
		if s.EventHandler.OnStatus != nil {
			s.EventHandler.OnStatus(&models.ControlResponse{
				StatusCode: uint32(StatusLeave),
				Status:     "leave",
				Message:    "Left " + e.Channel,
			})
		}
	})
}

// getSubType returns the subscription type prefix based on options
func (s *Subscription) getSubType() string {
	if s.options.LiteMode {
		return "lite"
	}
	return "query"
}

// updateMainChannelPosition replaces the main channel subscription with a new one
// using the current position. Called on reconnect to avoid replaying old data.
func (s *Subscription) updateMainChannelPosition() error {
	if s.mainChannelName == "" {
		return nil // No main channel to update
	}

	subType := s.getSubType()
	block, page := s.position.Get()
	newChannelName := fmt.Sprintf("%s:%s:%d:%d", subType, s.SubscriptionID, block, page)

	// If position hasn't changed, no need to update
	if newChannelName == s.mainChannelName {
		return nil
	}

	log.Printf("Updating main channel from %s to %s", s.mainChannelName, newChannelName)

	// Replace the subscription with new position
	sub, err := s.channels.ReplaceSubscription(s.mainChannelName, newChannelName, func(e centrifuge.PublicationEvent) {
		s.addToQueue(&pubEvent{Channel: "main", Data: e.Data})
	})
	if err != nil {
		return err
	}

	// Subscribe to the new channel
	if err := sub.Subscribe(); err != nil {
		return fmt.Errorf("subscribe to new channel: %w", err)
	}

	// Update tracked name
	s.mu.Lock()
	s.mainChannelName = newChannelName
	s.mu.Unlock()

	return nil
}

// setupChannels creates and configures all subscription channels
func (s *Subscription) setupChannels() error {
	subType := s.getSubType()
	block, page := s.position.Get()

	// Control channel
	controlChannel := fmt.Sprintf("%s:%s:control", subType, s.SubscriptionID)
	if _, err := s.channels.CreateSubscription(controlChannel, func(e centrifuge.PublicationEvent) {
		s.addToQueue(&pubEvent{Channel: "control", Data: e.Data})
	}); err != nil {
		return fmt.Errorf("create control channel: %w", err)
	}

	// Main transaction channel (if handler provided)
	if s.EventHandler.OnTransaction != nil {
		mainChannel := fmt.Sprintf("%s:%s:%d:%d", subType, s.SubscriptionID, block, page)
		if _, err := s.channels.CreateSubscription(mainChannel, func(e centrifuge.PublicationEvent) {
			s.addToQueue(&pubEvent{Channel: "main", Data: e.Data})
		}); err != nil {
			return fmt.Errorf("create main channel: %w", err)
		}
		// Track main channel name for reconnect updates
		s.mainChannelName = mainChannel
	}

	// Mempool channel (if handler provided)
	if s.EventHandler.OnMempool != nil {
		mempoolChannel := fmt.Sprintf("%s:%s:mempool", subType, s.SubscriptionID)
		if _, err := s.channels.CreateSubscription(mempoolChannel, func(e centrifuge.PublicationEvent) {
			s.addToQueue(&pubEvent{Channel: "mempool", Data: e.Data})
		}); err != nil {
			return fmt.Errorf("create mempool channel: %w", err)
		}
	}

	return nil
}

// Unsubscribe closes the active subscription on the client
func (jb *Client) Unsubscribe() error {
	jb.mu.Lock()
	sub := jb.subscription
	jb.subscription = nil
	jb.mu.Unlock()

	if sub == nil {
		return nil
	}

	return sub.Unsubscribe()
}

// Subscribe creates a subscription starting from a specific block
func (jb *Client) Subscribe(ctx context.Context, subscriptionID string, fromBlock uint64, eventHandler EventHandler) (*Subscription, error) {
	return jb.SubscribeWithQueue(ctx, subscriptionID, fromBlock, 0, eventHandler, &SubscribeOptions{
		QueueSize: 100000,
	})
}

// SubscribeWithQueue creates a subscription with custom queue options
func (jb *Client) SubscribeWithQueue(ctx context.Context, subscriptionID string, fromBlock uint64, fromPage uint64, eventHandler EventHandler, options *SubscribeOptions) (*Subscription, error) {
	// Default options
	if options == nil {
		options = &SubscribeOptions{QueueSize: 100000}
	}
	if options.QueueSize == 0 {
		options.QueueSize = 100000
	}

	// Create cancellable context
	subCtx, cancel := context.WithCancel(ctx)

	// Get or refresh token
	token := jb.transport.GetToken()
	if token == "" {
		var err error
		if token, err = jb.transport.GetSubscriptionToken(subCtx, subscriptionID); err != nil {
			cancel()
			return nil, fmt.Errorf("get subscription token: %w", err)
		}
		if token != "" {
			jb.transport.SetToken(token)
		}
	}

	// Build WebSocket URL
	protocol := "wss"
	if !jb.transport.IsSSL() {
		protocol = "ws"
	}
	url := fmt.Sprintf("%s://%s/connection/websocket?format=protobuf", protocol, jb.transport.GetServerURL())

	// Create centrifuge client
	centrifugeClient := centrifuge.NewProtobufClient(url, centrifuge.Config{
		Token: token,
		GetToken: func(event centrifuge.ConnectionTokenEvent) (string, error) {
			return jb.transport.RefreshToken(subCtx)
		},
		Name:               "go-junglebus",
		ReadTimeout:        30 * time.Second,
		WriteTimeout:       2 * time.Second,
		HandshakeTimeout:   30 * time.Second,
		MaxServerPingDelay: 30 * time.Second,
		EnableCompression:  true,
	})

	// Create subscription
	sub := &Subscription{
		SubscriptionID:   subscriptionID,
		FromBlock:        fromBlock,
		EventHandler:     eventHandler,
		state:            stateDisconnected,
		client:           jb,
		options:          options,
		position:         newPosition(uint32(fromBlock), fromPage),
		centrifugeClient: centrifugeClient,
		channels:         newChannelManager(centrifugeClient),
		eventQueue:       newEventQueue(options.QueueSize),
		ctx:              subCtx,
		cancel:           cancel,
		done:             make(chan struct{}),
	}

	// Setup event handlers
	sub.setupCentrifugeHandlers()

	// Start event processing goroutine
	go sub.handleEvents()

	// Setup channels
	if err := sub.setupChannels(); err != nil {
		sub.Unsubscribe()
		return nil, err
	}

	// Connect to server
	if err := centrifugeClient.Connect(); err != nil {
		sub.Unsubscribe()
		return nil, fmt.Errorf("connect: %w", err)
	}

	// Subscribe to all channels
	if err := sub.channels.SubscribeAll(); err != nil {
		sub.Unsubscribe()
		return nil, fmt.Errorf("subscribe channels: %w", err)
	}

	sub.setState(stateActive)

	// Store reference on client
	jb.mu.Lock()
	jb.subscription = sub
	jb.mu.Unlock()

	return sub, nil
}
