package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"temporal-worker-go/activities"
)

// EventInput represents an incoming event to process.
type EventInput struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
}

// EventResult holds the final output of the event processing pipeline.
type EventResult struct {
	EventID     string `json:"eventId"`
	Status      string `json:"status"`
	ProcessedAt string `json:"processedAt"`
	Steps       []string `json:"steps"`
}

// EventProcessingWorkflow demonstrates a sequential activity chain:
//   validate → process → notify
// Each activity has retry policies and timeouts configured.
func EventProcessingWorkflow(ctx workflow.Context, input EventInput) (*EventResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("EventProcessingWorkflow started", "eventType", input.Type, "eventId", input.ID)

	// Activity options with retry policy
	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	var steps []string

	// Step 1: Validate the event
	var validateResult string
	err := workflow.ExecuteActivity(ctx, activities.ValidateEvent, input.Type, input.ID).Get(ctx, &validateResult)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	steps = append(steps, validateResult)
	logger.Info("Validation complete", "result", validateResult)

	// Step 2: Process the event
	var processResult string
	err = workflow.ExecuteActivity(ctx, activities.ProcessEvent, input.Type, input.ID).Get(ctx, &processResult)
	if err != nil {
		return nil, fmt.Errorf("processing failed: %w", err)
	}
	steps = append(steps, processResult)
	logger.Info("Processing complete", "result", processResult)

	// Step 3: Send notification
	var notifyResult string
	err = workflow.ExecuteActivity(ctx, activities.NotifyEventProcessed, input.Type, input.ID).Get(ctx, &notifyResult)
	if err != nil {
		return nil, fmt.Errorf("notification failed: %w", err)
	}
	steps = append(steps, notifyResult)
	logger.Info("Notification complete", "result", notifyResult)

	result := &EventResult{
		EventID:     input.ID,
		Status:      "completed",
		ProcessedAt: time.Now().UTC().Format(time.RFC3339),
		Steps:       steps,
	}

	logger.Info("EventProcessingWorkflow completed", "eventId", input.ID)
	return result, nil
}
