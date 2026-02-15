package api

import "sync"

// LogHub is a fan-out writer that distributes log output to SSE subscribers.
// It implements io.Writer so it plugs into zerolog.MultiLevelWriter.
type LogHub struct {
	mu          sync.RWMutex
	subscribers map[chan []byte]struct{}
}

// NewLogHub creates a new LogHub.
func NewLogHub() *LogHub {
	return &LogHub{
		subscribers: make(map[chan []byte]struct{}),
	}
}

// Write sends a copy of p to every subscriber. It never returns an error
// so it won't break the zerolog writer chain.
func (h *LogHub) Write(p []byte) (int, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for ch := range h.subscribers {
		msg := make([]byte, len(p))
		copy(msg, p)
		select {
		case ch <- msg:
		default:
			// Drop if subscriber is slow.
		}
	}
	return len(p), nil
}

// Subscribe returns a channel that receives log messages and a cleanup
// function that must be called when the subscriber is done.
func (h *LogHub) Subscribe() (<-chan []byte, func()) {
	ch := make(chan []byte, 64)

	h.mu.Lock()
	h.subscribers[ch] = struct{}{}
	h.mu.Unlock()

	cleanup := func() {
		h.mu.Lock()
		delete(h.subscribers, ch)
		h.mu.Unlock()
	}

	return ch, cleanup
}
