import assert from "node:assert/strict";
import { afterEach, test } from "node:test";
import {
  assignmentText,
  canCoordinate,
  coordinationDraft,
  coordinationFields,
  currentProfileChoice,
  latestPositiveReview,
  profilePair,
  transitionChoices,
  type CoordinationCheck,
  type TaskCoordination,
} from "./coordination.ts";
import { readPage, submitAction } from "./api.ts";
import { snapshotPayload } from "../features/mutations/contracts.ts";
import { applyActionResult } from "../features/mutations/actionState.ts";
import type { ProfileSummary } from "./agent-profiles.ts";
import type { PageData, Task } from "./types.ts";
const profileID = `ap_${"1".repeat(32)}`;
const profile = {
  id: profileID,
  version: 3,
  role: "developer",
  state: "active",
} as ProfileSummary;
const task = {
  key: "T-2",
  type: "task",
  version: 7,
  generation: 2,
  role: "developer",
  state: "ready",
  evidence: {},
  assignment: {
    profile_id: profileID,
    profile_version: 3,
    profile_name: "Разработчик",
    role: "developer",
  },
} as Task;
const fetchBefore = globalThis.fetch;
afterEach(() => {
  globalThis.fetch = fetchBefore;
});
test("profile selection pins exact revision and rejects edited, archived or role-mismatched alternatives", () => {
  const choice = `${profileID}@3`;
  assert.deepEqual(profilePair(choice), {
    profile_id: profileID,
    profile_version: "3",
  });
  assert.equal(currentProfileChoice(choice, "developer", [profile]), true);
  for (const changed of [
    { ...profile, version: 4 },
    { ...profile, role: "reviewer" },
    { ...profile, state: "archived" } as ProfileSummary,
  ])
    assert.equal(currentProfileChoice(choice, "developer", [changed]), false);
  assert.equal(currentProfileChoice(choice, "developer", []), false);
  assert.equal(currentProfileChoice("", "developer", []), true);
  for (const value of [
    profileID,
    `${profileID}@0`,
    `${profileID}@03`,
    `${profileID}@1.5`,
    `${profileID}@9007199254740992`,
    `https://invalid/@3`,
  ])
    assert.throws(() => profilePair(value));
});
test("assignment draft remains pinned when server task refreshes and role-queue clear omits profile pair", () => {
  const draft = coordinationDraft(task);
  draft.reason = "Передача реализации";
  const fresh = { ...task, version: 8, assignment: undefined };
  const fields = coordinationFields("assign", draft, fresh, "human");
  assert.equal(fields.version, "7");
  assert.equal(fields.profile_version, "3");
  assert.equal(draft.profile, `${profileID}@3`);
  draft.profile = "";
  assert.deepEqual(coordinationFields("assign", draft, task, "human"), {
    version: "7",
    mode: "role_queue",
    reason: "Передача реализации",
  });
});
test("human administrative controls offer no agent completion or foreign lease release", () => {
  for (const state of ["created", "ready", "done", "cancelled", "in_progress"])
    assert.equal(
      transitionChoices({ ...task, state }).some(
        (choice) => choice.value === "done",
      ),
      false,
    );
  assert.deepEqual(
    transitionChoices({
      ...task,
      lease: { session_id: "agent", generation: 2, expires_at: "later" },
    }),
    [],
  );
  const leased = {
    ...task,
    lease: { session_id: "agent", generation: 2, expires_at: "later" },
  };
  assert.equal(
    coordinationFields("link", coordinationDraft(task), leased, "human")
      .generation,
    "0",
  );
  assert.equal(
    coordinationFields("link", coordinationDraft(task), leased, "agent")
      .generation,
    "2",
  );
  assert.equal(
    transitionChoices({ ...task, type: "epic", state: "open" })[0].value,
    "done",
  );
});
test("QA requires authoritative recursive success, not outcome or provenance alone", () => {
  const check = {
    task_key: "T-3",
    role: "reviewer",
    state: "done",
    outcome: "approve",
    current: true,
    latest: true,
    successful: true,
  } as CoordinationCheck;
  const view = (value: CoordinationCheck) =>
    ({ checks: [value] }) as TaskCoordination;
  assert.equal(latestPositiveReview(view(check))?.task_key, "T-3");
  for (const change of [
    { successful: false },
    { current: false },
    { latest: false },
    { state: "in_progress" },
    { outcome: "changes_requested" },
    { role: "tester" },
  ])
    assert.equal(
      latestPositiveReview(view({ ...check, ...change })),
      undefined,
    );
});
test("viewer and expired project views cannot acquire coordination controls even with stale capability flags", async () => {
  const page = {
    can_coordinate: true,
    can_continue: false,
    project: { can_write: true },
  } as PageData;
  assert.equal(canCoordinate(page), true);
  assert.equal(canCoordinate({ ...page, can_continue: true }), false);
  assert.equal(
    canCoordinate({ ...page, project: { ...page.project!, can_write: false } }),
    false,
  );
  assert.equal(canCoordinate({ ...page, can_coordinate: undefined }), false);
  assert.equal(
    canCoordinate({
      ...page,
      task,
      coordination: { task_version: 8 } as TaskCoordination,
    }),
    false,
  );
  globalThis.fetch = async () =>
    Response.json({
      ok: true,
      page: { ...page, session: {}, project: { can_write: false } },
    });
  assert.equal((await readPage("/app/tasks/T-2")).can_coordinate, false);
});
test("all coordination mutations keep scoped path, exact replay body and no redirect fallback", async () => {
  const scope = `p_${"a".repeat(32)}`;
  const body = new URLSearchParams({
    action_id: "same-action",
    version: "7",
    reason: "draft",
  });
  const pending = snapshotPayload(body);
  body.set("version", "8");
  for (const operation of ["assign", "link", "transition", "checks"]) {
    const path = `/web/p/${scope}/tasks/T-2/${operation}`;
    globalThis.fetch = async (input, init) => {
      assert.equal(input, path);
      assert.equal(init?.body, pending);
      assert.equal((init?.body as URLSearchParams).get("version"), "7");
      assert.equal(init?.credentials, "same-origin");
      assert.equal(init?.redirect, "error");
      return Response.json(
        {
          ok: false,
          retry_same_action: true,
          error: { code: "UNKNOWN_RESULT", message: "Повторите" },
        },
        { status: 503 },
      );
    };
    const result = await submitAction(path, pending);
    const state = applyActionResult(
      { actions: { "coord:assign": "same-action" } } as unknown as PageData,
      result,
    );
    assert.equal(state.keepPending, true);
    assert.equal(state.phase, "uncertain");
  }
  await assert.rejects(
    submitAction(`/web/p/${scope}/tasks/T-2/execute`, pending),
  );
});
test("check creation sends captured target version and role with optional exact profile pair", () => {
  const draft = coordinationDraft(task, "tester");
  draft.profile = "";
  assert.deepEqual(coordinationFields("checks", draft, task, "human"), {
    target_version: "7",
    role: "tester",
  });
  draft.profile = `${profileID}@3`;
  assert.deepEqual(coordinationFields("checks", draft, task, "human"), {
    target_version: "7",
    role: "tester",
    profile_id: profileID,
    profile_version: "3",
  });
});

test("assignment wording is the same for list and detail", () => {
  const base = { type: "task", role: "human" } as Parameters<
    typeof assignmentText
  >[0];
  assert.equal(assignmentText(base), "Вопрос человеку");
  assert.equal(
    assignmentText({ ...base, role: "developer" }),
    "Общая очередь роли",
  );
  assert.equal(
    assignmentText({ ...base, type: "epic", owner: "Анна" }),
    "Анна",
  );
});
