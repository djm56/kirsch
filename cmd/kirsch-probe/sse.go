package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
)

// SSEEvent represents a single server-sent event.
type SSEEvent struct {
	Event string
	Data  json.RawMessage
}

// ParseSSEStream parses a server-sent event stream and returns the events.
// It accepts "event:" and "data:" with or without a following space (N7).
func ParseSSEStream(data []byte) ([]SSEEvent, error) {
	var events []SSEEvent
	scanner := bufio.NewScanner(bytes.NewReader(data))
	// Set buffer to 8MB maximum (N7).
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)

	var currentEvent string
	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and comments.
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// Parse event field (with or without space) (N7).
		var eventValue string
		if strings.HasPrefix(line, "event:") {
			eventValue = strings.TrimPrefix(line, "event:")
			eventValue = strings.TrimPrefix(eventValue, " ")
			if eventValue != "" {
				currentEvent = eventValue
				continue
			}
		}

		// Parse data field (with or without space) (N7).
		var dataValue string
		if strings.HasPrefix(line, "data:") {
			dataValue = strings.TrimPrefix(line, "data:")
			dataValue = strings.TrimPrefix(dataValue, " ")
			if currentEvent != "" {
				events = append(events, SSEEvent{
					Event: currentEvent,
					Data:  json.RawMessage(dataValue),
				})
				currentEvent = ""
			}
		}
	}

	return events, scanner.Err()
}
