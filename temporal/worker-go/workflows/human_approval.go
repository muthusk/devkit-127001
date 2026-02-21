package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"temporal-worker-go/activities"
)

// ApprovalInput contains the request details.
type ApprovalInput struct {
	Requester string `json:"requester"`
	Item      string `json:"item"`
	Timeout   string `json:"timeout"` // e.g., "5m", "10m"
}

// ApprovalSignal is sent via the "approval" signal channel.
type ApprovalSignal struct {
	Approved bool   `json:"approved"`
	Approver string `json:"approver"`
	Reason   string `json:"reason"`
}

// ApprovalResult holds the final outcome.
type ApprovalResult struct {
	Item      string `json:"item"`
	Requester string `json:"requester"`
	Status    string `json:"status"` // "approved", "rejected", "timed_out"
	Approver  string `json:"approver,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

// HumanApprovalWorkflow blocks on a signal to simulate manual intervention.
// Demonstrates: signals, human-in-the-loop, configurable timeouts.
//
// Signal name: "approval"
// Signal payload: ApprovalSignal JSON
func HumanApprovalWorkflow(ctx workflow.Context, input ApprovalInput) (*ApprovalResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("HumanApprovalWorkflow started",
		"requester", input.Requester,
		"item", input.Item,
		"timeout", input.Timeout,
	)

	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 15 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	// Step 1: Send notification that approval is needed
	err := workflow.ExecuteActivity(ctx, activities.SendApprovalRequest, input.Requester, input.Item).Get(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to send approval request: %w", err)
	}
	logger.Info("Approval request sent", "item", input.Item)

	// Step 2: Wait for signal with timeout
	timeout, err := time.ParseDuration(input.Timeout)
	if err != nil {
		timeout = 10 * time.Minute // default
	}

	signalCh := workflow.GetSignalChannel(ctx, "approval")

	var signal ApprovalSignal
	var received bool

	// Create a selector to wait for either signal or timeout
	selector := workflow.NewSelector(ctx)

	selector.AddReceive(signalCh, func(ch workflow.ReceiveChannel, more bool) {
		ch.Receive(ctx, &signal)
		received = true
	})

	timerCtx, timerCancel := workflow.WithCancel(ctx)
	selector.AddFuture(workflow.NewTimer(timerCtx, timeout), func(f workflow.Future) {
		// Timer fired — timeout
		received = false
	})

	selector.Select(ctx)
	timerCancel() // Cancel timer if signal was received first

	// Step 3: Handle the result
	var result ApprovalResult
	result.Item = input.Item
	result.Requester = input.Requester

	if !received {
		// Timed out
		logger.Info("Approval timed out", "item", input.Item, "timeout", input.Timeout)
		result.Status = "timed_out"
		result.Reason = fmt.Sprintf("No response within %s", input.Timeout)
	} else if signal.Approved {
		logger.Info("Approval granted", "item", input.Item, "approver", signal.Approver)
		result.Status = "approved"
		result.Approver = signal.Approver
		result.Reason = signal.Reason
	} else {
		logger.Info("Approval rejected", "item", input.Item, "approver", signal.Approver)
		result.Status = "rejected"
		result.Approver = signal.Approver
		result.Reason = signal.Reason
	}

	// Step 4: Notify the outcome
	err = workflow.ExecuteActivity(ctx, activities.NotifyApprovalResult, result.Status, input.Item, input.Requester).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to send result notification", "error", err)
		// Non-fatal — workflow still completes
	}

	return &result, nil
}
