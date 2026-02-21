# TypeScript Worker

This worker implements the same 5 Temporal workflow patterns as the Go worker, but with **deliberate variations** to teach additional TypeScript SDK concepts. It listens on the `ts-task-queue` task queue.

## How It Differs From the Go Worker

Each workflow intentionally varies from its Go counterpart:

| Workflow | Go Version | TS Variation |
|----------|-----------|--------------|
| Event-Driven | Sequential activities | Parallel activities via `Promise.all` |
| Dummy Wait | Single sleep | Multi-stage sleep with checkpoints |
| Human Approval | Signal + timeout | Signal + timeout **+ Query handler** |
| Saga | 3 steps, clean compensations | 4 steps, **compensation can fail** |
| Child Fanout | All children at once | **Concurrency-limited** batching |

Reading both implementations side-by-side is the fastest way to learn how Temporal concepts translate across languages.

---

## Architecture

```
worker-ts/
├── src/
│   ├── worker.ts                  # Worker entrypoint — connects + starts
│   ├── client.ts                  # Optional programmatic workflow starter
│   ├── workflows/
│   │   ├── index.ts               # Re-exports all workflows (required by SDK)
│   │   ├── eventDriven.ts         # Parallel activity execution
│   │   ├── dummyWait.ts           # Multi-stage timer with checkpoints
│   │   ├── humanApproval.ts       # Signal + Query + timeout
│   │   ├── sagaCompensation.ts    # 4-step saga with failible compensation
│   │   └── childFanout.ts         # Concurrency-limited child workflows
│   └── activities/
│       ├── eventActivities.ts     # Validate, Enrich, Process, Notify
│       ├── approvalActivities.ts  # SendRequest, NotifyResult
│       ├── sagaActivities.ts      # 4 forward + 3 compensation activities
│       └── childActivities.ts     # ProcessItem
├── Dockerfile                     # Multi-stage Node.js build
├── package.json
├── tsconfig.json
└── jest.config.js                 # Testing setup
```

### How It Works

The Temporal TypeScript SDK has a unique architecture compared to Go:

- **Workflows run in a sandboxed V8 isolate** — they can only import from `@temporalio/workflow` and cannot do I/O, use `Date.now()`, or import Node.js modules.
- **Activities run in the main Node.js context** — they can do anything (HTTP calls, DB queries, etc.).
- **`worker.ts`** bundles workflows from the `workflowsPath` and registers activities as plain functions.
- **`workflows/index.ts`** re-exports all workflow functions. This file is the entry point the SDK bundles.

This is why workflows and activities are in separate directories and why `index.ts` is required.

---

## Workflows In Detail

### 1. eventProcessingWorkflow (Parallel Variant)

**File:** `src/workflows/eventDriven.ts`

**What's different from Go:** Steps 1 and 2 run in **parallel** using `Promise.all`. This is natural in TypeScript and demonstrates that Temporal activities can execute concurrently.

**Flow:**
```
Input → [ValidateEvent + EnrichEvent] (parallel) → ProcessEvent → NotifyEventProcessed → Result
```

**Input:**
```json
{
  "type": "user.signup",
  "id": "user-789",
  "timestamp": "2025-01-01T00:00:00Z"
}
```

**Trigger:**
```bash
make seed-event-ts
```

**Key TS pattern:**
```typescript
const [validateResult, enrichResult] = await Promise.all([
  validateEvent(input.type, input.id),
  enrichEvent(input.type, input.id),
]);
```

This is safe in Temporal because `Promise.all` is deterministic — the SDK replays both activities consistently.

---

### 2. dummyWaitWorkflow (Multi-Stage)

**File:** `src/workflows/dummyWait.ts`

**What's different from Go:** Instead of one big sleep, it splits the duration into 3 stages with checkpoint logging between each one.

**Flow:**
```
Start → Sleep(stage 1) → Checkpoint → Sleep(stage 2) → Checkpoint → Sleep(stage 3) → Done
```

**Output includes checkpoints:**
```json
{
  "requestedDuration": "30s",
  "status": "completed",
  "checkpoints": [
    "started:2025-01-01T00:00:00Z",
    "stage-1-complete:2025-01-01T00:00:10Z",
    "stage-2-complete:2025-01-01T00:00:20Z",
    "stage-3-complete:2025-01-01T00:00:30Z",
    "completed:2025-01-01T00:00:30Z"
  ]
}
```

**Trigger:**
```bash
make seed-wait-ts
```

**Key TS pattern:** Uses `CancelledFailure` from `@temporalio/workflow` for typed cancellation handling:
```typescript
import { CancelledFailure } from '@temporalio/workflow';

try {
  await sleep(duration);
} catch (err) {
  if (err instanceof CancelledFailure) {
    return { status: 'cancelled', ... };
  }
  throw err;
}
```

