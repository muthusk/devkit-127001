import { log } from '@temporalio/activity';

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

// --- Forward steps ---

export async function reserveInventory(orderId: string, item: string): Promise<string> {
  log.info('reserveInventory', { orderId, item });
  await delay(800);
  log.info(`📦 Inventory reserved: ${item} for order ${orderId}`);
  return `reserved:${item}:${orderId}`;
}

export async function chargePayment(orderId: string, amount: number): Promise<string> {
  log.info('chargePayment', { orderId, amount });
  await delay(1000);
  log.info(`💳 Payment charged: $${amount.toFixed(2)} for order ${orderId}`);
  return `charged:${amount.toFixed(2)}:${orderId}`;
}

export async function arrangeShipping(orderId: string, item: string): Promise<string> {
  log.info('arrangeShipping', { orderId, item });
  await delay(1000);
  log.info(`🚚 Shipping arranged: ${item} for order ${orderId}`);
  return `shipping-arranged:${item}:${orderId}`;
}

/**
 * sendConfirmation — has a ~30% failure rate to trigger compensations.
 */
export async function sendConfirmation(
  orderId: string,
  item: string,
  amount: number
): Promise<string> {
  log.info('sendConfirmation', { orderId, item, amount });
  await delay(500);

  // Simulate occasional failure (~30% chance)
  if (Math.random() < 0.3) {
    log.warn('sendConfirmation FAILED (simulated)', { orderId });
    throw new Error(`Confirmation service unavailable for order ${orderId}`);
  }

  log.info(`✉️ Confirmation sent: order ${orderId}, ${item}, $${amount.toFixed(2)}`);
  return `confirmed:${orderId}`;
}

// --- Compensation steps ---

export async function cancelReservation(orderId: string, item: string): Promise<string> {
  log.info('cancelReservation (compensation)', { orderId, item });
  await delay(500);
  log.info(`↩️ Reservation cancelled: ${item} for order ${orderId}`);
  return `cancelled-reservation:${item}:${orderId}`;
}

export async function refundPayment(orderId: string, amount: number): Promise<string> {
  log.info('refundPayment (compensation)', { orderId, amount });
  await delay(500);
  log.info(`💸 Payment refunded: $${amount.toFixed(2)} for order ${orderId}`);
  return `refunded:${amount.toFixed(2)}:${orderId}`;
}

/**
 * cancelShipping — TS-specific: this compensation itself has a ~20% failure chance.
 * Demonstrates that compensations can fail too, and the saga handles it gracefully.
 */
export async function cancelShipping(orderId: string, item: string): Promise<string> {
  log.info('cancelShipping (compensation)', { orderId, item });
  await delay(500);

  // Simulate occasional compensation failure
  if (Math.random() < 0.2) {
    log.error('cancelShipping FAILED (simulated)', { orderId });
    throw new Error(`Shipping cancellation failed for order ${orderId} — manual intervention needed`);
  }

  log.info(`🚫 Shipping cancelled: ${item} for order ${orderId}`);
  return `cancelled-shipping:${item}:${orderId}`;
}
