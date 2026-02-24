package signals

import "sync"

// Signal[T] is a reactive value that notifies subscribers when changed.
// No build tags — fully testable outside WASM.
type Signal[T any] struct {
	mu    sync.RWMutex
	value T
	subs  []func()
}

// NewSignal creates a Signal with an initial value.
func NewSignal[T any](initial T) *Signal[T] {
	return &Signal[T]{value: initial}
}

// Get returns the current value.
func (s *Signal[T]) Get() T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.value
}

// Set updates the value and notifies all subscribers.
func (s *Signal[T]) Set(v T) {
	s.mu.Lock()
	s.value = v
	subs := make([]func(), len(s.subs))
	copy(subs, s.subs)
	s.mu.Unlock()

	for _, fn := range subs {
		fn()
	}
}

// Subscribe registers a callback fired when the value changes.
// Returns an unsubscribe func — call it in OnUnmount to avoid memory leaks.
func (s *Signal[T]) Subscribe(fn func()) (unsubscribe func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := len(s.subs)
	s.subs = append(s.subs, fn)

	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.subs = append(s.subs[:idx], s.subs[idx+1:]...)
	}
}
