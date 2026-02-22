package api

import (
	"fmt"
	"testing"
	"time"
)

func TestLogHub_WriteAndSubscribe(t *testing.T) {
	hub := NewLogHub()

	ch, cleanup := hub.Subscribe()
	defer cleanup()

	_, _ = hub.Write([]byte(`{"level":"info","msg":"hello"}`))
	_, _ = hub.Write([]byte(`{"level":"error","msg":"oops"}`))

	select {
	case msg := <-ch:
		if string(msg) != `{"level":"info","msg":"hello"}` {
			t.Errorf("first message: got %q", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first message")
	}

	select {
	case msg := <-ch:
		if string(msg) != `{"level":"error","msg":"oops"}` {
			t.Errorf("second message: got %q", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for second message")
	}
}

func TestLogHub_Buffer(t *testing.T) {
	hub := NewLogHub()

	_, _ = hub.Write([]byte("line 1"))
	_, _ = hub.Write([]byte("line 2"))
	_, _ = hub.Write([]byte("line 3"))

	ch, cleanup := hub.Subscribe()
	defer cleanup()

	var msgs []string
	for i := 0; i < 3; i++ {
		select {
		case msg := <-ch:
			msgs = append(msgs, string(msg))
		case <-time.After(time.Second):
			t.Fatalf("timed out after %d messages", i)
		}
	}

	if msgs[0] != "line 1" {
		t.Errorf("first buffered message: got %q", msgs[0])
	}
	if msgs[2] != "line 3" {
		t.Errorf("last buffered message: got %q", msgs[2])
	}
}

func TestLogHub_BufferOverflow(t *testing.T) {
	hub := NewLogHub()

	for i := 0; i < logBufferSize+50; i++ {
		fmt.Fprintf(hub, "line %d", i)
	}

	ch, cleanup := hub.Subscribe()
	defer cleanup()

	count := 0
	for {
		select {
		case <-ch:
			count++
		case <-time.After(100 * time.Millisecond):
			if count != logBufferSize {
				t.Errorf("expected %d buffered messages, got %d", logBufferSize, count)
			}
			return
		}
	}
}

func TestLogHub_SlowSubscriber(t *testing.T) {
	hub := NewLogHub()

	ch, cleanup := hub.Subscribe()
	defer cleanup()

	// Fill channel beyond capacity; Write should not block.
	for i := 0; i < logBufferSize+logChannelHeadroom+100; i++ {
		fmt.Fprintf(hub, "line %d", i)
	}

	// Drain whatever arrived.
	count := 0
	for {
		select {
		case <-ch:
			count++
		case <-time.After(100 * time.Millisecond):
			if count == 0 {
				t.Error("expected at least some messages")
			}
			if count > logBufferSize+logChannelHeadroom {
				t.Errorf("got %d messages, expected at most %d", count, logBufferSize+logChannelHeadroom)
			}
			return
		}
	}
}
