package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"temporal-worker-go/activities"
)

// SagaInput represents an order to process through the saga.
type SagaInput struct {
	OrderID string  `json:"orderId"`
	Amount  float64 `json:"amount"`
	Item    string  `json:"item"`
}

// SagaResult holds the outcome of the saga.
type SagaResult struct {
	OrderID       string   `json:"orderId"`
	Status        string   `json:"status"` // "completed" or "compensated"
	StepsExecuted []string `json:"stepsExecuted"`
	Compensations []string `json:"compensations,omitempty"`
	Error         string   `json:"error,omitempty"`
}

// SagaWorkflow demonstrates the saga pattern with compensation.
//
// Steps: Reserve Inventory → Charge Payment → Ship Order
// If any step fails, run compensations in reverse order.
//
// To test compensation: the ShipOrder activity has a simulated ~30% failure rate.
func SagaWorkflow(ctx workflow.Context, input SagaInput) (*SagaResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("SagaWorkflow started", "orderId", input.OrderID, "amount", input.Amount, "item", input.Item)

	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1, // No retries — we want to demonstrate compensation
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	var stepsExecuted []string
	var compensations []string

	// Step 1: Reserve Inventory
	logger.Info("Step 1: Reserving inventory", "item", input.Item)
	var reserveResult string
	err := workflow.ExecuteActivity(ctx, activities.ReserveInventory, input.OrderID, input.Item).Get(ctx, &reserveResult)
	if err != nil {
		return &SagaResult{
			OrderID: input.OrderID,
			Status:  "failed",
			Error:   fmt.Sprintf("reserve failed: %v", err),
		}, nil
	}
	stepsExecuted = append(stepsExecuted, reserveResult)

	// Step 2: Charge Payment
	logger.Info("Step 2: Charging payment", "amount", input.Amount)
	var chargeResult string
	err = workflow.ExecuteActivity(ctx, activities.ChargePayment, input.OrderID, input.Amount).Get(ctx, &chargeResult)
	if err != nil {
		// Compensate step 1
		logger.Info("Payment failed, compensating: cancel reservation")
		var compResult string
		compErr := workflow.ExecuteActivity(ctx, activities.CancelReservation, input.OrderID, input.Item).Get(ctx, &compResult)
		if compErr != nil {
			compensations = append(compensations, fmt.Sprintf("FAILED: cancel reservation: %v", compErr))
		} else {
			compensations = append(compensations, compResult)
		}
		return &SagaResult{
			OrderID:       input.OrderID,
			Status:        "compensated",
			StepsExecuted: stepsExecuted,
			Compensations: compensations,
			Error:         fmt.Sprintf("payment failed: %v", err),
		}, nil
	}
	stepsExecuted = append(stepsExecuted, chargeResult)

	// Step 3: Ship Order (has simulated ~30% failure rate)
	logger.Info("Step 3: Shipping order", "orderId", input.OrderID)
	var shipResult string
	err = workflow.ExecuteActivity(ctx, activities.ShipOrder, input.OrderID, input.Item).Get(ctx, &shipResult)
	if err != nil {
		// Compensate steps 2 and 1 in reverse order
		logger.Info("Shipping failed, compensating: refund payment + cancel reservation")

		// Compensate step 2: refund
		var refundResult string
		compErr := workflow.ExecuteActivity(ctx, activities.RefundPayment, input.OrderID, input.Amount).Get(ctx, &refundResult)
		if compErr != nil {
			compensations = append(compensations, fmt.Sprintf("FAILED: refund payment: %v", compErr))
		} else {
			compensations = append(compensations, refundResult)
		}

		// Compensate step 1: cancel reservation
		var cancelResult string
		compErr = workflow.ExecuteActivity(ctx, activities.CancelReservation, input.OrderID, input.Item).Get(ctx, &cancelResult)
		if compErr != nil {
			compensations = append(compensations, fmt.Sprintf("FAILED: cancel reservation: %v", compErr))
		} else {
			compensations = append(compensations, cancelResult)
		}

		return &SagaResult{
			OrderID:       input.OrderID,
			Status:        "compensated",
			StepsExecuted: stepsExecuted,
			Compensations: compensations,
			Error:         fmt.Sprintf("shipping failed: %v", err),
		}, nil
	}
	stepsExecuted = append(stepsExecuted, shipResult)

	logger.Info("SagaWorkflow completed successfully", "orderId", input.OrderID)
	return &SagaResult{
		OrderID:       input.OrderID,
		Status:        "completed",
		StepsExecuted: stepsExecuted,
	}, nil
}
