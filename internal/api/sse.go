package api

import "sync"

const logBufferSize = 200

// logChannelHeadroom is extra channel capacity beyond logBufferSize so that
// Subscribe can replay the full buffer without blocking while concurrent
// writes land in the headroom slots.
const logChannelHeadroom = 64

// LogHub is a fan-out writer that distributes log output to SSE subscribers.
// It implements io.Writer so it plugs into the zerolog writer chain. A ring
// buffer of recent entries is kept so new subscribers receive history.
type LogHub struct {
	mu          sync.Mutex
	subscribers map[chan []byte]struct{}
	buffer      [][]byte
}

// NewLogHub creates a new LogHub.
func NewLogHub() *LogHub {
	return &LogHub{
		subscribers: make(map[chan []byte]struct{}),
	}
}

// Write sends a copy of p to every subscriber and appends it to the ring
// buffer. It never returns an error so it won't break the writer chain.
func (h *LogHub) Write(p []byte) (int, error) {
	msg := make([]byte, len(p))
	copy(msg, p)

	h.mu.Lock()
	defer h.mu.Unlock()

	h.buffer = append(h.buffer, msg)
	if len(h.buffer) > logBufferSize {
		trimmed := make([][]byte, logBufferSize)
		copy(trimmed, h.buffer[len(h.buffer)-logBufferSize:])
		h.buffer = trimmed
	}

	for ch := range h.subscribers {
		select {
		case ch <- msg:
		default:
			// Drop if subscriber is slow.
		}
	}
	return len(p), nil
}

// Subscribe returns a channel that receives log messages (pre-filled with
// buffered history) and a cleanup function that must be called when the
// subscriber is done.
func (h *LogHub) Subscribe() (<-chan []byte, func()) {
	// Channel capacity: logBufferSize for the history replay plus
	// logChannelHeadroom for concurrent writes during replay. The buffer
	// holds at most logBufferSize entries, so the history sends never block.
	ch := make(chan []byte, logBufferSize+logChannelHeadroom)

	h.mu.Lock()

	for _, msg := range h.buffer {
		ch <- msg
	}

	h.subscribers[ch] = struct{}{}
	h.mu.Unlock()

	cleanup := func() {
		h.mu.Lock()
		delete(h.subscribers, ch)
		h.mu.Unlock()
	}

	return ch, cleanup
}
