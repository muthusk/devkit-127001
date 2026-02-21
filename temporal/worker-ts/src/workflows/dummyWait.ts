import { sleep, log, CancellationScope, CancelledFailure } from '@temporalio/workflow';

export interface DummyWaitResult {
  requestedDuration: string;
  status: string;
  startedAt: string;
  completedAt: string;
  checkpoints: string[];
}

/**
 * Dummy Wait Workflow — TypeScript Variant
 *
 * DIFFERENCE FROM GO: Multi-stage wait with checkpoints.
 * Instead of a single sleep, this splits the duration into stages,
 * logging a checkpoint between each one. Shows timer + state tracking.
 */
export async function dummyWaitWorkflow(durationStr: string): Promise<DummyWaitResult> {
  const startedAt = new Date().toISOString();
  const checkpoints: string[] = [];

  // Parse duration string (e.g., "30s", "2m")
  const totalMs = parseDuration(durationStr);
  const stages = 3;
  const stageMs = Math.floor(totalMs / stages);

  log.info('dummyWaitWorkflow started', { duration: durationStr, stages, stageMs });
  checkpoints.push(`started:${startedAt}`);

  try {
    for (let i = 1; i <= stages; i++) {
      log.info(`Stage ${i}/${stages}: sleeping for ${stageMs}ms`);
      await sleep(stageMs);

      const checkpoint = `stage-${i}-complete:${new Date().toISOString()}`;
      checkpoints.push(checkpoint);
      log.info(`Checkpoint: ${checkpoint}`);
    }

    // Handle any remainder
    const remainder = totalMs - stageMs * stages;
    if (remainder > 0) {
      await sleep(remainder);
    }

    const completedAt = new Date().toISOString();
    checkpoints.push(`completed:${completedAt}`);
    log.info('dummyWaitWorkflow completed', { duration: durationStr });

    return {
      requestedDuration: durationStr,
      status: 'completed',
      startedAt,
      completedAt,
      checkpoints,
    };
  } catch (err) {
    if (err instanceof CancelledFailure) {
      log.info('dummyWaitWorkflow was cancelled during sleep');
      return {
        requestedDuration: durationStr,
        status: 'cancelled',
        startedAt,
        completedAt: new Date().toISOString(),
        checkpoints,
      };
    }
    throw err;
  }
}

function parseDuration(s: string): number {
  const match = s.match(/^(\d+)(ms|s|m|h)$/);
  if (!match) throw new Error(`Invalid duration: ${s}`);
  const value = parseInt(match[1], 10);
  const unit = match[2];
  switch (unit) {
    case 'ms':
      return value;
    case 's':
      return value * 1000;
    case 'm':
      return value * 60 * 1000;
    case 'h':
      return value * 60 * 60 * 1000;
    default:
      throw new Error(`Unknown duration unit: ${unit}`);
  }
}
