package junglebus

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/b-open-io/go-junglebus/models"
	"github.com/centrifugal/centrifuge-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func newTestEventHandler() (EventHandler, *bool, *bool, *bool) {
	onTransactionCalled := false
	onMempoolCalled := false
	onStatusCalled := false

	return EventHandler{
		OnTransaction: func(_ *models.TransactionResponse) {
			onTransactionCalled = true
		},
		OnMempool: func(_ *models.TransactionResponse) {
			onMempoolCalled = true
		},
		OnStatus: func(_ *models.ControlResponse) {
			onStatusCalled = true
		},
		OnError: func(_ error) {
			// Error handler
		},
	}, &onTransactionCalled, &onMempoolCalled, &onStatusCalled
}

func TestSubscription_HandlePubChan(t *testing.T) {
	t.Run("handle control message", func(t *testing.T) {
		handler, _, _, onStatusCalled := newTestEventHandler()
		sub := &Subscription{
			EventHandler: handler,
			pubChan:      make(chan *pubEvent),
		}

		// Create a control message
		status := &models.ControlResponse{
			StatusCode: uint32(SubscriptionBlockDone),
			Block:      100,
		}
		statusData, err := proto.Marshal(status)
		require.NoError(t, err)

		// Start handling messages
		go sub.handlePubChan(&SubscribeOptions{})

		// Send a control message
		sub.addToQueue(&pubEvent{
			Channel: "control",
			Data:    statusData,
		})

		// Close the channel to stop the handler
		close(sub.pubChan)
		sub.wg.Wait()

		assert.True(t, *onStatusCalled)
	})

	t.Run("handle transaction message", func(t *testing.T) {
		handler, onTransactionCalled, _, _ := newTestEventHandler()
		client, err := New()
		require.NoError(t, err)

		sub := &Subscription{
			EventHandler: handler,
			client:       client,
			pubChan:      make(chan *pubEvent),
		}

		// Create a transaction message
		tx := &models.TransactionResponse{
			Id:          "test-tx",
			BlockHeight: 100,
			Transaction: []byte("test-tx-data"),
		}
		txData, err := proto.Marshal(tx)
		require.NoError(t, err)

		// Start handling messages
		go sub.handlePubChan(&SubscribeOptions{})

		// Send a transaction message
		sub.addToQueue(&pubEvent{
			Channel: "main",
			Data:    txData,
		})

		// Close the channel to stop the handler
		close(sub.pubChan)
		sub.wg.Wait()

		assert.True(t, *onTransactionCalled)
	})

	t.Run("handle mempool message", func(t *testing.T) {
		handler, _, onMempoolCalled, _ := newTestEventHandler()
		client, err := New()
		require.NoError(t, err)

		sub := &Subscription{
			EventHandler: handler,
			client:       client,
			pubChan:      make(chan *pubEvent),
		}

		// Create a mempool message
		tx := &models.TransactionResponse{
			Id:          "test-tx",
			BlockHeight: 0,
			Transaction: []byte("test-tx-data"),
		}
		txData, err := proto.Marshal(tx)
		require.NoError(t, err)

		// Start handling messages
		go sub.handlePubChan(&SubscribeOptions{})

		// Send a mempool message
		sub.addToQueue(&pubEvent{
			Channel: "mempool",
			Data:    txData,
		})

		// Close the channel to stop the handler
		close(sub.pubChan)
		sub.wg.Wait()

		assert.True(t, *onMempoolCalled)
	})
}

//nolint:unused // Mock client and methods for future test cases
type mockCentrifugeClient struct {
	onConnectingCalled   bool
	onConnectedCalled    bool
	onDisconnectedCalled bool
	onErrorCalled        bool
	onMessageCalled      bool
	onSubscribedCalled   bool
	onSubscribingCalled  bool
	onUnsubscribedCalled bool
	onPublicationCalled  bool
	onJoinCalled         bool
	onLeaveCalled        bool
}

func newMockCentrifugeClient() *mockCentrifugeClient {
	return &mockCentrifugeClient{}
}

func (m *mockCentrifugeClient) OnConnecting(handler centrifuge.ConnectingHandler) {
	m.onConnectingCalled = true
	handler(centrifuge.ConnectingEvent{})
}

func (m *mockCentrifugeClient) OnConnected(handler centrifuge.ConnectedHandler) {
	m.onConnectedCalled = true
	handler(centrifuge.ConnectedEvent{})
}

func (m *mockCentrifugeClient) OnDisconnected(handler centrifuge.DisconnectHandler) {
	m.onDisconnectedCalled = true
	handler(centrifuge.DisconnectedEvent{})
}

func (m *mockCentrifugeClient) OnError(handler centrifuge.ErrorHandler) {
	m.onErrorCalled = true
	handler(centrifuge.ErrorEvent{})
}

func (m *mockCentrifugeClient) OnMessage(handler centrifuge.MessageHandler) {
	m.onMessageCalled = true
	handler(centrifuge.MessageEvent{})
}

func (m *mockCentrifugeClient) OnSubscribed(handler centrifuge.ServerSubscribedHandler) {
	m.onSubscribedCalled = true
	handler(centrifuge.ServerSubscribedEvent{})
}

