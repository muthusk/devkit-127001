package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/workflow"
)

// DummyWaitResult holds the output from the wait workflow.
type DummyWaitResult struct {
	RequestedDuration string `json:"requestedDuration"`
	Status            string `json:"status"`
	StartedAt         string `json:"startedAt"`
	CompletedAt       string `json:"completedAt"`
}

// DummyWaitWorkflow does nothing but sleep for a specified duration.
// Demonstrates: timers, cancellation, workflow visibility in the UI while running.
//
// Input: a duration string like "30s", "2m", "5m".
func DummyWaitWorkflow(ctx workflow.Context, durationStr string) (*DummyWaitResult, error) {
	logger := workflow.GetLogger(ctx)

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return nil, fmt.Errorf("invalid duration %q: %w", durationStr, err)
	}

	startedAt := workflow.Now(ctx).UTC().Format(time.RFC3339)
	logger.Info("DummyWaitWorkflow started", "duration", durationStr)

	// This is the core of this workflow — just a timer.
	// While sleeping, the workflow shows as "Running" in the Temporal UI.
	// You can cancel it via the UI or CLI to test cancellation behavior.
	err = workflow.Sleep(ctx, duration)
	if err != nil {
		// This happens if the workflow is cancelled during sleep.
		logger.Info("DummyWaitWorkflow was cancelled during sleep", "duration", durationStr)
		return &DummyWaitResult{
			RequestedDuration: durationStr,
			Status:            "cancelled",
			StartedAt:         startedAt,
			CompletedAt:       workflow.Now(ctx).UTC().Format(time.RFC3339),
		}, err
	}

	completedAt := workflow.Now(ctx).UTC().Format(time.RFC3339)
	logger.Info("DummyWaitWorkflow completed", "duration", durationStr)

	return &DummyWaitResult{
		RequestedDuration: durationStr,
		Status:            "completed",
		StartedAt:         startedAt,
		CompletedAt:       completedAt,
	}, nil
}
