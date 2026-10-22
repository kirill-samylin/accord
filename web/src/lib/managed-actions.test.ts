import { afterEach, test } from "node:test";
import assert from "node:assert/strict";
import { APIError, pageQueryKey, readPage, submitAction } from "./api.ts";
import {
  appToWebRoute,
  projectAppPath,
  projectWebPath,
  webToAppRoute,
} from "./project-routes.ts";
import {
  commandReason,
  hostOnline,
  messageError,
  profileOption,
  selectedProfile,
  startErrors,
  startFields,
  workspaceBusy,
  managedPollInterval,
} from "./managed-agents.ts";
import {
  project,
  other,
  now,
  host,
  run,
  profile,
  page,
  draft,
} from "./managed-fixtures.test-support.ts";
import {
  actionBody,
  snapshotPayload,
} from "../features/mutations/contracts.ts";
import { applyActionResult } from "../features/mutations/actionState.ts";
const original = globalThis.fetch;
afterEach(() => {
  globalThis.fetch = original;
});
test("uncertain command retries original issued action, version, message and project twice", async () => {
  const before = page();
  const body = actionBody(before, "managed:command", {
    run_id: "mr_test",
    version: "2",
    kind: "clarify",
    text: "Original",
  });
  const frozen = snapshotPayload(body);
  body.set("text", "Changed later");
  before.actions["managed:command"] = "new-issued-command";
  const state = applyActionResult(before, {
    ok: false,
    retry_same_action: true,
    actions: { "managed:command": "replacement" },
  });
  assert.equal(state.keepPending, true);
  let calls = 0;
  globalThis.fetch = async (path, options) => {
    calls++;
    assert.equal(path, `/web/p/${project}/managed/command`);
    assert.ok(options?.body instanceof URLSearchParams);
    assert.equal(options.body.get("action_id"), "issued-command");
    assert.equal(options.body.get("version"), "2");
    assert.equal(options.body.get("text"), "Original");
    return Response.json({
      ok: true,
      redirect: `/web/p/${project}/managed/runs/mr_test`,
    });
  };
  await submitAction(`/web/p/${project}/managed/command`, frozen);
  await submitAction(`/web/p/${project}/managed/command`, frozen);
  assert.equal(calls, 2);
});
test("definitive conflict refreshes server version and action without discarding user text", () => {
  const before = page();
  const latest = {
    ...before,
    managed_run: { run: { ...run(), version: 3 }, commands: [] },
  };
  const state = applyActionResult(before, {
    ok: false,
    page: latest,
    error: { code: "conflict", message: "Version changed" },
    actions: { "managed:command": "fresh" },
  });
  assert.equal(state.keepPending, false);
  assert.equal(state.page.managed_run?.run.version, 3);
  assert.equal(state.page.actions["managed:command"], "fresh");
});
test("viewer managed reads remove host control and aborted or foreign reads cannot replace project", async () => {
  globalThis.fetch = async () =>
    Response.json({
      ok: true,
      page: { ...page(), project: { ...page().project, can_write: false } },
    });
  assert.equal(
    (await readPage(`/app/p/${project}/managed`)).managed?.hosts[0].can_control,
    false,
  );
  const controller = new AbortController();
  globalThis.fetch = async () => {
    controller.abort();
    return Response.json({ ok: true, page: page() });
  };
  await assert.rejects(
    readPage(`/app/p/${project}/managed`, controller.signal),
    (error) => error instanceof Error && error.name === "AbortError",
  );
  globalThis.fetch = async () =>
    Response.json({
      ok: true,
      page: { ...page(), project: { ...page().project, id: other } },
    });
  await assert.rejects(
    readPage(`/app/p/${project}/managed`),
    (error) =>
      error instanceof APIError &&
      error.result.error?.code === "PROJECT_SCOPE_MISMATCH",
  );
});
test("foreign success and enrollment, shell or malformed mutation routes fail closed", async () => {
  globalThis.fetch = async () =>
    Response.json({ ok: true, redirect: `/web/p/${other}/managed` });
  await assert.rejects(
    submitAction(`/web/p/${project}/managed/start`, new URLSearchParams()),
    (error) =>
      error instanceof APIError && error.result.retry_same_action === true,
  );
  for (const tail of [
    "enroll",
    "update",
    "execute",
    "runs/mr_test/command",
    "command/extra",
    "../start",
  ])
    await assert.rejects(
      submitAction(`/web/p/${project}/managed/${tail}`, new URLSearchParams()),
    );
});