func (m *mockCentrifugeClient) OnSubscribing(handler centrifuge.ServerSubscribingHandler) {
	m.onSubscribingCalled = true
	handler(centrifuge.ServerSubscribingEvent{})
}

func (m *mockCentrifugeClient) OnUnsubscribed(handler centrifuge.ServerUnsubscribedHandler) {
	m.onUnsubscribedCalled = true
	handler(centrifuge.ServerUnsubscribedEvent{})
}

func (m *mockCentrifugeClient) OnPublication(handler centrifuge.ServerPublicationHandler) {
	m.onPublicationCalled = true
	handler(centrifuge.ServerPublicationEvent{})
}

func (m *mockCentrifugeClient) OnJoin(handler centrifuge.ServerJoinHandler) {
	m.onJoinCalled = true
	handler(centrifuge.ServerJoinEvent{})
}

func (m *mockCentrifugeClient) OnLeave(handler centrifuge.ServerLeaveHandler) {
	m.onLeaveCalled = true
	handler(centrifuge.ServerLeaveEvent{})
}

func (m *mockCentrifugeClient) NewSubscription(channel string, _ centrifuge.SubscriptionConfig) (*centrifuge.Subscription, error) {
	if channel == "" {
		return nil, fmt.Errorf("channel cannot be empty")
	}
	return &centrifuge.Subscription{}, nil
}

func (m *mockCentrifugeClient) Connect() error { return nil }
func (m *mockCentrifugeClient) Close()         {}

func TestSubscribe(t *testing.T) {
	// Test with nil context
	client, err := New()
	require.NoError(t, err)

	_, err = client.Subscribe(context.TODO(), "", 0, EventHandler{})
	require.Error(t, err)

	// Test with empty subscription ID
	_, err = client.Subscribe(context.Background(), "", 0, EventHandler{})
	require.Error(t, err)

	// Test with nil context
	_, err = client.Subscribe(context.TODO(), "test-sub", 0, EventHandler{})
	require.Error(t, err)
}

func TestSubscribeWithQueue(t *testing.T) {
	// Test with nil context
	client, err := New()
	require.NoError(t, err)

	_, err = client.SubscribeWithQueue(context.TODO(), "", 0, 0, EventHandler{}, &SubscribeOptions{})
	require.Error(t, err)

	// Test with empty subscription ID
	_, err = client.SubscribeWithQueue(context.Background(), "", 0, 0, EventHandler{}, &SubscribeOptions{})
	require.Error(t, err)

	// Test with nil options
	_, err = client.SubscribeWithQueue(context.Background(), "test-sub", 0, 0, EventHandler{}, nil)
	require.Error(t, err)
}

func TestStartSubscription(t *testing.T) {
	// Create a subscription directly
	sub := &Subscription{
		SubscriptionID: "test-sub",
		subscriptions:  make(map[string]*centrifuge.Subscription),
		pubChan:        make(chan *pubEvent),
	}

	// Test start subscription with empty channel
	centrifugeSub, err := sub.startSubscription("")
	require.Error(t, err)
	assert.Nil(t, centrifugeSub)
	assert.Equal(t, "subscription channel cannot be empty", err.Error())
}

func TestAddToQueue(t *testing.T) {
	// Create a subscription directly
	sub := &Subscription{
		SubscriptionID: "test-sub",
		subscriptions:  make(map[string]*centrifuge.Subscription),
		pubChan:        make(chan *pubEvent, 1), // Buffer size of 1 to avoid blocking
	}

	// Create a test event
	event := &pubEvent{
		Channel: "test-channel",
		Data:    []byte("test-data"),
	}

	// Start a goroutine to handle messages
	go func() {
		e := <-sub.pubChan
		assert.Equal(t, event.Channel, e.Channel)
		assert.Equal(t, event.Data, e.Data)
		sub.wg.Done()
	}()

	// Add event to queue
	sub.addToQueue(event)

	// Wait for message to be processed
	sub.wg.Wait()
}

func TestSetDebug(t *testing.T) {
	client, err := New()
	require.NoError(t, err)

	// Test setting debug to true
	client.SetDebug(true)
	assert.True(t, client.IsDebug())

	// Test setting debug to false
	client.SetDebug(false)
	assert.False(t, client.IsDebug())
}

func TestTransportGetVersion(t *testing.T) {
	// Create client with version
	client, err := New(WithVersion("v2"))
	require.NoError(t, err)

	// Get version from transport
	version := client.transport.GetVersion()
	assert.Equal(t, "v2", version)
}

func TestTransportClientOptions(t *testing.T) {
	// Test WithHTTPClient
	testClient := &http.Client{}
	client, err := New(WithHTTPClient("test-url", testClient))
	require.NoError(t, err)
	assert.Equal(t, "test-url", client.transport.GetServerURL())

	// Test WithToken
	client, err = New(WithToken("test-token"))
	require.NoError(t, err)
	assert.Equal(t, "test-token", client.transport.GetToken())

	// Test WithSSL
	client, err = New(WithSSL(true))
	require.NoError(t, err)
	assert.True(t, client.transport.IsSSL())

	client, err = New(WithSSL(false))
	require.NoError(t, err)
	assert.False(t, client.transport.IsSSL())

	// Test WithVersion
	client, err = New(WithVersion("v2"))
	require.NoError(t, err)
	assert.Equal(t, "v2", client.transport.GetVersion())
}
