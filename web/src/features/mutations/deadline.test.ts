import assert from "node:assert/strict";
import test from "node:test";
import { withMutationDeadline } from "./deadline.ts";
import { APIError, submitAction } from "../../lib/api.ts";

test("stalled ordinary action aborts at 15 seconds without an automatic retry", async (t) => {
  t.mock.timers.enable({ apis: ["setTimeout"] });
  let calls = 0;
  let signal: AbortSignal;
  const result = withMutationDeadline("/web/tasks", (incoming) => {
    calls++;
    signal = incoming;
    return new Promise((_, reject) => {
      incoming.addEventListener("abort", () => reject(new Error("aborted")));
    });
  });
  const rejected = assert.rejects(result, /aborted/);
  t.mock.timers.tick(14_999);
  assert.equal(signal!.aborted, false);
  t.mock.timers.tick(1);
  await rejected;
  assert.equal(calls, 1);
});

test("upload remains available beyond 30 seconds but cannot stay locked forever", async (t) => {
  t.mock.timers.enable({ apis: ["setTimeout"] });
  let signal: AbortSignal;
  const result = withMutationDeadline("/web/designs", (incoming) => {
    signal = incoming;
    return new Promise((_, reject) => {
      incoming.addEventListener("abort", () => reject(new Error("aborted")));
    });
  });
  const rejected = assert.rejects(result, /aborted/);
  t.mock.timers.tick(129_999);
  assert.equal(signal!.aborted, false);
  t.mock.timers.tick(1);
  await rejected;
});

test("successful action clears its deadline before the next request", async (t) => {
  t.mock.timers.enable({ apis: ["setTimeout"] });
  let signal: AbortSignal;
  const result = await withMutationDeadline(
    "/web/heartbeat",
    async (incoming) => {
      signal = incoming;
      await Promise.resolve();
      return { ok: true, next_action_id: "fresh" };
    },
  );
  t.mock.timers.tick(130_000);
  assert.equal(signal!.aborted, false);
  assert.equal(result.next_action_id, "fresh");
});

test("received headers cannot bypass the body deadline or replace the pending action", async (t) => {
  t.mock.timers.enable({ apis: ["setTimeout"] });
  const original = globalThis.fetch;
  t.after(() => {
    globalThis.fetch = original;
  });
  const body = new URLSearchParams({
    action_id: "immutable",
    title: "Черновик",
  });
  const sent: BodyInit[] = [];
  globalThis.fetch = async (_url, options) => {
    sent.push(options!.body!);
    return {
      status: 200,
      ok: true,
      json: () =>
        sent.length === 1
          ? new Promise((_, reject) => {
              options!.signal!.addEventListener("abort", () =>
                reject(new Error("response body aborted")),
              );
            })
          : Promise.resolve({ ok: true, redirect: "/web/tasks/T-1" }),
    } as Response;
  };
  const send = () =>
    withMutationDeadline("/web/tasks", (signal) =>
      submitAction("/web/tasks", body, signal),
    );
  const pending = send();
  // Headers have arrived, but the server's response body is still incomplete.
  await Promise.resolve();
  const rejected = assert.rejects(
    pending,
    (error) =>
      error instanceof APIError &&
      error.result.error?.code === "UNKNOWN_RESULT" &&
      error.result.retry_same_action === true,
  );
  t.mock.timers.tick(15_000);
  await rejected;
  assert.equal(sent.length, 1);
  assert.equal((await send()).ok, true); // Only an explicit retry sends again.
  assert.equal(sent.length, 2);
  assert.equal(sent[0], sent[1]);
  assert.equal(body.get("action_id"), "immutable");
});
