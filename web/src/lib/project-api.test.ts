import { afterEach, test } from "node:test";
import assert from "node:assert/strict";
import { APIError, pageQueryKey, readPage, submitAction } from "./api.ts";
import { catalogBody } from "./catalog-action.ts";
import type { CatalogPage } from "./projects.ts";
import { applyActionResult } from "../features/mutations/actionState.ts";
import type { PageData } from "./types.ts";
const a = "p_" + "a".repeat(32),
  b = "p_" + "b".repeat(32);
const originalFetch = globalThis.fetch;
afterEach(() => {
  globalThis.fetch = originalFetch;
});
test("a response containing another project's page is rejected before becoming query data", async () => {
  globalThis.fetch = async () =>
    Response.json({
      ok: true,
      page: {
        name: "task",
        project: { id: b },
        task: { key: "T-1", title: "other" },
      },
    });
  await assert.rejects(
    readPage(`/app/p/${a}/tasks/T-1`),
    (error) =>
      error instanceof APIError &&
      error.result.error?.code === "PROJECT_SCOPE_MISMATCH",
  );
  assert.notDeepEqual(
    pageQueryKey(`/app/p/${a}/tasks/T-1`),
    pageQueryKey(`/app/p/${b}/tasks/T-1`),
  );
});
test("mutation retries keep the captured scoped endpoint and reject cross-project page or redirect", async () => {
  const path = `/web/p/${a}/tasks/T-1/answer`;
  const body = new URLSearchParams({
    action_id: "original",
    answer: "draft A",
  });
  globalThis.fetch = async (actual, options) => {
    assert.equal(actual, path);
    assert.equal(options?.body, body);
    return Response.json({ ok: true, redirect: `/web/p/${b}/tasks/T-1` });
  };
  await assert.rejects(
    submitAction(path, body),
    (error) =>
      error instanceof APIError && error.result.retry_same_action === true,
  );
  globalThis.fetch = async () =>
    Response.json(
      { ok: false, page: { project: { id: b } }, error: { code: "CONFLICT" } },
      { status: 409 },
    );
  await assert.rejects(
    submitAction(path, body),
    (error) =>
      error instanceof APIError &&
      error.result.error?.code === "PROJECT_SCOPE_MISMATCH",
  );
});
test("catalog detail responses cannot retain another project's members or forms", async () => {
  globalThis.fetch = async () =>
    Response.json({
      ok: true,
      page: { name: "project-settings", project: { id: b }, members: [] },
    });
  await assert.rejects(
    readPage(`/app/projects/${a}`),
    (error) =>
      error instanceof APIError &&
      error.result.error?.code === "PROJECT_SCOPE_MISMATCH",
  );
});
test("definitive scoped validation and access errors never become an uncertain retry", async () => {
  const previous = {
    project: { id: a },
    actions: { answer: "old" },
  } as unknown as PageData;
  for (const status of [400, 403]) {
    globalThis.fetch = async () =>
      Response.json(
        {
          ok: false,
          error: {
            code: status === 400 ? "VALIDATION" : "FORBIDDEN",
            message: "Rejected",
          },
          page: {
            actions: { answer: "fresh" },
            field_errors: { answer: "Required" },
            task: { key: "T-foreign" },
          },
        },
        { status },
      );
    const result = await submitAction(
      `/web/p/${a}/tasks/T-1/answer`,
      new URLSearchParams(),
    );
    assert.equal(result.retry_same_action, undefined);
    assert.equal(result.page, undefined);
    assert.deepEqual(result.field_errors, { answer: "Required" });
    const state = applyActionResult(previous, result);
    assert.equal(state.keepPending, false);
    assert.notEqual(state.phase, "uncertain");
    assert.equal(state.phase, status === 400 ? "idle" : "blocked");
    assert.equal(state.page.project?.id, a);
    assert.equal(state.page.task, undefined);
  }
});
test("read-only project capabilities suppress inherited board edit flags", async () => {
  globalThis.fetch = async () =>
    Response.json({
      ok: true,
      page: {
        name: "task",
        session: {},
        project: { id: a, can_write: false },
        can_edit: true,
        can_update: true,
      },
    });
  const page = await readPage(`/app/p/${a}/tasks/T-1`);
  assert.equal(page.can_edit, false);
  assert.equal(page.can_update, false);
});
test("catalog action snapshots cannot inherit edited values or injected csrf/action capabilities", () => {
  const page = {
    name: "projects",
    registry: {
      identity: { id: "identity", owner: "fixture", machine: "fixture" },
      csrf: "catalog-csrf",
      actions: { [`switch:${a}`]: "action-a", [`switch:${b}`]: "action-b" },
    },
  } as CatalogPage;
  const fields = new URLSearchParams({
    project_id: a,
    csrf: "injected",
    action_id: "injected",
  });
  const captured = catalogBody(page, `switch:${a}`, fields);
  fields.set("project_id", b);
  assert.equal(captured.get("project_id"), a);
  assert.equal(captured.get("csrf"), "catalog-csrf");
  assert.equal(captured.get("action_id"), "action-a");
  assert.equal(
    catalogBody(page, `switch:${b}`, fields).get("action_id"),
    "action-b",
  );
});
test("aborted old project reads cannot publish a successfully decoded late body", async () => {
  const controller = new AbortController();
  globalThis.fetch = async () => {
    controller.abort();
    return Response.json({
      ok: true,
      page: { name: "task", project: { id: a } },
    });
  };
  await assert.rejects(
    readPage(`/app/p/${a}/tasks/T-1`, controller.signal),
    (error) => error instanceof Error && error.name === "AbortError",
  );
});
