package bus

import (
	"sync"
)

type EventType string

const (
	EventUserInput     EventType = "user_input"
	EventAssistantResp EventType = "assistant_response"
	EventToolCall      EventType = "tool_call"
	EventToolResult    EventType = "tool_result"
	EventError         EventType = "error"
	EventModeChange    EventType = "mode_change"
	EventSessionNew    EventType = "session_new"
	EventSessionLoad   EventType = "session_load"
	EventModelChange   EventType = "model_change"
	EventPluginLoad    EventType = "plugin_load"
	EventPluginUnload  EventType = "plugin_unload"
	EventSkillActivate EventType = "skill_activate"
	EventConfigChange  EventType = "config_change"
	EventFileChange    EventType = "file_change"
	EventLSPDiagnostic EventType = "lsp_diagnostic"
)

type Event struct {
	Type    EventType
	Payload interface{}
	Source  string
}

type Handler func(Event)

type Bus struct {
	mu       sync.RWMutex
	handlers map[EventType][]Handler
}

var DefaultBus = New()

func New() *Bus {
	return &Bus{
		handlers: make(map[EventType][]Handler),
	}
}

func (b *Bus) Subscribe(eventType EventType, handler Handler) func() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[eventType] = append(b.handlers[eventType], handler)

	return func() {
		b.Unsubscribe(eventType, handler)
	}
}

func (b *Bus) Unsubscribe(eventType EventType, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	handlers := b.handlers[eventType]
	for i, h := range handlers {
		if &h == &handler {
			b.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			return
		}
	}
}

func (b *Bus) Publish(event Event) {
	b.mu.RLock()
	handlers := make([]Handler, len(b.handlers[event.Type]))
	copy(handlers, b.handlers[event.Type])
	b.mu.RUnlock()

	for _, handler := range handlers {
		handler(event)
	}
}

func (b *Bus) PublishEvent(eventType EventType, payload interface{}, source string) {
	b.Publish(Event{
		Type:    eventType,
		Payload: payload,
		Source:  source,
	})
}

func Subscribe(eventType EventType, handler Handler) func() {
	return DefaultBus.Subscribe(eventType, handler)
}

func Publish(event Event) {
	DefaultBus.Publish(event)
}

func PublishEvent(eventType EventType, payload interface{}, source string) {
	DefaultBus.PublishEvent(eventType, payload, source)
}
