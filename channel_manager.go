package junglebus

import (
	"errors"
	"fmt"
	"sync"

	"github.com/centrifugal/centrifuge-go"
)

// channelManager handles centrifuge channel subscriptions for a subscription.
// It provides thread-safe management of multiple channels (control, main, mempool).
type channelManager struct {
	mu       sync.RWMutex
	client   *centrifuge.Client
	channels map[string]*centrifuge.Subscription
}

// newChannelManager creates a new channel manager for the given centrifuge client
func newChannelManager(client *centrifuge.Client) *channelManager {
	return &channelManager{
		client:   client,
		channels: make(map[string]*centrifuge.Subscription),
	}
}

// CreateSubscription creates a new channel subscription without subscribing.
// The handler is called for each publication received on the channel.
func (m *channelManager) CreateSubscription(name string, handler func(e centrifuge.PublicationEvent)) (*centrifuge.Subscription, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sub, err := m.client.NewSubscription(name, centrifuge.SubscriptionConfig{
		Recoverable: true,
	})
	if err != nil {
		return nil, fmt.Errorf("create subscription %s: %w", name, err)
	}

	if handler != nil {
		sub.OnPublication(handler)
	}

	m.channels[name] = sub
	return sub, nil
}

// Subscribe subscribes to a previously created channel
func (m *channelManager) Subscribe(name string) error {
	m.mu.RLock()
	sub, exists := m.channels[name]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("channel %s not found", name)
	}

	return sub.Subscribe()
}

// SubscribeAll subscribes to all created channels
func (m *channelManager) SubscribeAll() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var errs []error
	for name, sub := range m.channels {
		if err := sub.Subscribe(); err != nil {
			errs = append(errs, fmt.Errorf("subscribe %s: %w", name, err))
		}
	}

	return errors.Join(errs...)
}

// Unsubscribe unsubscribes from a specific channel
func (m *channelManager) Unsubscribe(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sub, exists := m.channels[name]
	if !exists {
		return nil // Already unsubscribed or never existed
	}

	err := sub.Unsubscribe()
	delete(m.channels, name)
	return err
}

// UnsubscribeAll unsubscribes from all channels, collecting all errors.
// This is safe to call multiple times.
func (m *channelManager) UnsubscribeAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.channels) == 0 {
		return nil
	}

	var errs []error
	for name, sub := range m.channels {
		if err := sub.Unsubscribe(); err != nil {
			errs = append(errs, fmt.Errorf("unsubscribe %s: %w", name, err))
		}
		delete(m.channels, name)
	}

	return errors.Join(errs...)
}

// Get returns a channel subscription by name
func (m *channelManager) Get(name string) *centrifuge.Subscription {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.channels[name]
}

// Has checks if a channel subscription exists
func (m *channelManager) Has(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.channels[name]
	return exists
}

// Count returns the number of active channel subscriptions
func (m *channelManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.channels)
}

// Names returns the names of all channel subscriptions
func (m *channelManager) Names() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.channels))
	for name := range m.channels {
		names = append(names, name)
	}
	return names
}

// ReplaceSubscription unsubscribes from oldName and creates a new subscription with newName.
// This is used to update the main channel position on reconnect.
func (m *channelManager) ReplaceSubscription(oldName, newName string, handler func(e centrifuge.PublicationEvent)) (*centrifuge.Subscription, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Unsubscribe from old channel if it exists
	if oldSub, exists := m.channels[oldName]; exists {
		oldSub.Unsubscribe()
		delete(m.channels, oldName)
	}

	// Create new subscription
	sub, err := m.client.NewSubscription(newName, centrifuge.SubscriptionConfig{
		Recoverable: false, // Don't recover on reconnect - we handle position ourselves
	})
	if err != nil {
		return nil, fmt.Errorf("create subscription %s: %w", newName, err)
	}

	if handler != nil {
		sub.OnPublication(handler)
	}

	m.channels[newName] = sub
	return sub, nil
}
