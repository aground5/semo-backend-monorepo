package streaming

import (
	"encoding/json"
	"fmt"
)

// EventSender handles sending events to a stream channel
type EventSender struct {
	StreamChan chan<- string
}

// NewEventSender creates a new EventSender
func NewEventSender(streamChan chan<- string) *EventSender {
	return &EventSender{
		StreamChan: streamChan,
	}
}

// Send sends an event to the stream channel
func (s *EventSender) Send(eventType string, data string) {
	s.StreamChan <- fmt.Sprintf("event: %s", eventType)

	// Create a map with the data
	wrapper := map[string]interface{}{
		"event": eventType,
		"v":     data,
	}

	// Marshal to JSON with proper encoding
	jsonBytes, err := json.Marshal(wrapper)
	if err != nil {
		// Handle error
		jsonBytes = []byte(`{"v":"Error encoding JSON"}`)
	}

	s.StreamChan <- fmt.Sprintf("data: %s", string(jsonBytes))
}

// SendEventWithAdditional is a helper function to format and send SSE events with additional data
func (s *EventSender) SendEventWithAdditional(eventType string, data string, additionalData map[string]interface{}) {
	s.StreamChan <- fmt.Sprintf("event: %s", eventType)

	// Create a map with the data
	wrapper := map[string]interface{}{
		"event": eventType,
		"v":     data,
	}

	// Merge additionalData into wrapper
	for key, value := range additionalData {
		wrapper[key] = value
	}

	// Marshal to JSON with proper encoding
	jsonBytes, err := json.Marshal(wrapper)
	if err != nil {
		// Handle error
		jsonBytes = []byte(`{"v":"Error encoding JSON"}`)
	}

	s.StreamChan <- fmt.Sprintf("data: %s", string(jsonBytes))
}

// SendCrash sends a crash event to the stream channel
func (s *EventSender) SendCrash() {
	s.StreamChan <- "event: fail"
	s.StreamChan <- fmt.Sprintf("data: [CRASH]")
}

// SendComplete sends a completion event to the stream channel
func (s *EventSender) SendComplete(message string) {
	s.Send("complete", message)
}

// SendError sends an error event to the stream channel
func (s *EventSender) SendError(err error) {
	s.Send("error", err.Error())
}
