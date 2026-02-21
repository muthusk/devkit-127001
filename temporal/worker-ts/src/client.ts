/**
 * client.ts — Optional programmatic workflow starter
 *
 * Usage:
 *   npx ts-node src/client.ts <workflow-name> [json-input]
 *
 * Examples:
 *   npx ts-node src/client.ts eventProcessingWorkflow '{"type":"test","id":"1","timestamp":"now"}'
 *   npx ts-node src/client.ts dummyWaitWorkflow '"10s"'
 */

import { Client, Connection } from '@temporalio/client';

async function main() {
  const workflowName = process.argv[2];
  const inputStr = process.argv[3] || '{}';

  if (!workflowName) {
    console.error('Usage: npx ts-node src/client.ts <workflow-name> [json-input]');
    console.error('');
    console.error('Available workflows:');
    console.error('  eventProcessingWorkflow');
    console.error('  dummyWaitWorkflow');
    console.error('  humanApprovalWorkflow');
    console.error('  sagaWorkflow');
    console.error('  childFanoutWorkflow');
    process.exit(1);
  }

  const address = process.env.TEMPORAL_ADDRESS || 'localhost:7233';
  const taskQueue = process.env.TEMPORAL_TASK_QUEUE || 'ts-task-queue';

  const connection = await Connection.connect({ address });
  const client = new Client({ connection });

  let input: unknown;
  try {
    input = JSON.parse(inputStr);
  } catch {
    console.error(`Invalid JSON input: ${inputStr}`);
    process.exit(1);
  }

  const workflowId = `${workflowName}-${Date.now()}`;

  console.log(`Starting workflow: ${workflowName}`);
  console.log(`  Task Queue: ${taskQueue}`);
  console.log(`  Workflow ID: ${workflowId}`);
  console.log(`  Input: ${JSON.stringify(input)}`);

  const handle = await client.workflow.start(workflowName, {
    taskQueue,
    workflowId,
    args: [input],
  });

  console.log(`\nWorkflow started!`);
  console.log(`  Workflow ID: ${handle.workflowId}`);
  console.log(`  Run ID: ${handle.firstExecutionRunId}`);

  // Optionally wait for result
  console.log('\nWaiting for result...');
  const result = await handle.result();
  console.log('Result:', JSON.stringify(result, null, 2));
}

main().catch((err) => {
  console.error('Error:', err);
  process.exit(1);
});
