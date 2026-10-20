import { test } from "node:test";
import assert from "node:assert/strict";
import {
  DashboardReadError,
  readDashboard,
  retryDelay,
} from "./dashboard-polling.ts";
import { withReadDeadline } from "./read-deadline.ts";
function waitForAbort(signal: AbortSignal): Promise<never> {
  return new Promise((_, reject) => {
    if (signal.aborted) reject(signal.reason);
    else
      signal.addEventListener("abort", () => reject(signal.reason), {
        once: true,
      });
  });
}
test("a stalled read aborts at 8 seconds, including body parsing after successful headers", async (t) => {
  t.mock.timers.enable({ apis: ["setTimeout"] });
  let supplied: AbortSignal | undefined;
  const outcome = withReadDeadline(undefined, async (signal) => {
    supplied = signal;
    return waitForAbort(signal);
  });
  const rejection = assert.rejects(
    outcome,
    (error: unknown) =>
      error instanceof DOMException && error.name === "TimeoutError",
  );
  t.mock.timers.tick(7999);
  assert.equal(supplied?.aborted, false);
  t.mock.timers.tick(1);
  assert.equal(supplied?.aborted, true);
  await rejection;
});
test("query cancellation wins before the deadline and completed reads detach parent cancellation", async (t) => {
  t.mock.timers.enable({ apis: ["setTimeout"] });
  const parent = new AbortController();
  let active: AbortSignal | undefined;
  const outcome = withReadDeadline(parent.signal, async (signal) => {
    active = signal;
    return waitForAbort(signal);
  });
  const reason = new DOMException(
    "Hidden/offline/unmounted query",
    "AbortError",
  );
  const rejection = assert.rejects(outcome, (error) => error === reason);
  t.mock.timers.tick(1200);
  parent.abort(reason);
  await rejection;
  assert.equal(active?.reason, reason);
  const completedParent = new AbortController();
  let completedSignal: AbortSignal | undefined;
  assert.equal(
    await withReadDeadline(completedParent.signal, async (signal) => {
      completedSignal = signal;
      return "last good";
    }),
    "last good",
  );
  completedParent.abort();
  t.mock.timers.tick(10000);
  assert.equal(completedSignal?.aborted, false);
});
test("already cancelled queries never start a request", async () => {
  const parent = new AbortController();
  parent.abort();
  let calls = 0;
  await assert.rejects(
    withReadDeadline(parent.signal, async () => {
      calls++;
      return null;
    }),
  );
  assert.equal(calls, 0);
});
test("backoff grows after consecutive failures and server cooldown is respected within bounds", () => {
  assert.deepEqual(
    [0, 1, 2, 3, 4, 20].map((count) => retryDelay(count)),
    [15000, 30000, 60000, 120000, 120000, 120000],
  );
  assert.equal(retryDelay(1, 429, "90"), 90000);
  assert.equal(retryDelay(1, 503, "999999"), 120000);
  assert.equal(retryDelay(1, 503, "1"), 30000);
  assert.equal(retryDelay(1, 500, "90"), 30000);
  assert.equal(retryDelay(1, 429, "invalid"), 30000);
  assert.equal(retryDelay(1, 503, "-1"), 30000);
  const now = Date.parse("2026-10-20T10:00:00Z");
  assert.equal(retryDelay(1, 503, "Tue, 20 Oct 2026 10:01:00 GMT", now), 60000);
});
test("dashboard cooldown and lease loss preserve the domain error without manufacturing empty data", async (t) => {
  const original = globalThis.fetch;
  try {
    globalThis.fetch = async () =>
      Response.json(
        { code: "UNAVAILABLE", error: "Try later" },
        { status: 503, headers: { "Retry-After": "75" } },
      );
    await assert.rejects(
      readDashboard(new AbortController().signal),
      (error) =>
        error instanceof DashboardReadError &&
        error.status === 503 &&
        error.retryAfter === "75",
    );
    globalThis.fetch = async () =>
      Response.json(
        { code: "LEASE_LOST", error: "Continue explicitly" },
        { status: 409 },
      );
    await assert.rejects(
      readDashboard(new AbortController().signal),
      (error) =>
        error instanceof DashboardReadError &&
        error.status === 409 &&
        error.result.error?.code === "LEASE_LOST",
    );
    globalThis.fetch = async () =>
      Response.json({
        overview: { tasks_total: 0 },
        agents: { sessions: [] },
        activity: { events: [] },
      });
    await assert.rejects(
      readDashboard(new AbortController().signal),
      (error) =>
        error instanceof DashboardReadError &&
        error.result.error?.code === "INVALID_RESPONSE",
    );
    t.mock.timers.enable({ apis: ["setTimeout"] });
    globalThis.fetch = async (_path, init) => {
      const response = Response.json({});
      response.json = () => waitForAbort(init!.signal!);
      return response;
    };
    const pending = readDashboard(new AbortController().signal);
    const rejection = assert.rejects(
      pending,
      (error) => error instanceof DOMException && error.name === "TimeoutError",
    );
    await Promise.resolve();
    t.mock.timers.tick(8000);
    await rejection;
  } finally {
    globalThis.fetch = original;
  }
});
