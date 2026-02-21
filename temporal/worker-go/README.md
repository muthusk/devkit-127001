# Go Worker

This worker implements 5 Temporal workflow patterns in Go using the [Temporal Go SDK](https://pkg.go.dev/go.temporal.io/sdk). It listens on the `go-task-queue` task queue.

## Architecture

```
worker-go/
├── main.go                      # Worker entrypoint — registers all workflows + activities
├── workflows/
│   ├── event_driven.go          # Sequential activity chain
│   ├── dummy_wait.go            # Timer-based sleep workflow
│   ├── human_approval.go        # Signal-based human-in-the-loop
│   ├── saga_compensation.go     # Saga pattern with reverse compensations
│   └── child_fanout.go          # Parent spawns N child workflows
└── activities/
    ├── event_activities.go      # Validate, Process, Notify
    ├── approval_activities.go   # SendApprovalRequest, NotifyApprovalResult
    ├── saga_activities.go       # Reserve, Charge, Ship + compensations
    └── child_activities.go      # ProcessItem
```

### How It Works

`main.go` does three things:

1. **Connects** to the Temporal server using `TEMPORAL_ADDRESS` and `TEMPORAL_NAMESPACE` env vars.
2. **Registers** all 6 workflows and 11 activities with the worker.
3. **Starts listening** on the configured task queue, blocking until interrupted.

Every workflow function and activity function must be registered explicitly. When a workflow execution is started on the `go-task-queue`, the Temporal server dispatches it to this worker, which runs the workflow function.

---

## Workflows In Detail

### 1. EventProcessingWorkflow

**File:** `workflows/event_driven.go`

**Purpose:** Demonstrates a sequential activity pipeline with structured input/output and retry policies.

**Flow:**
```
Input (EventInput) → ValidateEvent → ProcessEvent → NotifyEventProcessed → Result (EventResult)
```

**Input:**
```json
{
  "type": "order.created",
  "id": "order-123",
  "timestamp": "2025-01-01T00:00:00Z"
}
```

**Output:**
```json
{
  "eventId": "order-123",
  "status": "completed",
  "processedAt": "2025-01-01T00:00:01Z",
  "steps": ["validated:order.created:order-123", "processed:...", "notified:..."]
}
```

**Trigger:**
```bash
make seed-event-go
```

**Patterns demonstrated:**
- `workflow.ActivityOptions` with retry policies
- Sequential `workflow.ExecuteActivity` calls
- Structured Go types for workflow I/O
- Error wrapping with `fmt.Errorf`

---

### 2. DummyWaitWorkflow

**File:** `workflows/dummy_wait.go`

**Purpose:** The simplest possible workflow — sleeps for a specified duration. Useful for testing the UI, cancellation, and timer behavior.

**Input:** A duration string like `"30s"`, `"2m"`, or `"5m"`.

**Output:**
```json
{
  "requestedDuration": "30s",
  "status": "completed",
  "startedAt": "...",
  "completedAt": "..."
}
```

**Trigger:**
```bash
make seed-wait-go
```

**Try cancellation:**
1. Start the workflow with a long duration: change input to `"5m"`
2. In the Temporal UI, find the running workflow and click "Terminate" or "Cancel"
3. Check logs: the workflow reports `cancelled` status

**Patterns demonstrated:**
- `workflow.Sleep(ctx, duration)`
- `workflow.Now(ctx)` for deterministic timestamps
- Cancellation detection via context error
- `time.ParseDuration` for flexible input

---

### 3. HumanApprovalWorkflow

**File:** `workflows/human_approval.go`

**Purpose:** Blocks execution until a human sends a signal. Includes a configurable timeout for auto-rejection.

**Flow:**
```
Start → SendApprovalRequest → Wait for Signal ──┬── Signal received → Approved/Rejected
                                                  └── Timeout → Auto-rejected
```

**Input:**
```json
{
  "requester": "alice",
  "item": "production-deploy-v2.1",
  "timeout": "5m"
}
```

**Signal name:** `approval`

**Signal payload:**
```json
{
  "approved": true,
  "approver": "admin",
  "reason": "Looks good"
}
```

**Trigger and interact:**
```bash
# Start the workflow
make seed-approval-go

# Send approval signal (replace with actual workflow ID)
make signal-approve WID=approval-go-1234567890

# Or reject it
make signal-reject WID=approval-go-1234567890
```

**Patterns demonstrated:**
- `workflow.GetSignalChannel(ctx, "approval")` for signal reception
- `workflow.NewSelector(ctx)` for racing signal vs. timer
- `workflow.WithCancel(ctx)` for timer cancellation after signal
- Configurable timeouts with graceful fallback

---

### 4. SagaWorkflow

**File:** `workflows/saga_compensation.go`

**Purpose:** The classic saga/compensation pattern. Runs a multi-step transaction and automatically rolls back completed steps if a later step fails.

**Flow (success):**
```
ReserveInventory → ChargePayment → ShipOrder → ✅ Completed
```

**Flow (ship fails — ~30% chance):**
```
ReserveInventory → ChargePayment → ShipOrder ❌
    └── Compensate: RefundPayment → CancelReservation → Status: "compensated"
```

**Input:**
```json
{
  "orderId": "order-456",
  "amount": 99.99,
  "item": "Widget Pro"
}
```

**Output (compensated):**
```json
{
  "orderId": "order-456",
  "status": "compensated",
  "stepsExecuted": ["reserved:Widget Pro:order-456", "charged:99.99:order-456"],
  "compensations": ["refunded:99.99:order-456", "cancelled-reservation:Widget Pro:order-456"],
  "error": "shipping failed: warehouse unavailable for order order-456"
}
```

**Trigger:**
```bash
make seed-saga-go
```

> **Tip:** Run this workflow 3-5 times to see both success and compensation paths, since the failure is probabilistic.

**Patterns demonstrated:**
- Saga pattern: forward steps + reverse compensations
- `MaximumAttempts: 1` to prevent retries (let compensation handle it)
- Compensation ordering (reverse of execution)
- Graceful handling of compensation failures

---

### 5. ChildFanoutWorkflow + ChildProcessItemWorkflow

**File:** `workflows/child_fanout.go`

**Purpose:** A parent workflow spawns N child workflows in parallel, waits for all of them, and aggregates results. Handles partial failures gracefully.

**Flow:**
```
Parent ──┬── Child(item-a) → ✅
         ├── Child(item-b) → ✅
         ├── Child(item-c) → ✅
         ├── Child(item-d) → ✅
         └── Child(item-e) → ✅
         └── Aggregate results
```

**Input:**
```json
{
  "items": ["item-a", "item-b", "item-c", "item-d", "item-e"]
}
```

**Output:**
```json
{
  "totalItems": 5,
  "completed": 5,
  "failed": 0,
  "results": [
    {"itemId": "item-a", "status": "completed", "processedBy": "go-worker"},
    ...
  ]
}
```

**Trigger:**
```bash
make seed-child-go
```

**In the Temporal UI:** You'll see the parent workflow and 5 child workflows. Each child is its own workflow execution with its own history.

**Patterns demonstrated:**
- `workflow.ExecuteChildWorkflow` for spawning children
- `workflow.ChildWorkflowFuture` for async result collection
- `workflow.ChildWorkflowOptions` with retry and timeout
- Partial failure handling: parent continues even if some children fail

---

## Activities Reference

All activities are stateless functions that simulate real work with `time.Sleep()`. They use `activity.GetLogger(ctx)` for structured logging.

| Activity | File | Simulated Latency | Failure Rate |
|----------|------|-------------------|-------------|
| `ValidateEvent` | `event_activities.go` | 500ms | 0% |
| `ProcessEvent` | `event_activities.go` | 1s | 0% |
| `NotifyEventProcessed` | `event_activities.go` | 300ms | 0% |
| `SendApprovalRequest` | `approval_activities.go` | 500ms | 0% |
| `NotifyApprovalResult` | `approval_activities.go` | 300ms | 0% |
| `ReserveInventory` | `saga_activities.go` | 800ms | 0% |
| `ChargePayment` | `saga_activities.go` | 1s | 0% |
| `ShipOrder` | `saga_activities.go` | 1s | **~30%** |
| `CancelReservation` | `saga_activities.go` | 500ms | 0% |
| `RefundPayment` | `saga_activities.go` | 500ms | 0% |
| `ProcessItem` | `child_activities.go` | 1-3s (variable) | 0% |

---

## How to Add a New Workflow

1. **Create the workflow file:**
   ```bash
   touch worker-go/workflows/my_workflow.go
   ```

2. **Define your workflow function:**
   ```go
   package workflows

   import "go.temporal.io/sdk/workflow"

   type MyInput struct {
       Name string `json:"name"`
   }

   func MyWorkflow(ctx workflow.Context, input MyInput) (string, error) {
       // Your workflow logic here
       return "done", nil
   }
   ```

3. **Create activities if needed:**
   ```bash
   touch worker-go/activities/my_activities.go
   ```

4. **Register in `main.go`:**
   ```go
   w.RegisterWorkflow(workflows.MyWorkflow)
   w.RegisterActivity(activities.MyActivity)
   ```

5. **Rebuild:**
   ```bash
   make rebuild-go
   ```

6. **Run it:**
   ```bash
   docker compose exec temporal-admin-tools temporal workflow start \
       --task-queue go-task-queue \
       --type MyWorkflow \
       --input '{"name": "test"}'
   ```

---

## Running Outside Docker (Local Development)

For faster iteration without Docker rebuilds:

1. **Install Go 1.22+** on your host machine
2. **Start infrastructure only:**
   ```bash
   docker compose up -d postgres temporal-server temporal-ui temporal-admin-tools
   ```
3. **Run the worker locally:**
   ```bash
   cd worker-go
   TEMPORAL_ADDRESS=localhost:7233 go run main.go
   ```

The worker connects to the Dockerized Temporal server via `localhost:7233`.

---

## Temporal Go SDK Version

This worker uses `go.temporal.io/sdk v1.29.1`. For the latest version and API docs, see the [Temporal Go SDK documentation](https://pkg.go.dev/go.temporal.io/sdk).

Key SDK packages used:
- `go.temporal.io/sdk/workflow` — workflow definitions, timers, signals, child workflows
- `go.temporal.io/sdk/activity` — activity definitions, logging, heartbeats
- `go.temporal.io/sdk/client` — Temporal server connection
- `go.temporal.io/sdk/worker` — worker creation and task queue registration
- `go.temporal.io/sdk/temporal` — retry policies, error types
