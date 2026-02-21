package activities

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
)

// SendApprovalRequest simulates sending an approval request notification.
func SendApprovalRequest(ctx context.Context, requester string, item string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("SendApprovalRequest",
		"requester", requester,
		"item", item,
	)

	// Simulate sending an email/Slack message/webhook
	time.Sleep(500 * time.Millisecond)

	logger.Info(fmt.Sprintf("📨 Approval request sent: %s is requesting approval for '%s'", requester, item))
	return nil
}

// NotifyApprovalResult simulates sending a notification about the approval outcome.
func NotifyApprovalResult(ctx context.Context, status string, item string, requester string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("NotifyApprovalResult",
		"status", status,
		"item", item,
		"requester", requester,
	)

	time.Sleep(300 * time.Millisecond)

	logger.Info(fmt.Sprintf("📬 Approval result: '%s' was %s (requested by %s)", item, status, requester))
	return nil
}
