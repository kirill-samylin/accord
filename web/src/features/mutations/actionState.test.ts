import assert from "node:assert/strict";
import test from "node:test";
import type { PageData, SessionView, Task } from "../../lib/types.ts";
import { applyActionResult } from "./actionState.ts";

const task = {
  key: "T-7",
  generation: 3,
  lease: {
    session_id: "session-1",
    generation: 3,
    expires_at: "2026-10-20T16:00:00Z",
  },
} as Task;
const page = {
  csrf: "csrf-fixture",
  actions: { answer: "original", "heartbeat:T-7": "old-heartbeat" },
  task,
  can_edit: true,
  draft: { answer: "original-draft" },
} as unknown as PageData;

test("uncertain response cannot replace the pending action or adopt fresh server capabilities", () => {
  const state = applyActionResult(page, {
    ok: false,
    retry_same_action: true,
    actions: { answer: "new-id" },
    page: { ...page, task: { ...task, generation: 4 } },
  });
  assert.equal(state.phase, "uncertain");
  assert.equal(state.keepPending, true);
  assert.equal(state.page, page);
});

test("confirmed validation adopts newly issued IDs and keeps server task ownership", () => {
  const state = applyActionResult(page, {
    ok: false,
    error: { code: "VALIDATION", message: "fixture" },
    actions: { answer: "correctable-id" },
    page: { ...page, can_edit: true },
  });
  assert.equal(state.phase, "idle");
  assert.equal(state.keepPending, false);
  assert.equal(state.page.actions.answer, "correctable-id");
});

test("confirmed lease loss blocks writes and applies the server ownership", () => {
  const state = applyActionResult(page, {
    ok: false,
    error: { code: "LEASE_LOST", message: "fixture" },
    page: { ...page, can_edit: false, task: { ...task, generation: 4 } },
  });
  assert.equal(state.phase, "blocked");
  assert.equal(state.page.can_edit, false);
  assert.equal(state.page.task?.generation, 4);
  assert.equal(state.keepPending, false);
});

test("heartbeat updates the current lease and both aliases, leaving unrelated actions unchanged", () => {
  const updated = {
    ...task,
    lease: { ...task.lease!, expires_at: "2026-10-20T17:00:00Z" },
  };
  const session = {
    session_id: "session-1",
    my_tasks: [updated],
  } as SessionView;
  const state = applyActionResult(page, {
    ok: true,
    session,
    next_action_id: "heartbeat-next",
  });
  assert.equal(state.phase, "idle");
  assert.equal(state.page.task?.lease?.expires_at, updated.lease.expires_at);
  assert.equal(state.page.actions["heartbeat:T-7"], "heartbeat-next");
  assert.equal(state.page.actions.heartbeat, "heartbeat-next");
  assert.equal(state.page.actions.answer, "original");
});

test("a successful redirect settles the form until the new page capabilities arrive", () => {
  assert.equal(
    applyActionResult(page, { ok: true, redirect: "/web/tasks/T-7" }).phase,
    "settled",
  );
});
