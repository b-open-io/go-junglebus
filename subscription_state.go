package junglebus

import "sync"

// subscriptionState represents the current state of a subscription connection
type subscriptionState int

const (
	stateDisconnected subscriptionState = iota
	stateConnecting
	stateConnected
	stateSubscribing
	stateActive
	stateClosed
)

// String returns a human-readable representation of the state
func (s subscriptionState) String() string {
	switch s {
	case stateDisconnected:
		return "disconnected"
	case stateConnecting:
		return "connecting"
	case stateConnected:
		return "connected"
	case stateSubscribing:
		return "subscribing"
	case stateActive:
		return "active"
	case stateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// subscriptionPosition tracks the current block and page position for a subscription.
// This is per-subscription state, replacing the previous global variables.
type subscriptionPosition struct {
	mu    sync.RWMutex
	block uint32
	page  uint64
}

// newPosition creates a new position tracker with initial values
func newPosition(block uint32, page uint64) *subscriptionPosition {
	return &subscriptionPosition{
		block: block,
		page:  page,
	}
}

// Get returns the current block and page
func (p *subscriptionPosition) Get() (block uint32, page uint64) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.block, p.page
}

// GetBlock returns just the current block
func (p *subscriptionPosition) GetBlock() uint32 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.block
}

// GetPage returns just the current page
func (p *subscriptionPosition) GetPage() uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.page
}

// Update sets both block and page
func (p *subscriptionPosition) Update(block uint32, page uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.block = block
	p.page = page
}

// AdvanceBlock moves to a new block and resets page to 0
func (p *subscriptionPosition) AdvanceBlock(block uint32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.block = block
	p.page = 0
}

// AdvancePage updates both block and page (for SubscriptionPageDone)
func (p *subscriptionPosition) AdvancePage(block uint32, page uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.block = block
	p.page = page
}

// SetBlock updates only the block (for transaction events)
func (p *subscriptionPosition) SetBlock(block uint32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.block = block
}
