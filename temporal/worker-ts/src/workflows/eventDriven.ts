import { proxyActivities, log } from '@temporalio/workflow';
import type * as activities from '../activities/eventActivities';

const { validateEvent, enrichEvent, processEvent, notifyEventProcessed } = proxyActivities<
  typeof activities
>({
  startToCloseTimeout: '30s',
  retry: {
    initialInterval: '1s',
    backoffCoefficient: 2,
    maximumInterval: '30s',
    maximumAttempts: 3,
  },
});

export interface EventInput {
  type: string;
  id: string;
  timestamp: string;
}

export interface EventResult {
  eventId: string;
  status: string;
  processedAt: string;
  steps: string[];
}

/**
 * Event Processing Workflow — TypeScript Variant
 *
 * DIFFERENCE FROM GO: Runs validate + enrich in PARALLEL using Promise.all,
 * then runs process sequentially. Demonstrates concurrent activity execution.
 */
export async function eventProcessingWorkflow(input: EventInput): Promise<EventResult> {
  log.info('eventProcessingWorkflow started', { eventType: input.type, eventId: input.id });

  const steps: string[] = [];

  // Step 1 & 2: Validate AND Enrich in parallel
  const [validateResult, enrichResult] = await Promise.all([
    validateEvent(input.type, input.id),
    enrichEvent(input.type, input.id),
  ]);
  steps.push(validateResult, enrichResult);
  log.info('Validation and enrichment complete (parallel)', { validateResult, enrichResult });

  // Step 3: Process (sequential — depends on validation)
  const processResult = await processEvent(input.type, input.id);
  steps.push(processResult);
  log.info('Processing complete', { processResult });

  // Step 4: Notify
  const notifyResult = await notifyEventProcessed(input.type, input.id);
  steps.push(notifyResult);
  log.info('Notification complete', { notifyResult });

  return {
    eventId: input.id,
    status: 'completed',
    processedAt: new Date().toISOString(),
    steps,
  };
}