---

### 3. humanApprovalWorkflow (With Query Handler)

**File:** `src/workflows/humanApproval.ts`

**What's different from Go:** Adds a **Query handler** that lets you check the approval status at any time without opening the UI.

**Signal name:** `approval` (same as Go)
**Query name:** `approvalStatus` (TS-only addition)

**Trigger and interact:**
```bash
# Start the workflow
make seed-approval-ts

# Check status at any time (TS-only feature)
make query-approval-ts WID=approval-ts-1234567890

# Approve it
make signal-approve WID=approval-ts-1234567890
```

**Query response:**
```json
{
  "item": "staging-db-migration",
  "requester": "bob",
  "status": "pending",
  "waitingSince": "2025-01-01T00:00:00Z"
}
```

**Key TS patterns:**
```typescript
import { defineSignal, defineQuery, setHandler, condition } from '@temporalio/workflow';

export const approvalSignal = defineSignal<[ApprovalSignalInput]>('approval');
export const approvalStatusQuery = defineQuery<ApprovalStatus>('approvalStatus');

// Inside workflow:
setHandler(approvalSignal, (signal) => { /* update state */ });
setHandler(approvalStatusQuery, () => currentStatus);

// Wait for signal with timeout (cleaner than Go's selector pattern)
const received = await condition(() => signalReceived, timeoutMs);
```

The `condition()` function is TS-specific and much more concise than Go's `NewSelector` pattern.

---

### 4. sagaWorkflow (4-Step with Failible Compensation)

**File:** `src/workflows/sagaCompensation.ts`

**What's different from Go:**
1. **4 steps** instead of 3: reserve → charge → ship → confirm
2. The `cancelShipping` compensation has a **~20% chance of failing itself**
3. Uses a **compensation stack** pattern (push compensations as you go, pop in reverse)

**Flow (confirmation fails + shipping compensation fails):**
```
Reserve ✅ → Charge ✅ → Ship ✅ → Confirm ❌
    └── Compensate: CancelShipping ❌ (FAILED) → RefundPayment ✅ → CancelReservation ✅
```

**Output when compensation itself fails:**
```json
{
  "orderId": "order-789",
  "status": "compensated",
  "stepsExecuted": ["reserved:...", "charged:...", "shipping-arranged:..."],
  "compensations": [
    "FAILED: Shipping cancellation failed for order order-789 — manual intervention needed",
    "refunded:149.99:order-789",
    "cancelled-reservation:Gadget Ultra:order-789"
  ],
  "error": "Confirmation service unavailable for order order-789"
}
```

**Trigger:**
```bash
make seed-saga-ts
```

**Key TS pattern — compensation stack:**
```typescript
const compensationStack: Array<() => Promise<string>> = [];

try {
  const result = await reserveInventory(orderId, item);
  compensationStack.push(() => cancelReservation(orderId, item));

  // ... more steps, each pushing their compensation

} catch (err) {
  // Run compensations in reverse
  while (compensationStack.length > 0) {
    const compensate = compensationStack.pop()!;
    try {
      await compensate();
    } catch (compErr) {
      // Handle compensation failure
    }
  }
}
```

---

### 5. childFanoutWorkflow (Concurrency-Limited)

**File:** `src/workflows/childFanout.ts`

**What's different from Go:** Instead of launching all children at once, processes items in **batches** with a configurable concurrency limit (default: 3).

**Flow (5 items, concurrency limit 3):**
```
Batch 1: [task-1, task-2, task-3] → wait for all 3
Batch 2: [task-4, task-5]         → wait for both
→ Aggregate all results
```

**Input:**
```json
{
  "items": ["task-1", "task-2", "task-3", "task-4", "task-5"],
  "concurrencyLimit": 3
}
```

**Trigger:**
```bash
make seed-child-ts
```

**Key TS pattern — batched execution:**
```typescript
for (let i = 0; i < items.length; i += concurrencyLimit) {
  const batch = items.slice(i, i + concurrencyLimit);

  const batchPromises = batch.map((item) =>
    executeChild(childProcessItemWorkflow, { args: [item], workflowId: `...` })
  );

  const batchResults = await Promise.all(batchPromises);
  // Process batch results before starting next batch
}
```

---

## Activities Reference

