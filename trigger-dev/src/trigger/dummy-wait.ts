import { task, wait, logger } from "@trigger.dev/sdk/v3";

/**
 * Dummy task that simply waits for a configurable duration.
 * Useful for testing the infrastructure, verifying timeouts,
 * and observing task lifecycle in the dashboard.
 */
export const dummyWait = task({
  id: "dummy-wait",
  retry: { maxAttempts: 1 },
  run: async (payload: { seconds?: number; label?: string }) => {
    const seconds = payload.seconds ?? 10;
    const label = payload.label ?? "default";

    logger.info(`Starting dummy wait`, { label, seconds });
    await wait.for({ seconds });
    logger.info(`Wait complete`, { label, seconds });

    return {
      label,
      waitedSeconds: seconds,
      completedAt: new Date().toISOString(),
    };
  },
});

/**
 * A chain of waits — useful for testing multi-step task visibility
 * in the dashboard trace view.
 */
export const multiStepWait = task({
  id: "multi-step-wait",
  retry: { maxAttempts: 1 },
  run: async (payload: { steps?: number; secondsPerStep?: number }) => {
    const steps = payload.steps ?? 3;
    const secondsPerStep = payload.secondsPerStep ?? 5;
    const results: string[] = [];

    for (let i = 1; i <= steps; i++) {
      logger.info(`Step ${i}/${steps}: waiting ${secondsPerStep}s...`);
      await wait.for({ seconds: secondsPerStep });
      results.push(`Step ${i} done at ${new Date().toISOString()}`);
    }

    logger.info(`All ${steps} steps complete`);
    return { steps: results };
  },
});
