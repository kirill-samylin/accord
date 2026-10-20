export const READ_DEADLINE_MS = 8000;
// The same signal covers headers and body parsing. Completion removes both
// timers and parent listeners, so a later cancellation cannot affect this read.
export async function withReadDeadline<T>(
  parent: AbortSignal | undefined,
  operation: (signal: AbortSignal) => Promise<T>,
  timeoutMs = READ_DEADLINE_MS,
): Promise<T> {
  const controller = new AbortController();
  const cancel = () => controller.abort(parent?.reason);
  if (parent?.aborted) cancel();
  else parent?.addEventListener("abort", cancel, { once: true });
  const timer = setTimeout(
    () =>
      controller.abort(
        new DOMException("Истекло время чтения", "TimeoutError"),
      ),
    timeoutMs,
  );
  try {
    controller.signal.throwIfAborted();
    return await operation(controller.signal);
  } finally {
    clearTimeout(timer);
    parent?.removeEventListener("abort", cancel);
  }
}