| Activity | File | Simulated Latency | Failure Rate |
|----------|------|-------------------|-------------|
| `validateEvent` | `eventActivities.ts` | 500ms | 0% |
| `enrichEvent` | `eventActivities.ts` | 700ms | 0% |
| `processEvent` | `eventActivities.ts` | 1s | 0% |
| `notifyEventProcessed` | `eventActivities.ts` | 300ms | 0% |
| `sendApprovalRequest` | `approvalActivities.ts` | 500ms | 0% |
| `notifyApprovalResult` | `approvalActivities.ts` | 300ms | 0% |
| `reserveInventory` | `sagaActivities.ts` | 800ms | 0% |
| `chargePayment` | `sagaActivities.ts` | 1s | 0% |
| `arrangeShipping` | `sagaActivities.ts` | 1s | 0% |
| `sendConfirmation` | `sagaActivities.ts` | 500ms | **~30%** |
| `cancelReservation` | `sagaActivities.ts` | 500ms | 0% |
| `refundPayment` | `sagaActivities.ts` | 500ms | 0% |
| `cancelShipping` | `sagaActivities.ts` | 500ms | **~20%** |
| `processItem` | `childActivities.ts` | 1-3s | 0% |

---

## How to Add a New Workflow

1. **Create the workflow file:**
   ```bash
   touch worker-ts/src/workflows/myWorkflow.ts
   ```

2. **Write your workflow:**
   ```typescript
   import { proxyActivities, log } from '@temporalio/workflow';
   import type * as activities from '../activities/myActivities';

   const { myActivity } = proxyActivities<typeof activities>({
     startToCloseTimeout: '30s',
   });

   export async function myWorkflow(input: { name: string }): Promise<string> {
     log.info('myWorkflow started', { name: input.name });
     const result = await myActivity(input.name);
     return result;
   }
   ```

3. **Export from `workflows/index.ts`:**
   ```typescript
   export { myWorkflow } from './myWorkflow';
   ```

4. **Create activities:**
   ```typescript
   // src/activities/myActivities.ts
   export async function myActivity(name: string): Promise<string> {
     return `Hello, ${name}!`;
   }
   ```

5. **Register activities in `worker.ts`:**
   ```typescript
   import * as myActivities from './activities/myActivities';
   // ...
   activities: { ...activities, ...myActivities },
   ```

6. **Rebuild:**
   ```bash
   make rebuild-ts
   ```

---

## Running Outside Docker (Local Development)

For faster iteration:

1. **Install Node.js 20+** on your host machine
2. **Start infrastructure only:**
   ```bash
   docker compose up -d postgres temporal-server temporal-ui temporal-admin-tools
   ```
3. **Install dependencies and run locally:**
   ```bash
   cd worker-ts
   npm install
   TEMPORAL_ADDRESS=localhost:7233 npm run dev
   ```

### Using the Programmatic Client

Instead of the CLI, you can start workflows programmatically:

```bash
cd worker-ts
npx ts-node src/client.ts eventProcessingWorkflow '{"type":"test","id":"1","timestamp":"now"}'
npx ts-node src/client.ts dummyWaitWorkflow '"10s"'
npx ts-node src/client.ts sagaWorkflow '{"orderId":"test-1","amount":50,"item":"Test"}'
```

The client connects to Temporal, starts the workflow, and waits for the result.

---

## Testing

A basic Jest setup is included. The Temporal TS SDK supports workflow testing with the `@temporalio/testing` package.

```bash
cd worker-ts
npm test
```

For testing patterns, see the [Temporal TS testing guide](https://docs.temporal.io/develop/typescript/testing-suite).

---

## TypeScript SDK Key Concepts

### Workflow Sandbox

Workflows run in an isolated V8 context. They **cannot**:
- Import Node.js built-in modules (`fs`, `path`, `http`, etc.)
- Use `Date.now()` (use `workflow.now()` or let the SDK handle it)
- Make network calls
- Access global state

This is by design — it ensures workflows are deterministic and can be replayed.

### `proxyActivities`

Activities are accessed via `proxyActivities<typeof activities>()` which creates typed proxy functions. The `typeof` import gives you full type safety without importing the actual activity code into the sandbox.

### Signals and Queries

```typescript
// Define at module level
export const mySignal = defineSignal<[MyPayload]>('my-signal');
export const myQuery = defineQuery<MyResponse>('my-query');

// Set handlers inside workflow function
setHandler(mySignal, (payload) => { /* mutate workflow state */ });
setHandler(myQuery, () => { /* return current state */ });
```

### `condition()` vs Go's `Selector`

The TS SDK's `condition(fn, timeout?)` is much more concise than Go's selector pattern:

```typescript
// Wait until `signalReceived` becomes true, or timeout after 5 minutes
const received = await condition(() => signalReceived, 300_000);
```

---

## SDK Version

This worker uses `@temporalio/worker ^1.11.0` and related packages. For the latest version and API docs, see:

- [Temporal TypeScript SDK Docs](https://typescript.temporal.io/)
- [API Reference](https://typescript.temporal.io/api/namespaces/workflow)
- [GitHub Repository](https://github.com/temporalio/sdk-typescript)
