package client

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/ag/ai-agent-builder/internal/schema"
)

// ParseNDJSON reads all lines from reader and parses each as a BuildEvent.
// Empty lines are skipped. Non-JSON lines are treated as status strings.
func ParseNDJSON(reader io.Reader) ([]schema.BuildEvent, error) {
	var events []schema.BuildEvent
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if e := parseLine(line); e != nil {
			events = append(events, *e)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

// StreamNDJSON reads lines from reader and sends each parsed BuildEvent to
// eventCh as it arrives. Closes eventCh when done. Handles context cancellation.
func StreamNDJSON(ctx context.Context, reader io.Reader, eventCh chan<- schema.BuildEvent) error {
	defer close(eventCh)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		line := scanner.Text()
		if e := parseLine(line); e != nil {
			select {
			case eventCh <- *e:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return scanner.Err()
}

// parseLine attempts to parse a line as JSON into a BuildEvent.
// If it fails, returns a BuildEvent with the line as the Message field.
// Returns nil for empty lines.
func parseLine(line string) *schema.BuildEvent {
	if strings.TrimSpace(line) == "" {
		return nil
	}
	var e schema.BuildEvent
	if json.Unmarshal([]byte(line), &e) == nil {
		return &e
	}
	return &schema.BuildEvent{Message: line}
}
