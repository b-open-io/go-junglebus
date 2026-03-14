package junglebus

import (
	"sync"
	"sync/atomic"
)

// eventQueue provides a thread-safe buffered channel for subscription events.
// It handles the race condition between sending events and closing the queue.
type eventQueue struct {
	ch     chan *pubEvent
	closed atomic.Bool
	wg     sync.WaitGroup
}

// newEventQueue creates a new event queue with the specified buffer size
func newEventQueue(size uint32) *eventQueue {
	if size == 0 {
		size = 100000 // Default size
	}
	return &eventQueue{
		ch: make(chan *pubEvent, size),
	}
}

// Send adds an event to the queue.
// Returns false if the queue is closed or full (non-blocking).
// The caller should check the return value and handle accordingly.
func (q *eventQueue) Send(event *pubEvent) bool {
	// Atomic check prevents the race condition where we check closed,
	// then the channel closes, then we try to send
	if q.closed.Load() {
		return false
	}

	// Use select with default for non-blocking send
	// This prevents blocking when the queue is full
	q.wg.Add(1)
	select {
	case q.ch <- event:
		return true
	default:
		// Queue is full - undo the wg.Add
		q.wg.Done()
		return false
	}
}

// SendBlocking adds an event to the queue, blocking until space is available.
// Returns false only if the queue is closed.
func (q *eventQueue) SendBlocking(event *pubEvent) bool {
	if q.closed.Load() {
		return false
	}

	q.wg.Add(1)

	// We need to handle the case where the queue closes while we're waiting
	select {
	case q.ch <- event:
		return true
	default:
		// Channel might be closed or full, try again with closed check
		if q.closed.Load() {
			q.wg.Done()
			return false
		}
		// Actually block this time
		q.ch <- event
		return true
	}
}

// Close signals that no more events will be sent.
// It's safe to call multiple times.
func (q *eventQueue) Close() {
	// Swap returns the previous value - if it was already true, we already closed
	if q.closed.Swap(true) {
		return
	}
	close(q.ch)
}

// Done marks an event as processed.
// Must be called once for each successfully sent event after processing.
func (q *eventQueue) Done() {
	q.wg.Done()
}

// Wait blocks until all sent events have been processed (Done called).
func (q *eventQueue) Wait() {
	q.wg.Wait()
}

// Channel returns the underlying channel for range iteration.
// The consumer goroutine should use: for event := range q.Channel()
func (q *eventQueue) Channel() <-chan *pubEvent {
	return q.ch
}

// IsClosed returns whether the queue has been closed.
func (q *eventQueue) IsClosed() bool {
	return q.closed.Load()
}

// Len returns the current number of events in the queue.
func (q *eventQueue) Len() int {
	return len(q.ch)
}
