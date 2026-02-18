package api

import (
	"testing"
	"time"
)

func TestProcessOutputHub_WriteAndSubscribe(t *testing.T) {
	hub := NewProcessOutputHub()

	ch, cleanup := hub.Subscribe("myapp", "web")
	defer cleanup()

	hub.Write("myapp", "web", "stdout", "hello world")
	hub.Write("myapp", "web", "stderr", "error msg")
	hub.Write("myapp", "api", "stdout", "should not appear")

	select {
	case line := <-ch:
		if line.Line != "hello world" || line.Source != "stdout" {
			t.Errorf("first line: got %+v", line)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first line")
	}

	select {
	case line := <-ch:
		if line.Line != "error msg" || line.Source != "stderr" {
			t.Errorf("second line: got %+v", line)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for second line")
	}

	// The third write was for a different service; channel should be empty.
	select {
	case line := <-ch:
		t.Errorf("unexpected line: %+v", line)
	case <-time.After(100 * time.Millisecond):
		// expected
	}
}

func TestProcessOutputHub_Buffer(t *testing.T) {
	hub := NewProcessOutputHub()

	hub.Write("myapp", "web", "stdout", "line 1")
	hub.Write("myapp", "web", "stdout", "line 2")
	hub.Write("myapp", "web", "stdout", "line 3")

	// New subscriber should get buffered history.
	ch, cleanup := hub.Subscribe("myapp", "web")
	defer cleanup()

	var lines []OutputLine
	for i := 0; i < 3; i++ {
		select {
		case line := <-ch:
			lines = append(lines, line)
		case <-time.After(time.Second):
			t.Fatalf("timed out after %d lines", i)
		}
	}

	if lines[0].Line != "line 1" {
		t.Errorf("first buffered line: got %q", lines[0].Line)
	}
	if lines[2].Line != "line 3" {
		t.Errorf("last buffered line: got %q", lines[2].Line)
	}
}

func TestProcessOutputHub_BufferOverflow(t *testing.T) {
	hub := NewProcessOutputHub()

	// Write more than buffer size.
	for i := 0; i < outputBufferSize+50; i++ {
		hub.Write("myapp", "web", "stdout", "line")
	}

	ch, cleanup := hub.Subscribe("myapp", "web")
	defer cleanup()

	count := 0
	for {
		select {
		case <-ch:
			count++
		case <-time.After(100 * time.Millisecond):
			if count != outputBufferSize {
				t.Errorf("expected %d buffered lines, got %d", outputBufferSize, count)
			}
			return
		}
	}
}
