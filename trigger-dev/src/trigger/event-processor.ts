import { task, queue, logger } from "@trigger.dev/sdk/v3";

const eventQueue = queue({
  id: "event-processing-queue",
  concurrencyLimit: 5,
});

/**
 * Event-driven background job.
 * Trigger from your app: eventProcessor.trigger({ type: "user.signup", data: {...} })
 */
export const eventProcessor = task({
  id: "event-processor",
  queue: eventQueue,
  retry: {
    maxAttempts: 5,
    minTimeoutInMs: 1000,
    maxTimeoutInMs: 30000,
    factor: 2,
  },
  run: async (payload: {
    type: string;
    data: Record<string, unknown>;
  }) => {
    logger.info(`Processing event`, { type: payload.type });

    if (!payload.type || !payload.data) {
      throw new Error("Invalid event payload: type and data are required");
    }

    switch (payload.type) {
      case "user.signup":
        return await handleUserSignup(payload.data);
      case "order.placed":
        return await handleOrderPlaced(payload.data);
      case "file.uploaded":
        return await handleFileUploaded(payload.data);
      default:
        logger.warn(`Unknown event type: ${payload.type}`);
        return { status: "skipped", reason: `Unknown type: ${payload.type}` };
    }
  },
});

async function handleUserSignup(data: Record<string, unknown>) {
  logger.info("Handling user signup", { email: data.email });
  await new Promise((r) => setTimeout(r, 2000));
  return { status: "processed", action: "welcome_email_sent", email: data.email };
}

async function handleOrderPlaced(data: Record<string, unknown>) {
  logger.info("Handling order placed", { orderId: data.orderId });
  await new Promise((r) => setTimeout(r, 3000));
  return { status: "processed", action: "order_confirmed", orderId: data.orderId };
}

async function handleFileUploaded(data: Record<string, unknown>) {
  logger.info("Handling file upload", { fileName: data.fileName });
  await new Promise((r) => setTimeout(r, 1500));
  return { status: "processed", action: "file_indexed", fileName: data.fileName };
}

/**
 * Batch processor — triggers multiple event runs at once.
 */
export const batchEventProcessor = task({
  id: "batch-event-processor",
  run: async (payload: {
    events: Array<{ type: string; data: Record<string, unknown> }>;
  }) => {
    logger.info(`Batch processing ${payload.events.length} events`);

    const handles = [];
    for (const event of payload.events) {
      const handle = await eventProcessor.trigger(event);
      handles.push(handle);
    }

    return { triggered: handles.length };
  },
});
