package bus

import (
	"sync"
	"testing"
)

func TestNewBus(t *testing.T) {
	b := New()
	if b == nil {
		t.Fatal("expected non-nil bus")
	}
}

func TestSubscribeAndPublish(t *testing.T) {
	b := New()

	var mu sync.Mutex
	var received []Event

	b.Subscribe(EventUserInput, func(e Event) {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
	})

	b.PublishEvent(EventUserInput, "test payload", "test")

	mu.Lock()
	if len(received) != 1 {
		t.Errorf("expected 1 event, got %d", len(received))
	}
	if received[0].Type != EventUserInput {
		t.Errorf("expected EventUserInput, got %s", received[0].Type)
	}
	if received[0].Source != "test" {
		t.Errorf("expected source 'test', got %s", received[0].Source)
	}
	mu.Unlock()
}

func TestSubscribeMultiple(t *testing.T) {
	b := New()

	count1 := 0
	count2 := 0

	b.Subscribe(EventModeChange, func(e Event) {
		count1++
	})
	b.Subscribe(EventModeChange, func(e Event) {
		count2++
	})

	b.PublishEvent(EventModeChange, "mode1", "test")

	if count1 != 1 {
		t.Errorf("expected handler1 to be called once, got %d", count1)
	}
	if count2 != 1 {
		t.Errorf("expected handler2 to be called once, got %d", count2)
	}
}

func TestUnsubscribe(t *testing.T) {
	b := New()

	count := 0
	unsub := b.Subscribe(EventError, func(e Event) {
		count++
	})

	b.PublishEvent(EventError, "err1", "test")
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	unsub()

	b.PublishEvent(EventError, "err2", "test")
	if count != 1 {
		t.Errorf("expected count still 1 after unsubscribe, got %d", count)
	}
}

func TestMultipleEventTypes(t *testing.T) {
	b := New()

	events := make(map[EventType]int)
	var mu sync.Mutex

	handler := func(e Event) {
		mu.Lock()
		events[e.Type]++
		mu.Unlock()
	}

	b.Subscribe(EventUserInput, handler)
	b.Subscribe(EventToolCall, handler)
	b.Subscribe(EventError, handler)

	b.PublishEvent(EventUserInput, "input", "test")
	b.PublishEvent(EventToolCall, "tool", "test")
	b.PublishEvent(EventUserInput, "input2", "test")
	b.PublishEvent(EventError, "error", "test")

	mu.Lock()
	if events[EventUserInput] != 2 {
		t.Errorf("expected 2 user input events, got %d", events[EventUserInput])
	}
	if events[EventToolCall] != 1 {
		t.Errorf("expected 1 tool call event, got %d", events[EventToolCall])
	}
	if events[EventError] != 1 {
		t.Errorf("expected 1 error event, got %d", events[EventError])
	}
	mu.Unlock()
}

func TestDefaultBus(t *testing.T) {
	count := 0
	unsub := Subscribe(EventSessionNew, func(e Event) {
		count++
	})

	PublishEvent(EventSessionNew, "new session", "test")
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}

	unsub()
	PublishEvent(EventSessionNew, "new session 2", "test")
	if count != 1 {
		t.Errorf("expected still 1 after unsub, got %d", count)
	}
}

func TestPublishEventStruct(t *testing.T) {
	b := New()

	var received Event
	b.Subscribe(EventFileChange, func(e Event) {
		received = e
	})

	event := Event{
		Type:    EventFileChange,
		Payload: map[string]string{"file": "test.go"},
		Source:  "watcher",
	}

	b.Publish(event)

	if received.Type != EventFileChange {
		t.Errorf("expected EventFileChange, got %s", received.Type)
	}
}
