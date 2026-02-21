#!/usr/bin/env bash
# =============================================================================
# seed-go.sh — Triggers all Go worker example workflows
# =============================================================================

set -euo pipefail

TASK_QUEUE="${GO_TASK_QUEUE:-go-task-queue}"
TEMPORAL="docker compose exec temporal-admin-tools temporal"
TS=$(date +%s)
NOW=$(date -u +%FT%TZ)

echo "1/5 Event-Driven Background Job..."
$TEMPORAL workflow start \
    --task-queue "$TASK_QUEUE" \
    --type EventProcessingWorkflow \
    --workflow-id "event-go-${TS}" \
    --input "{\"type\":\"order.created\",\"id\":\"order-123\",\"timestamp\":\"${NOW}\"}" \
    2>&1 | head -5

echo "2/5 Dummy Wait (30 seconds)..."
$TEMPORAL workflow start \
    --task-queue "$TASK_QUEUE" \
    --type DummyWaitWorkflow \
    --workflow-id "wait-go-${TS}" \
    --input '"30s"' \
    2>&1 | head -5

echo "3/5 Human Approval..."
$TEMPORAL workflow start \
    --task-queue "$TASK_QUEUE" \
    --type HumanApprovalWorkflow \
    --workflow-id "approval-go-${TS}" \
    --input '{"requester":"alice","item":"production-deploy-v2.1","timeout":"5m"}' \
    2>&1 | head -5

echo "4/5 Saga / Compensation..."
$TEMPORAL workflow start \
    --task-queue "$TASK_QUEUE" \
    --type SagaWorkflow \
    --workflow-id "saga-go-${TS}" \
    --input '{"orderId":"order-456","amount":99.99,"item":"Widget Pro"}' \
    2>&1 | head -5

echo "5/5 Child Workflow Fanout..."
$TEMPORAL workflow start \
    --task-queue "$TASK_QUEUE" \
    --type ChildFanoutWorkflow \
    --workflow-id "fanout-go-${TS}" \
    --input '{"items":["item-a","item-b","item-c","item-d","item-e"]}' \
    2>&1 | head -5

echo ""
echo "✅ All Go workflows started!"
echo "   Approval workflow ID: approval-go-${TS}"
echo "   (use 'make signal-approve WID=approval-go-${TS}' to approve)"
