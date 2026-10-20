import { afterEach, test } from "node:test";
import assert from "node:assert/strict";
import {
  profileDraft,
  profileDraftErrors,
  profileFields,
  profileVersionChanged,
  type AgentProfile,
} from "./agent-profiles.ts";
import { APIError, pageQueryKey, readPage, submitAction } from "./api.ts";
import {
  appToWebRoute,
  projectAppPath,
  projectWebPath,
  webToAppRoute,
} from "./project-routes.ts";
import {
  actionBody,
  snapshotPayload,
} from "../features/mutations/contracts.ts";
import { applyActionResult } from "../features/mutations/actionState.ts";
import type { PageData } from "./types.ts";
const project = "p_" + "a".repeat(32);
const other = "p_" + "b".repeat(32);
const id = "ap_" + "c".repeat(32);
function profile(): AgentProfile {
  return {
    id,
    version: 1,
    name: "Разработчик",
    description: "Описание",
    role: "developer",
    instructions: "Исходные инструкции",
    provider: "unconfigured",
    model: "",
    state: "active",
    created_at: "2026-10-20T12:00:00Z",
    updated_at: "2026-10-20T12:00:00Z",
    created_by: "session",
    updated_by: "session",
  };
}
function page(): PageData {
  return {
    profile: profile(),
    actions: { "profile:update": "update-original" },
    csrf: "csrf",
    project: { id: project },
    can_manage_profiles: true,
  } as unknown as PageData;
}
const original = globalThis.fetch;
afterEach(() => {
  globalThis.fetch = original;
});
test("profile list, immutable revision and mutation routes retain exactly their project", () => {
  const path = `/app/p/${project}/profiles/${id}?version=1`;
  assert.equal(
    appToWebRoute(path),
    `/web/p/${project}/profiles/${id}?version=1`,
  );
  assert.equal(
    webToAppRoute(`/web/p/${project}/profiles/${id}`),
    `/app/p/${project}/profiles/${id}`,
  );
  assert.equal(
    projectAppPath(project, "/app/profiles?state=archived"),
    `/app/p/${project}/profiles?state=archived`,
  );
  assert.equal(
    projectWebPath(project, `/web/profiles/${id}/archive`),
    `/web/p/${project}/profiles/${id}/archive`,
  );
  assert.notDeepEqual(
    pageQueryKey(path),
    pageQueryKey(path.replace(project, other)),
  );
  assert.notDeepEqual(
    pageQueryKey(path),
    pageQueryKey(path.replace("version=1", "version=2")),
  );
  for (const invalid of [
    `/app/profiles/${id}/archive`,
    "/app/profiles/name",
    `/app/p/${project}/profiles/ap_BAD`,
    `/app/profiles/${id}/../other`,
  ])
    assert.throws(() => appToWebRoute(invalid));
});
test("refreshing a newer revision never mutates or silently rebases the user's draft", () => {
  const old = profile();
  const draft = { ...profileDraft(old), instructions: "Черновик пользователя" };
  const latest = {
    ...old,
    version: 2,
    instructions: "Правка другого участника",
  };
  assert.equal(profileVersionChanged(draft, latest), true);
  assert.equal(draft.version, "1");
  assert.equal(draft.instructions, "Черновик пользователя");
  assert.equal(profileFields(draft, true).version, "1");
  const adopted = { ...draft, version: String(latest.version) };
  assert.equal(profileVersionChanged(adopted, latest), false);
  assert.equal(adopted.instructions, draft.instructions);
  assert.equal("version" in profileFields(profileDraft(), false), false);
});
test("UTF-8 limits and Unicode character limits are validated independently", () => {
  const draft = {
    ...profileDraft(),
    name: "😀".repeat(120),
    description: "Я".repeat(2048),
    instructions: "Я".repeat(8192),
    model: "м".repeat(200),
  };
  assert.deepEqual(profileDraftErrors(draft), {});
  const errors = profileDraftErrors({
    ...draft,
    name: draft.name + "a",
    description: draft.description + "Я",
    instructions: draft.instructions + "Я",
    model: "unsafe\nname",
    role: "human",
    provider: "toString",
  });
  for (const field of [
    "name",
    "description",
    "instructions",
    "model",
    "role",
    "provider",
  ])
    assert.ok(errors[field], field);
});
test("profile retry freezes original revision, action ID, body and scoped endpoint", async () => {
  const initial = page();
  const draft = profileDraft(initial.profile);
  const body = actionBody(
    initial,
    "profile:update",
    profileFields(draft, true),
  );
  const captured = snapshotPayload(body);
  body.set("version", "9");
  body.set("instructions", "Поздняя правка");
  initial.actions["profile:update"] = "replacement";
  const path = `/web/p/${project}/profiles/${id}/update`;
  globalThis.fetch = async (actual, options) => {
    assert.equal(actual, path);
    const sent = options?.body as URLSearchParams;
    assert.equal(sent.get("version"), "1");
    assert.equal(sent.get("instructions"), "Исходные инструкции");
    assert.equal(sent.get("action_id"), "update-original");
    return Response.json({
      ok: true,
      redirect: `/web/p/${project}/profiles/${id}`,
    });
  };
  await submitAction(path, captured);
  await submitAction(path, captured);
});
test("foreign project success and malformed profile mutation addresses are rejected", async () => {
  globalThis.fetch = async () =>
    Response.json({ ok: true, redirect: `/web/p/${other}/profiles/${id}` });
  await assert.rejects(
    submitAction(
      `/web/p/${project}/profiles/${id}/archive`,
      new URLSearchParams(),
    ),
    (error) =>
      error instanceof APIError && error.result.retry_same_action === true,
  );
  for (const path of [
    "/web/profiles/name/update",
    `/web/profiles/${id}/launch`,
    `/web/profiles/${id}`,
  ])
    await assert.rejects(submitAction(path, new URLSearchParams()));
});
test("definitive version conflict replaces capabilities while retaining original draft version", async () => {
  const before = page();
  const draft = {
    ...profileDraft(before.profile),
    instructions: "Мой черновик",
  };
  globalThis.fetch = async () =>
    Response.json(
      {
        ok: false,
        error: { code: "conflict", message: "Version differs" },
        page: {
          ...before,
          profile: { ...profile(), version: 2 },
          actions: { "profile:update": "new-action" },
        },
      },
      { status: 409 },
    );
  const result = await submitAction(
    `/web/p/${project}/profiles/${id}/update`,
    new URLSearchParams(),
  );
  const state = applyActionResult(before, result);
  assert.equal(state.keepPending, false);
  assert.equal(state.phase, "idle");
  assert.equal(state.page.actions["profile:update"], "new-action");
  assert.equal(profileVersionChanged(draft, state.page.profile), true);
  assert.equal(draft.instructions, "Мой черновик");
  assert.equal(draft.version, "1");
});
test("viewer capability and aborted profile reads cannot expose writable stale state", async () => {
  globalThis.fetch = async () =>
    Response.json({
      ok: true,
      page: {
        ...page(),
        session: {},
        project: { id: project, can_write: false },
      },
    });
  assert.equal(
    (await readPage(`/app/p/${project}/profiles/${id}`)).can_manage_profiles,
    false,
  );
  const controller = new AbortController();
  globalThis.fetch = async () => {
    controller.abort();
    return Response.json({ ok: true, page: page() });
  };
  await assert.rejects(
    readPage(`/app/p/${project}/profiles/${id}`, controller.signal),
    (error) => error instanceof Error && error.name === "AbortError",
  );
});
