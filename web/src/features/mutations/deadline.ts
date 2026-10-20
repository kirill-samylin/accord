// Aborting a POST cannot establish whether Go committed it. Callers must retain
// the original payload and offer an explicit same-action retry after rejection.
export async function withMutationDeadline<T>(
  path: string,
  operation: (signal: AbortSignal) => Promise<T>,
): Promise<T> {
  const controller = new AbortController();
  const timeout = setTimeout(
    () => controller.abort(),
    path === "/web/designs" ? 130_000 : 15_000,
  );
  try {
    return await operation(controller.signal);
  } finally {
    clearTimeout(timeout);
  }
}
