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
  profileSupportsProvider,
  providerName,
  selectedProfile,
  startErrors,
  startFields,
  startPrompt,
  workspaceBusy,
  managedPollInterval,
} from "./managed-agents.ts";
import {
  project,
  other,
  now,
  observedAt,
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
test("managed pages, details, redirects and command paths retain project scope", () => {
  const path = `/app/p/${project}/managed/runs/mr_test`;
  assert.equal(appToWebRoute(path), `/web/p/${project}/managed/runs/mr_test`);
  assert.equal(webToAppRoute(`/web/p/${project}/managed/runs/mr_test`), path);
  assert.equal(
    projectAppPath(project, "/app/managed"),
    `/app/p/${project}/managed`,
  );
  assert.equal(
    projectWebPath(project, "/web/managed/command"),
    `/web/p/${project}/managed/command`,
  );
  assert.notDeepEqual(
    pageQueryKey(path),
    pageQueryKey(path.replace(project, other)),
  );
  for (const path of [
    "/app/managed/start",
    "/app/managed/runs/x/extra",
    "/app/managed/runs/../other",
    "/app/managed/runs/%2fother",
  ])
    assert.throws(() => appToWebRoute(path));
});
test("online is fresh worker heartbeat with current ownership lease, never run state", () => {
  assert.equal(hostOnline(host(), observedAt), true);
  assert.equal(
    hostOnline(
      { ...host(), last_seen_at: new Date(now - 30001).toISOString() },
      observedAt,
    ),
    false,
  );
  assert.equal(
    hostOnline(
      { ...host(), lease_expires_at: new Date(now).toISOString() },
      observedAt,
    ),
    false,
  );
  assert.equal(hostOnline(host(), undefined), false);
  assert.equal(hostOnline({ ...host(), state: "disabled" }, observedAt), false);
  assert.equal(hostOnline(undefined, observedAt), false);
  assert.equal(run().state, "running");
});
test("a project editor cannot control another person's computer without independent grant", () => {
  assert.match(
    commandReason("stop", run(), { ...host(), can_control: false }, true),
    /не разрешил/,
  );
  assert.match(commandReason("stop", run(), host(), false), /просмотра/);
  assert.match(
    commandReason("stop", run(), { ...host(), state: "disabled" }, true),
    /отключено/,
  );
  assert.equal(
    commandReason("stop", run(), { ...host(), last_seen_at: "" }, true),
    "",
    "stop may be durably queued while offline",
  );
});
test("resume requires confirmed exit and exact native conversation and online allowed host", () => {
  for (const state of [
    "queued",
    "starting",
    "running",
    "stopping",
    "failed",
    "needs_recovery",
  ])
    assert.ok(
      commandReason("resume", { ...run(), state }, host(), true, observedAt),
      state,
    );
  for (const state of ["stopped", "completed"])
    assert.equal(
      commandReason("resume", { ...run(), state }, host(), true, observedAt),
      "",
    );
  assert.match(
    commandReason(
      "resume",
      { ...run(), state: "stopped", provider_session_id: undefined },
      host(),
      true,
    ),
    /диалога/,
  );
  assert.match(
    commandReason(
      "resume",
      { ...run(), state: "stopped" },
      { ...host(), last_seen_at: "" },
      true,
    ),
    /Подключите/,
  );
  assert.equal(
    commandReason(
      "clarify",
      { ...run(), state: "stopped" },
      host(),
      true,
      observedAt,
    ),
    "",
    "saving clarification does not spawn",
  );
});
test("start validates permission, host allowlists, busy workspace and immutable profile selection", () => {
  const data = { observed_at: observedAt, hosts: [host()], runs: [] };
  assert.deepEqual(startErrors(draft(), data, [profile()], true), {});
  assert.ok(
    startErrors(draft(), { ...data, runs: [run()] }, [profile()], true)
      .workspace,
  );
  assert.ok(
    startErrors(
      { ...draft(), workspace: "arbitrary/path", provider: "shell" },
      data,
      [profile()],
      true,
    ).workspace,
  );
  assert.ok(
    startErrors(
      { ...draft(), workspace: "arbitrary/path", provider: "shell" },
      data,
      [profile()],
      true,
    ).provider,
  );
  assert.ok(
    startErrors(draft(), data, [{ ...profile(), version: 2 }], true).profile,
  );
  assert.equal(
    selectedProfile(draft().profile, [{ ...profile(), state: "archived" }]),
    undefined,
  );
  assert.ok(startErrors(draft(), data, [profile()], false).host_id);
  assert.ok(
    startErrors(
      draft(),
      { ...data, hosts: [{ ...host(), can_control: false }] },
      [profile()],
      true,
    ).host_id,
  );
  assert.deepEqual(startFields(draft(), profile()), {
    host_id: "mh_test",
    workspace: "gui",
    provider: "codex",
    profile_id: profile().id,
    profile_version: "1",
    prompt: "Работа",
    task_key: "T-12",
  });
  assert.equal(
    selectedProfile(draft().profile, [{ ...profile(), version: 2 }]),
    undefined,
  );
  assert.equal(
    draft().profile,
    profileOption(profile()),
    "refresh cannot rebase selected revision",
  );
});
test("workspace remains busy through starting, stopping and uncertain ownership", () => {
  for (const state of [
    "queued",
    "starting",
    "running",
    "stopping",
    "needs_recovery",
  ])
    assert.equal(workspaceBusy([{ ...run(), state }], "mh_test", "gui"), true);
  assert.equal(workspaceBusy([run()], "mh_other", "gui"), false);
  assert.equal(workspaceBusy([run()], "mh_test", "other"), false);
  assert.equal(
    workspaceBusy([{ ...run(), state: "stopped" }], "mh_test", "gui"),
    false,
  );
});
test("manual editing requires a task; taskless diagnostics require explicit read-only choice", () => {
  const data = { observed_at: observedAt, hosts: [host()], runs: [] };
  const taskless = { ...draft(), task_key: "  " };
  assert.match(
    startErrors(taskless, data, [profile()], true).task_key,
    /задачу/,
  );
  const readOnly = { ...taskless, read_only: true };
  assert.deepEqual(startErrors(readOnly, data, [profile()], true), {});
  const fields = startFields(readOnly, profile());
  assert.equal("task_key" in fields, false);
  assert.match(fields.prompt, /Не изменяй файлы/);
  assert.match(fields.prompt, /не захватывай задачи и аренды Accord/);
  assert.ok(fields.prompt.endsWith(taskless.prompt));
  assert.equal(startPrompt(draft()), draft().prompt);
  assert.equal(
    "task_key" in startFields({ ...draft(), read_only: true }, profile()),
    false,
    "read-only choice cannot send a retained task reference",
  );
  assert.ok(startErrors(readOnly, data, [profile()], false).host_id);
  assert.ok(
    startErrors(readOnly, { ...data, runs: [run()] }, [profile()], true)
      .workspace,
    "diagnostics retain the existing workspace exclusion",
  );
  assert.ok(
    startErrors(
      { ...readOnly, prompt: "a".repeat(16384) },
      data,
      [profile()],
      true,
    ).prompt,
    "the diagnostic instruction counts towards the server prompt quota",
  );
});
test("text quotas count UTF-8 bytes and retain original valid clarification", () => {
  assert.equal(messageError("Я".repeat(2048), true), "");
  assert.ok(messageError("Я".repeat(2049), true));
  assert.ok(messageError("  ", true));
  assert.equal(messageError("", false), "");
  assert.ok(
    startErrors(
      { ...draft(), prompt: "Я".repeat(8193) },
      { observed_at: observedAt, hosts: [host()], runs: [] },
      [profile()],
      true,
    ).prompt,
  );
});
test("managed polling is project route only and pauses for pending writes or authority failures", () => {
  assert.equal(
    managedPollInterval(`/app/p/${project}/managed`, false, false),
    5000,
  );
  assert.equal(
    managedPollInterval(
      `/app/p/${project}/managed/runs/mr_test`,
      false,
      false,
      503,
      3,
    ),
    30000,
  );
  assert.equal(
    managedPollInterval(`/app/p/${project}/managed`, true, false),
    false,
  );
  assert.equal(
    managedPollInterval(`/app/p/${project}/managed`, false, true),
    false,
  );
  for (const status of [401, 403, 404, 409])
    assert.equal(
      managedPollInterval(`/app/p/${project}/managed`, false, false, status),
      false,
    );
  assert.equal(
    managedPollInterval(`/app/p/${project}/profiles`, false, false),
    false,
  );
});
test("profile provider compatibility rejects another vendor before queueing start", () => {
  assert.ok(
    startErrors(
      draft(),
      { observed_at: observedAt, hosts: [host()], runs: [] },
      [{ ...profile(), provider: "anthropic" }],
      true,
    ).profile,
  );
  assert.deepEqual(
    startErrors(
      draft(),
      { observed_at: observedAt, hosts: [host()], runs: [] },
      [{ ...profile(), provider: "unconfigured" }],
      true,
    ),
    {},
  );
});
test("gennadiy is a third native provider with its own profile provider", () => {
  assert.equal(providerName("gennadiy"), "Gennadiy (OpenCode)");
  assert.equal(providerName("codex"), "Codex");
  assert.equal(providerName("other"), "other");
  const base = profile();
  for (const [profileProvider, provider, expected] of [
    ["gennadiy", "gennadiy", true],
    ["unconfigured", "gennadiy", true],
    ["openai", "gennadiy", false],
    ["anthropic", "gennadiy", false],
    ["gennadiy", "codex", false],
    ["gennadiy", "claude", false],
  ] as const) {
    assert.equal(
      profileSupportsProvider({ ...base, provider: profileProvider }, provider),
      expected,
      `${profileProvider} profile on ${provider}`,
    );
  }
  const data = {
    observed_at: observedAt,
    hosts: [{ ...host(), providers: ["gennadiy"] }],
    runs: [],
  };
  const errors = startErrors(
    { ...draft(), provider: "gennadiy" },
    data,
    [{ ...base, provider: "gennadiy" }],
    true,
  );
  assert.equal(errors.provider, undefined);
  assert.equal(errors.profile, undefined);
  assert.ok(
    startErrors(
      { ...draft(), provider: "gennadiy" },
      data,
      [{ ...base, provider: "openai" }],
      true,
    ).profile,
  );
});
