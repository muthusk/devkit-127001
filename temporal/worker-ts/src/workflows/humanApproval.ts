import {
  proxyActivities,
  defineSignal,
  defineQuery,
  setHandler,
  condition,
  log,
} from '@temporalio/workflow';
import type * as activities from '../activities/approvalActivities';

const { sendApprovalRequest, notifyApprovalResult } = proxyActivities<typeof activities>({
  startToCloseTimeout: '15s',
  retry: { maximumAttempts: 3 },
});

// Signal and Query definitions
export const approvalSignal = defineSignal<[ApprovalSignalInput]>('approval');
export const approvalStatusQuery = defineQuery<ApprovalStatus>('approvalStatus');

export interface ApprovalInput {
  requester: string;
  item: string;
  timeout: string; // e.g., "5m"
}

export interface ApprovalSignalInput {
  approved: boolean;
  approver: string;
  reason: string;
}

export interface ApprovalStatus {
  item: string;
  requester: string;
  status: 'pending' | 'approved' | 'rejected' | 'timed_out';
  approver?: string;
  reason?: string;
  waitingSince: string;
}

export interface ApprovalResult {
  item: string;
  requester: string;
  status: string;
  approver?: string;
  reason?: string;
}

/**
 * Human Approval Workflow — TypeScript Variant
 *
 * DIFFERENCE FROM GO: Adds a QUERY handler so you can check the approval
 * status at any time without looking at the UI. Use:
 *   temporal workflow query --workflow-id <id> --name approvalStatus
 */
export async function humanApprovalWorkflow(input: ApprovalInput): Promise<ApprovalResult> {
  log.info('humanApprovalWorkflow started', {
    requester: input.requester,
    item: input.item,
    timeout: input.timeout,
  });

  // Internal state
  let signalReceived = false;
  let signalData: ApprovalSignalInput | undefined;
  const waitingSince = new Date().toISOString();
  let currentStatus: ApprovalStatus = {
    item: input.item,
    requester: input.requester,
    status: 'pending',
    waitingSince,
  };

  // Register signal handler
  setHandler(approvalSignal, (signal: ApprovalSignalInput) => {
    log.info('Approval signal received', { approved: signal.approved, approver: signal.approver });
    signalReceived = true;
    signalData = signal;
    currentStatus = {
      ...currentStatus,
      status: signal.approved ? 'approved' : 'rejected',
      approver: signal.approver,
      reason: signal.reason,
    };
  });

  // Register query handler — this is the TS-specific addition
  setHandler(approvalStatusQuery, () => currentStatus);

  // Step 1: Send notification that approval is needed
  await sendApprovalRequest(input.requester, input.item);
  log.info('Approval request sent', { item: input.item });

  // Step 2: Wait for signal with timeout
  const timeoutMs = parseDuration(input.timeout);
  const received = await condition(() => signalReceived, timeoutMs);

  // Step 3: Handle result
  let result: ApprovalResult;

  if (!received) {
    log.info('Approval timed out', { item: input.item, timeout: input.timeout });
    currentStatus.status = 'timed_out';
    result = {
      item: input.item,
      requester: input.requester,
      status: 'timed_out',
      reason: `No response within ${input.timeout}`,
    };
  } else if (signalData?.approved) {
    result = {
      item: input.item,
      requester: input.requester,
      status: 'approved',
      approver: signalData.approver,
      reason: signalData.reason,
    };
  } else {
    result = {
      item: input.item,
      requester: input.requester,
      status: 'rejected',
      approver: signalData?.approver,
      reason: signalData?.reason,
    };
  }

  // Step 4: Notify outcome
  try {
    await notifyApprovalResult(result.status, input.item, input.requester);
  } catch (err) {
    log.warn('Failed to send result notification', { error: String(err) });
  }

  return result;
}

function parseDuration(s: string): number {
  const match = s.match(/^(\d+)(ms|s|m|h)$/);
  if (!match) return 10 * 60 * 1000; // default 10m
  const value = parseInt(match[1], 10);
  switch (match[2]) {
    case 'ms': return value;
    case 's': return value * 1000;
    case 'm': return value * 60 * 1000;
    case 'h': return value * 3600 * 1000;
    default: return 10 * 60 * 1000;
  }
}
