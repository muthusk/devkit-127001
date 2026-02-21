package activities

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
)

// ValidateEvent checks that the event is well-formed.
func ValidateEvent(ctx context.Context, eventType string, eventID string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("ValidateEvent", "type", eventType, "id", eventID)

	// Simulate validation work
	time.Sleep(500 * time.Millisecond)

	if eventType == "" || eventID == "" {
		return "", fmt.Errorf("invalid event: type and id are required")
	}

	return fmt.Sprintf("validated:%s:%s", eventType, eventID), nil
}

// ProcessEvent performs the main processing logic for the event.
func ProcessEvent(ctx context.Context, eventType string, eventID string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("ProcessEvent", "type", eventType, "id", eventID)

	// Simulate processing work
	time.Sleep(1 * time.Second)

	return fmt.Sprintf("processed:%s:%s", eventType, eventID), nil
}

// NotifyEventProcessed sends a notification that event processing is complete.
func NotifyEventProcessed(ctx context.Context, eventType string, eventID string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("NotifyEventProcessed", "type", eventType, "id", eventID)

	// Simulate sending a notification (email, webhook, etc.)
	time.Sleep(300 * time.Millisecond)

	return fmt.Sprintf("notified:%s:%s", eventType, eventID), nil
}
