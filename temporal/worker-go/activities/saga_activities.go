package activities

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"go.temporal.io/sdk/activity"
)

// ReserveInventory simulates reserving an item in inventory.
func ReserveInventory(ctx context.Context, orderID string, item string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("ReserveInventory", "orderId", orderID, "item", item)

	time.Sleep(800 * time.Millisecond)

	logger.Info(fmt.Sprintf("📦 Inventory reserved: %s for order %s", item, orderID))
	return fmt.Sprintf("reserved:%s:%s", item, orderID), nil
}

// ChargePayment simulates charging a payment.
func ChargePayment(ctx context.Context, orderID string, amount float64) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("ChargePayment", "orderId", orderID, "amount", amount)

	time.Sleep(1 * time.Second)

	logger.Info(fmt.Sprintf("💳 Payment charged: $%.2f for order %s", amount, orderID))
	return fmt.Sprintf("charged:%.2f:%s", amount, orderID), nil
}

// ShipOrder simulates shipping an order. Has a ~30% simulated failure rate
// to demonstrate saga compensation.
func ShipOrder(ctx context.Context, orderID string, item string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("ShipOrder", "orderId", orderID, "item", item)

	time.Sleep(1 * time.Second)

	// Simulate occasional shipping failure (~30% chance)
	// Uses workflow-deterministic randomness via activity info
	info := activity.GetInfo(ctx)
	r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(info.Attempt)))
	if r.Float64() < 0.3 {
		logger.Warn("ShipOrder FAILED (simulated)", "orderId", orderID)
		return "", fmt.Errorf("shipping failed: warehouse unavailable for order %s", orderID)
	}

	logger.Info(fmt.Sprintf("🚚 Order shipped: %s (%s)", orderID, item))
	return fmt.Sprintf("shipped:%s:%s", item, orderID), nil
}

// CancelReservation compensates for ReserveInventory.
func CancelReservation(ctx context.Context, orderID string, item string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("CancelReservation (compensation)", "orderId", orderID, "item", item)

	time.Sleep(500 * time.Millisecond)

	logger.Info(fmt.Sprintf("↩️ Reservation cancelled: %s for order %s", item, orderID))
	return fmt.Sprintf("cancelled-reservation:%s:%s", item, orderID), nil
}

// RefundPayment compensates for ChargePayment.
func RefundPayment(ctx context.Context, orderID string, amount float64) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("RefundPayment (compensation)", "orderId", orderID, "amount", amount)

	time.Sleep(500 * time.Millisecond)

	logger.Info(fmt.Sprintf("💸 Payment refunded: $%.2f for order %s", amount, orderID))
	return fmt.Sprintf("refunded:%.2f:%s", amount, orderID), nil
}
