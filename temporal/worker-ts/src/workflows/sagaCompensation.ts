import { proxyActivities, log } from '@temporalio/workflow';
import type * as activities from '../activities/sagaActivities';

const {
  reserveInventory,
  chargePayment,
  arrangeShipping,
  sendConfirmation,
  cancelReservation,
  refundPayment,
  cancelShipping,
} = proxyActivities<typeof activities>({
  startToCloseTimeout: '30s',
  retry: { maximumAttempts: 1 }, // No retries — demonstrate compensation
});

export interface SagaInput {
  orderId: string;
  amount: number;
  item: string;
}

export interface SagaResult {
  orderId: string;
  status: 'completed' | 'compensated' | 'failed';
  stepsExecuted: string[];
  compensations: string[];
  error?: string;
}

/**
 * Saga Workflow — TypeScript Variant (4 steps)
 *
 * DIFFERENCE FROM GO: Has 4 steps instead of 3:
 *   Reserve → Charge → Arrange Shipping → Send Confirmation
 *
 * The cancelShipping compensation itself has a simulated failure chance,
 * demonstrating nested error handling in compensations.
 *
 * SendConfirmation has a ~30% failure rate to trigger compensations.
 */
export async function sagaWorkflow(input: SagaInput): Promise<SagaResult> {
  log.info('sagaWorkflow started', { orderId: input.orderId, amount: input.amount, item: input.item });

  const stepsExecuted: string[] = [];
  const compensations: string[] = [];

  // Build a compensation stack — compensations run in reverse order
  const compensationStack: Array<() => Promise<string>> = [];

  try {
    // Step 1: Reserve Inventory
    log.info('Step 1: Reserving inventory', { item: input.item });
    const reserveResult = await reserveInventory(input.orderId, input.item);
    stepsExecuted.push(reserveResult);
    compensationStack.push(() => cancelReservation(input.orderId, input.item));

    // Step 2: Charge Payment
    log.info('Step 2: Charging payment', { amount: input.amount });
    const chargeResult = await chargePayment(input.orderId, input.amount);
    stepsExecuted.push(chargeResult);
    compensationStack.push(() => refundPayment(input.orderId, input.amount));

    // Step 3: Arrange Shipping
    log.info('Step 3: Arranging shipping');
    const shippingResult = await arrangeShipping(input.orderId, input.item);
    stepsExecuted.push(shippingResult);
    compensationStack.push(() => cancelShipping(input.orderId, input.item));

    // Step 4: Send Confirmation (has ~30% failure rate)
    log.info('Step 4: Sending confirmation');
    const confirmResult = await sendConfirmation(input.orderId, input.item, input.amount);
    stepsExecuted.push(confirmResult);

    log.info('sagaWorkflow completed successfully', { orderId: input.orderId });
    return {
      orderId: input.orderId,
      status: 'completed',
      stepsExecuted,
      compensations: [],
    };
  } catch (err) {
    const errorMessage = err instanceof Error ? err.message : String(err);
    log.warn('Saga step failed, running compensations', { error: errorMessage });

    // Run compensations in reverse order
    while (compensationStack.length > 0) {
      const compensate = compensationStack.pop()!;
      try {
        const result = await compensate();
        compensations.push(result);
      } catch (compErr) {
        // Compensation itself failed — this is the TS-specific twist
        const compErrorMsg = compErr instanceof Error ? compErr.message : String(compErr);
        log.error('Compensation FAILED', { error: compErrorMsg });
        compensations.push(`FAILED: ${compErrorMsg}`);
      }
    }

    return {
      orderId: input.orderId,
      status: 'compensated',
      stepsExecuted,
      compensations,
      error: errorMessage,
    };
  }
}
