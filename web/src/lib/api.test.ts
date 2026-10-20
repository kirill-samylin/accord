import { afterEach, test } from "node:test";
import assert from "node:assert/strict";
import { APIError, appToWeb, readPage, submitAction, webToApp } from "./api.ts";
import { validateGoOrigin } from "./upstream.ts";
import { groupTasks } from "./task-groups.ts";
import type { Task } from "./types.ts";
const originalFetch = globalThis.fetch;
afterEach(() => {
  globalThis.fetch = originalFetch;
});
test("only screen routes map; binary downloads and external redirects cannot enter router", () => {
  assert.equal(
    webToApp("/web/tasks/T-7?layout=board"),
    "/app/tasks/T-7?layout=board",
  );
  assert.equal(appToWeb("/app"), "/web/");
  assert.equal(appToWeb("/app/history?cursor=13"), "/web/history?cursor=13");
  for (const path of [
    "https://evil.invalid/web/tasks",
    "//evil.invalid/web",
    "/web/designs/asset/download",
    "/web\\evil",
    "/web/tasks/T-1/answer",
    "/webnot-a-route",
  ])
    assert.throws(() => webToApp(path));
});
test("deployment upstream accepts explicit literal loopback and rejects credential/path/remote forwarding", () => {
  assert.equal(
    validateGoOrigin("http://127.0.0.1:3101"),
    "http://127.0.0.1:3101",
  );
  assert.equal(validateGoOrigin("http://[::1]:3101"), "http://[::1]:3101");
  for (const value of [
    "https://127.0.0.1:3101",
    "http://localhost:3101",
    "http://127.0.0.1",
    "http://127.0.0.1:3101/api",
    "http://u:p@127.0.0.1:3101",
    "http://example.com:3101",
    "http://127.0.0.1:3101?target=elsewhere",
  ])
    assert.throws(() => validateGoOrigin(value));
});
test("page reads explicitly negotiate JSON with same-origin cookie and no cache", async () => {
  const controller = new AbortController();
  globalThis.fetch = async (input, init) => {
    assert.equal(input, "/web/tasks?state=ready");
    assert.equal(init?.cache, "no-store");
    assert.equal(init?.credentials, "same-origin");
    assert.equal(init?.redirect, "error");
    assert.equal(init?.signal, controller.signal);
    assert.equal(new Headers(init?.headers).get("Accept"), "application/json");
    return Response.json({ ok: true, page: { view: "tasks", tasks: [] } });
  };
  assert.equal(
    (await readPage("/app/tasks?state=ready", controller.signal)).view,
    "tasks",
  );
});
test("read failures retain status and public domain code for revocation/session handling", async () => {
  globalThis.fetch = async () =>
    Response.json(
      {
        ok: false,
        error: { code: "LEASE_LOST", message: "Continue explicitly" },
      },
      { status: 409 },
    );
  await assert.rejects(
    readPage("/app"),
    (error: unknown) =>
      error instanceof APIError &&
      error.status === 409 &&
      error.result.error?.code === "LEASE_LOST",
  );
});
test("mutations preserve exact supplied body and domain conflicts do not become success", async () => {
  const body = new URLSearchParams({
    csrf: "fixture",
    action_id: "same-id",
    generation: "7",
    answer: "Keep my draft",
  });
  globalThis.fetch = async (input, init) => {
    assert.equal(input, "/web/tasks/T-1/answer");
    assert.equal(init?.body, body);
    return Response.json(
      {
        ok: false,
        error: { code: "LEASE_LOST", message: "Lease lost" },
        retry_same_action: false,
      },
      { status: 409 },
    );
  };
  const result = await submitAction("/web/tasks/T-1/answer", body);
  assert.equal(result.ok, false);
  assert.equal(result.error?.code, "LEASE_LOST");
});
test("non-JSON mutation outcome is uncertain and keeps exact replay requirement", async () => {
  globalThis.fetch = async () =>
    new Response("<h1>Gateway timeout</h1>", { status: 503 });
  await assert.rejects(
    submitAction("/web/tasks", new URLSearchParams()),
    (error: unknown) =>
      error instanceof APIError && error.result.retry_same_action === true,
  );
});
test("open epics and unfamiliar server states remain visible exactly once", () => {
  const tasks = [
    { key: "E-1", state: "open" },
    { key: "E-2", state: "done" },
    { key: "E-3", state: "future-state" },
  ] as Task[];
  const groups = groupTasks(tasks, [
    "created",
    "ready",
    "in_progress",
    "done",
    "cancelled",
  ]);
  assert.equal(groups.flatMap((group) => group.tasks).length, 3);
  assert.equal(
    groups.find((group) => group.state === "open")?.tasks[0].key,
    "E-1",
  );
  assert.equal(
    groups.find((group) => group.state === "future-state")?.tasks[0].key,
    "E-3",
  );
});
test("revoked background page reads notify the identity cache boundary", async () => {
  let events = 0;
  const descriptor = Object.getOwnPropertyDescriptor(globalThis, "window");
  Object.defineProperty(globalThis, "window", {
    configurable: true,
    value: {
      dispatchEvent: (event: Event) => {
        assert.equal(event.type, "accord:unauthorized");
        events++;
      },
    },
  });
  try {
    globalThis.fetch = async () =>
      Response.json(
        { ok: false, error: { code: "AUTH_REQUIRED", message: "Sign in" } },
        { status: 401 },
      );
    await assert.rejects(readPage("/app/tasks?q=background"));
    assert.equal(events, 1);
  } finally {
    if (descriptor) Object.defineProperty(globalThis, "window", descriptor);
    else Reflect.deleteProperty(globalThis, "window");
  }
});
