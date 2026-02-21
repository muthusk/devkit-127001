import { log } from '@temporalio/activity';

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * processItem — simulates processing a single item with variable duration.
 */
export async function processItem(itemId: string): Promise<string> {
  log.info('processItem', { itemId });

  // Simulate variable processing time (1-3 seconds)
  const processingTime = 1000 + Math.floor(Math.random() * 2000);
  await delay(processingTime);

  log.info(`✅ Item processed: ${itemId} (${processingTime}ms)`);
  return `processed:${itemId}`;
}
