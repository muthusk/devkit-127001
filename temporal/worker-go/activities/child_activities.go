package activities

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
)

// ProcessItem simulates processing a single item in a fanout.
func ProcessItem(ctx context.Context, itemID string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("ProcessItem", "itemId", itemID)

	// Simulate variable processing time (1-3 seconds)
	info := activity.GetInfo(ctx)
	processingTime := time.Duration(1000+info.Attempt*500) * time.Millisecond
	time.Sleep(processingTime)

	logger.Info(fmt.Sprintf("✅ Item processed: %s", itemID))
	return fmt.Sprintf("processed:%s", itemID), nil
}
