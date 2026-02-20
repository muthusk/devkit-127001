import { task, wait, logger } from "@trigger.dev/sdk/v3";

/**
 * Human-in-the-loop approval flow using v4 waitpoint tokens.
 *
 * This task pauses and waits for a human to approve or reject.
 * Complete the token via API:
 *
 *   curl -X POST http://localhost:8030/api/v1/waitpoint-tokens/{tokenId}/complete \
 *     -H "Authorization: Bearer <access-token>" \
 *     -H "Content-Type: application/json" \
 *     -d '{"output": {"approved": true, "reviewer": "alice"}}'
 */
export const approvalFlow = task({
  id: "approval-flow",
  retry: { maxAttempts: 1 },
  run: async (payload: {
    title: string;
    description?: string;
    requestedBy?: string;
    timeoutMinutes?: number;
  }) => {
    const timeoutMinutes = payload.timeoutMinutes ?? 60;

    logger.info(`Approval requested`, { title: payload.title });
    logger.info(`⏸️  Pausing for human approval (timeout: ${timeoutMinutes}m)`);

    const token = await wait.createToken({
      id: `approval-${Date.now()}`,
      timeout: `${timeoutMinutes}m`,
      tags: ["approval", payload.requestedBy ?? "unknown"],
    });

    logger.info(`Waitpoint token created`, { tokenId: token.id });

    const result = await wait.forToken<{
      approved: boolean;
      reviewer?: string;
      reason?: string;
    }>(token);

    if (!result.ok) {
      logger.error(`Approval timed out`, { title: payload.title });
      return { status: "timed_out", title: payload.title };
    }

    const decision = result.output;

    if (decision.approved) {
      logger.info(`✅ Approved`, { reviewer: decision.reviewer });
      await new Promise((r) => setTimeout(r, 2000)); // simulate work
      return {
        status: "approved",
        title: payload.title,
        reviewer: decision.reviewer,
        completedAt: new Date().toISOString(),
      };
    } else {
      logger.warn(`❌ Rejected`, { reason: decision.reason });
      return {
        status: "rejected",
        title: payload.title,
        reviewer: decision.reviewer,
        reason: decision.reason,
      };
    }
  },
});

/**
 * Multi-stage approval — requires sequential approvals (e.g. manager → director).
 */
export const multiStageApproval = task({
  id: "multi-stage-approval",
  retry: { maxAttempts: 1 },
  run: async (payload: { title: string; stages: string[] }) => {
    const stages = payload.stages.length > 0 ? payload.stages : ["manager", "director"];
    const approvals: Array<{ stage: string; reviewer?: string; approved: boolean }> = [];

    for (const stage of stages) {
      logger.info(`Waiting for ${stage} approval...`);

      const token = await wait.createToken({
        id: `approval-${stage}-${Date.now()}`,
        timeout: "2h",
        tags: ["multi-approval", stage],
      });

      logger.info(`Token for ${stage}`, { tokenId: token.id });

      const result = await wait.forToken<{
        approved: boolean;
        reviewer?: string;
      }>(token);

      if (!result.ok || !result.output.approved) {
        logger.warn(`Rejected at ${stage} stage`);
        return { status: "rejected", rejectedAt: stage, approvals };
      }

      approvals.push({ stage, reviewer: result.output.reviewer, approved: true });
      logger.info(`${stage} approved ✅`);
    }

    return {
      status: "fully_approved",
      title: payload.title,
      approvals,
      completedAt: new Date().toISOString(),
    };
  },
});
