import { executeChild, log } from '@temporalio/workflow';
import { proxyActivities } from '@temporalio/workflow';
import type * as activities from '../activities/childActivities';

const { processItem } = proxyActivities<typeof activities>({
  startToCloseTimeout: '30s',
  retry: { maximumAttempts: 2 },
});

export interface ChildFanoutInput {
  items: string[];
  concurrencyLimit?: number; // defaults to 3
}

export interface ChildItemResult {
  itemId: string;
  status: string;
  processedBy: string;
}

export interface ChildFanoutResult {
  totalItems: number;
  completed: number;
  failed: number;
  results: ChildItemResult[];
}

/**
 * Child Fanout Workflow — TypeScript Variant
 *
 * DIFFERENCE FROM GO: Adds a CONCURRENCY LIMIT. Instead of launching all
 * children at once, this processes items in batches to demonstrate
 * controlled parallelism. Default concurrency: 3.
 */
export async function childFanoutWorkflow(input: ChildFanoutInput): Promise<ChildFanoutResult> {
  const concurrencyLimit = input.concurrencyLimit ?? 3;
  log.info('childFanoutWorkflow started', {
    itemCount: input.items.length,
    concurrencyLimit,
  });

  const results: ChildItemResult[] = [];
  let completed = 0;
  let failed = 0;

  // Process items in batches of `concurrencyLimit`
  for (let i = 0; i < input.items.length; i += concurrencyLimit) {
    const batch = input.items.slice(i, i + concurrencyLimit);
    const batchNum = Math.floor(i / concurrencyLimit) + 1;
    log.info(`Processing batch ${batchNum}`, { items: batch });

    // Launch child workflows for this batch in parallel
    const batchPromises = batch.map((item) =>
      executeChild(childProcessItemWorkflow, {
        args: [item],
        workflowId: `child-ts-${item}-${Date.now()}`,
      }).catch((err: Error) => {
        log.warn('Child workflow failed', { item, error: err.message });
        return {
          itemId: item,
          status: `failed: ${err.message}`,
          processedBy: 'ts-worker',
        } as ChildItemResult;
      })
    );

    // Wait for all in this batch to complete before starting next batch
    const batchResults = await Promise.all(batchPromises);

    for (const result of batchResults) {
      results.push(result);
      if (result.status === 'completed') {
        completed++;
      } else {
        failed++;
      }
    }

    log.info(`Batch ${batchNum} complete`, {
      batchCompleted: batchResults.filter((r) => r.status === 'completed').length,
    });
  }

  log.info('childFanoutWorkflow completed', {
    total: input.items.length,
    completed,
    failed,
  });

  return {
    totalItems: input.items.length,
    completed,
    failed,
    results,
  };
}

/**
 * Child Process Item Workflow — processes a single item.
 * Called by childFanoutWorkflow as a child workflow.
 */
export async function childProcessItemWorkflow(itemId: string): Promise<ChildItemResult> {
  log.info('childProcessItemWorkflow started', { itemId });

  const result = await processItem(itemId);

  return {
    itemId,
    status: 'completed',
    processedBy: 'ts-worker',
  };
}
