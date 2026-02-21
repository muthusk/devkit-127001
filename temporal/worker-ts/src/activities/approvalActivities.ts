import { log } from '@temporalio/activity';

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export async function sendApprovalRequest(requester: string, item: string): Promise<void> {
  log.info('sendApprovalRequest', { requester, item });
  await delay(500);
  log.info(`📨 Approval request sent: ${requester} is requesting approval for '${item}'`);
}

export async function notifyApprovalResult(
  status: string,
  item: string,
  requester: string
): Promise<void> {
  log.info('notifyApprovalResult', { status, item, requester });
  await delay(300);
  log.info(`📬 Approval result: '${item}' was ${status} (requested by ${requester})`);
}
