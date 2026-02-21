#!/usr/bin/env bash
# =============================================================================
# seed-ts.sh — Triggers all TypeScript worker example workflows
# =============================================================================

set -euo pipefail

TASK_QUEUE="${TS_TASK_QUEUE:-ts-task-queue}"
TEMPORAL="docker compose exec temporal-admin-tools temporal"
TS=$(date +%s)
NOW=$(date -u +%FT%TZ)

echo "1/5 Event-Driven Background Job (parallel variant)..."
$TEMPORAL workflow start \
    --task-queue "$TASK_QUEUE" \
    --type eventProcessingWorkflow \
    --workflow-id "event-ts-${TS}" \
    --input "{\"type\":\"user.signup\",\"id\":\"user-789\",\"timestamp\":\"${NOW}\"}" \
    2>&1 | head -5

echo "2/5 Dummy Wait — Multi-Stage (30 seconds)..."
$TEMPORAL workflow start \
    --task-queue "$TASK_QUEUE" \
    --type dummyWaitWorkflow \
    --workflow-id "wait-ts-${TS}" \
    --input '"30s"' \
    2>&1 | head -5

echo "3/5 Human Approval (with query support)..."
$TEMPORAL workflow start \
    --task-queue "$TASK_QUEUE" \
    --type humanApprovalWorkflow \
    --workflow-id "approval-ts-${TS}" \
    --input '{"requester":"bob","item":"staging-db-migration","timeout":"5m"}' \
    2>&1 | head -5

echo "4/5 Saga / Compensation (4-step variant)..."
$TEMPORAL workflow start \
    --task-queue "$TASK_QUEUE" \
    --type sagaWorkflow \
    --workflow-id "saga-ts-${TS}" \
    --input '{"orderId":"order-789","amount":149.99,"item":"Gadget Ultra"}' \
    2>&1 | head -5

echo "5/5 Child Workflow Fanout (with concurrency limit)..."
$TEMPORAL workflow start \
    --task-queue "$TASK_QUEUE" \
    --type childFanoutWorkflow \
    --workflow-id "fanout-ts-${TS}" \
    --input '{"items":["task-1","task-2","task-3","task-4","task-5"],"concurrencyLimit":3}' \
    2>&1 | head -5

echo ""
echo "✅ All TS workflows started!"
echo "   Approval workflow ID: approval-ts-${TS}"
echo "   (use 'make signal-approve WID=approval-ts-${TS}' to approve)"
echo "   (use 'make query-approval-ts WID=approval-ts-${TS}' to check status)"
