import { NativeConnection, Worker } from '@temporalio/worker';
import * as activities from './activities/eventActivities';
import * as approvalActivities from './activities/approvalActivities';
import * as sagaActivities from './activities/sagaActivities';
import * as childActivities from './activities/childActivities';

async function run() {
  const address = process.env.TEMPORAL_ADDRESS || 'localhost:7233';
  const namespace = process.env.TEMPORAL_NAMESPACE || 'default';
  const taskQueue = process.env.TEMPORAL_TASK_QUEUE || 'ts-task-queue';

  console.log(`Starting TypeScript worker...`);
  console.log(`  Temporal:  ${address}`);
  console.log(`  Namespace: ${namespace}`);
  console.log(`  TaskQueue: ${taskQueue}`);

  // Connect to Temporal server with retry
  let connection: NativeConnection | undefined;
  const maxRetries = 30;
  for (let i = 1; i <= maxRetries; i++) {
    try {
      connection = await NativeConnection.connect({ address });
      console.log(`Connected to Temporal server (attempt ${i})`);
      break;
    } catch (err) {
      console.log(`Connection attempt ${i}/${maxRetries} failed, retrying in 3s...`);
      await new Promise((resolve) => setTimeout(resolve, 3000));
    }
  }

  if (!connection) {
    throw new Error(`Failed to connect to Temporal after ${maxRetries} attempts`);
  }

  // Create and start the worker
  const worker = await Worker.create({
    connection,
    namespace,
    taskQueue,
    workflowsPath: require.resolve('./workflows'),
    activities: {
      ...activities,
      ...approvalActivities,
      ...sagaActivities,
      ...childActivities,
    },
  });

  console.log(`Worker started. Listening on task queue: ${taskQueue}`);
  await worker.run();
}

run().catch((err) => {
  console.error('Worker failed:', err);
  process.exit(1);
});
