package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"temporal-worker-go/activities"
)

// ChildFanoutInput contains items to process in parallel via child workflows.
type ChildFanoutInput struct {
	Items []string `json:"items"`
}

// ChildItemResult holds the result of processing a single item.
type ChildItemResult struct {
	ItemID      string `json:"itemId"`
	Status      string `json:"status"`
	ProcessedBy string `json:"processedBy"`
}

// ChildFanoutResult aggregates all child results.
type ChildFanoutResult struct {
	TotalItems int               `json:"totalItems"`
	Completed  int               `json:"completed"`
	Failed     int               `json:"failed"`
	Results    []ChildItemResult `json:"results"`
}

// ChildFanoutWorkflow fans out N child workflows and waits for all to complete.
// Demonstrates: child workflows, parallel execution, result aggregation, partial failure handling.
func ChildFanoutWorkflow(ctx workflow.Context, input ChildFanoutInput) (*ChildFanoutResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("ChildFanoutWorkflow started", "itemCount", len(input.Items))

	childOpts := workflow.ChildWorkflowOptions{
		WorkflowExecutionTimeout: 2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 2,
		},
	}
	ctx = workflow.WithChildOptions(ctx, childOpts)

	// Launch all child workflows in parallel
	var futures []workflow.ChildWorkflowFuture
	for _, item := range input.Items {
		future := workflow.ExecuteChildWorkflow(ctx, ChildProcessItemWorkflow, item)
		futures = append(futures, future)
	}

	// Collect results
	var results []ChildItemResult
	completed := 0
	failed := 0

	for i, future := range futures {
		var result ChildItemResult
		err := future.Get(ctx, &result)
		if err != nil {
			logger.Warn("Child workflow failed", "item", input.Items[i], "error", err)
			results = append(results, ChildItemResult{
				ItemID: input.Items[i],
				Status: fmt.Sprintf("failed: %v", err),
			})
			failed++
		} else {
			results = append(results, result)
			completed++
		}
	}

	logger.Info("ChildFanoutWorkflow completed",
		"total", len(input.Items),
		"completed", completed,
		"failed", failed,
	)

	return &ChildFanoutResult{
		TotalItems: len(input.Items),
		Completed:  completed,
		Failed:     failed,
		Results:    results,
	}, nil
}

// ChildProcessItemWorkflow processes a single item.
// This is the child workflow that gets spawned by ChildFanoutWorkflow.
func ChildProcessItemWorkflow(ctx workflow.Context, itemID string) (*ChildItemResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("ChildProcessItemWorkflow started", "itemId", itemID)

	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 2,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	var result string
	err := workflow.ExecuteActivity(ctx, activities.ProcessItem, itemID).Get(ctx, &result)
	if err != nil {
		return nil, err
	}

	return &ChildItemResult{
		ItemID:      itemID,
		Status:      "completed",
		ProcessedBy: "go-worker",
	}, nil
}
