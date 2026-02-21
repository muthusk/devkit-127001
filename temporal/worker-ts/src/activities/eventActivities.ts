import { log } from '@temporalio/activity';

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export async function validateEvent(eventType: string, eventId: string): Promise<string> {
  log.info('validateEvent', { type: eventType, id: eventId });
  await delay(500);

  if (!eventType || !eventId) {
    throw new Error('Invalid event: type and id are required');
  }

  return `validated:${eventType}:${eventId}`;
}

/**
 * enrichEvent — TS-specific activity that runs in parallel with validateEvent.
 * Simulates looking up metadata or enriching the event payload.
 */
export async function enrichEvent(eventType: string, eventId: string): Promise<string> {
  log.info('enrichEvent', { type: eventType, id: eventId });
  await delay(700); // Slightly longer than validate to show parallel benefit

  return `enriched:${eventType}:${eventId}`;
}

export async function processEvent(eventType: string, eventId: string): Promise<string> {
  log.info('processEvent', { type: eventType, id: eventId });
  await delay(1000);

  return `processed:${eventType}:${eventId}`;
}

export async function notifyEventProcessed(eventType: string, eventId: string): Promise<string> {
  log.info('notifyEventProcessed', { type: eventType, id: eventId });
  await delay(300);

  return `notified:${eventType}:${eventId}`;
}
