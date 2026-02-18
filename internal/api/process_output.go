package api

import (
	"sync"
	"time"
)

const outputBufferSize = 200

// OutputLine represents a single line of process output.
type OutputLine struct {
	Project string `json:"project"`
	Service string `json:"service"`
	Source  string `json:"source"`
	Line    string `json:"line"`
	Time    string `json:"time"`
}

type outputSubscriber struct {
	ch      chan OutputLine
	project string
	service string
}

// ProcessOutputHub fans out process output to SSE subscribers with per-process
// ring buffers so new subscribers get recent history.
type ProcessOutputHub struct {
	mu          sync.Mutex
	buffers     map[string][]OutputLine // key: "project/service"
	subscribers map[*outputSubscriber]struct{}
}

// NewProcessOutputHub creates a new ProcessOutputHub.
func NewProcessOutputHub() *ProcessOutputHub {
	return &ProcessOutputHub{
		buffers:     make(map[string][]OutputLine),
		subscribers: make(map[*outputSubscriber]struct{}),
	}
}

func bufferKey(project, service string) string {
	return project + "\x00" + service
}

// Write appends a line to the ring buffer and fans it out to matching subscribers.
func (h *ProcessOutputHub) Write(project, service, source, line string) {
	ol := OutputLine{
		Project: project,
		Service: service,
		Source:  source,
		Line:    line,
		Time:    time.Now().UTC().Format(time.RFC3339Nano),
	}

	key := bufferKey(project, service)

	h.mu.Lock()
	defer h.mu.Unlock()

	buf := h.buffers[key]
	buf = append(buf, ol)
	if len(buf) > outputBufferSize {
		buf = buf[len(buf)-outputBufferSize:]
	}
	h.buffers[key] = buf

	for sub := range h.subscribers {
		if sub.project == project && sub.service == service {
			select {
			case sub.ch <- ol:
			default:
			}
		}
	}
}

// Subscribe returns a channel that receives output lines for the given
// project/service, pre-filled with buffered history. The returned cleanup
// function must be called when the subscriber is done.
func (h *ProcessOutputHub) Subscribe(project, service string) (<-chan OutputLine, func()) {
	ch := make(chan OutputLine, outputBufferSize+64)

	h.mu.Lock()

	// Send buffered history.
	key := bufferKey(project, service)
	for _, ol := range h.buffers[key] {
		ch <- ol
	}

	sub := &outputSubscriber{ch: ch, project: project, service: service}
	h.subscribers[sub] = struct{}{}
	h.mu.Unlock()

	cleanup := func() {
		h.mu.Lock()
		delete(h.subscribers, sub)
		h.mu.Unlock()
	}

	return ch, cleanup
}
