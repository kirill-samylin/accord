import assert from "node:assert/strict";
import test from "node:test";
import type { PageData } from "../../lib/types.ts";
import { readPage, submitAction } from "../../lib/api.ts";
import { applyActionResult, applyRefreshedPage } from "./actionState.ts";
import { createDraft } from "./createDraft.ts";

const page = {
  name: "login",
  csrf: "preauth-existing",
  draft: {},
  actions: {},
} as PageData;

test("wrong login credentials keep the valid form editable and release the rejected payload", () => {
  const result = {
    ok: false,
    error: { code: "UNAUTHORIZED", message: "Неверный токен" },
    page,
  };
  const login = applyActionResult(page, result, "login");
  assert.equal(login.phase, "idle");
  assert.equal(login.page.csrf, "preauth-existing");
  assert.equal(login.keepPending, false);
  assert.equal(
    applyActionResult(page, result).phase,
    "blocked",
    "domain mutations still stop on lost authentication",
  );
});

test("expired preauth blocks another credential submission until explicit fresh bootstrap", async () => {
  const rejected = applyActionResult(
    page,
    { ok: false, error: { code: "FORBIDDEN", message: "Обновите форму" } },
    "login",
  );
  assert.equal(rejected.phase, "blocked");
  assert.equal(rejected.page.csrf, "");
  assert.equal(rejected.keepPending, false);
  const originalFetch = globalThis.fetch;
  const requests: { path: string; init?: RequestInit }[] = [];
  globalThis.fetch = async (input, init) => {
    requests.push({ path: String(input), init });
    return Response.json({
      ok: true,
      page: { ...page, csrf: "fresh-preauth" },
    });
  };
  try {
    const fresh = await readPage("/app/login");
    const restored = applyRefreshedPage(fresh, "login");
    assert.equal(restored.phase, "idle");
    assert.equal(restored.page.csrf, "fresh-preauth");
    assert.equal(requests[0].path, "/web/login");
    assert.equal(
      requests[0].init?.body,
      undefined,
      "bootstrap never resubmits token or file",
    );
    assert.equal(requests[0].init?.credentials, "same-origin");
    assert.equal(
      applyRefreshedPage({ ...fresh, csrf: "" }, "login").phase,
      "blocked",
    );
  } finally {
    globalThis.fetch = originalFetch;
  }
});

test("unknown login outcome still requires immutable retry and ignores a replacement CSRF", () => {
  const state = applyActionResult(
    page,
    {
      ok: false,
      retry_same_action: true,
      page: { ...page, csrf: "unsuitable-new" },
    },
    "login",
  );
  assert.equal(state.phase, "uncertain");
  assert.equal(state.keepPending, true);
  assert.equal(state.page.csrf, "preauth-existing");
});

test("login policy allows correcting a confirmed error without any new bootstrap request", async () => {
  const originalFetch = globalThis.fetch;
  let requests = 0;
  globalThis.fetch = async () => {
    requests++;
    return Response.json(
      {
        ok: false,
        error: { code: "UNAUTHORIZED", message: "Неверный токен" },
        page,
      },
      { status: 401 },
    );
  };
  // submitAction's login401 exception must avoid broadcasting browser-session
  // revocation, which would unmount/reset the editable login input.
  const oldWindow = Object.getOwnPropertyDescriptor(globalThis, "window");
  let unauthorized = false;
  Object.defineProperty(globalThis, "window", {
    configurable: true,
    value: {
      dispatchEvent: () => {
        unauthorized = true;
      },
    },
  });
  try {
    const result = await submitAction(
      "/web/login",
      new URLSearchParams({
        csrf: page.csrf,
        token: "synthetic-invalid-token",
      }),
    );
    assert.equal(applyActionResult(page, result, "login").phase, "idle");
    assert.equal(requests, 1);
    assert.equal(unauthorized, false);
  } finally {
    globalThis.fetch = originalFetch;
    if (oldWindow) Object.defineProperty(globalThis, "window", oldWindow);
    else Reflect.deleteProperty(globalThis, "window");
  }
});

test("activity query fields cannot poison a create form operation or capability", () => {
  const draft = createDraft({
    type: "claimed",
    role: "not-a-role",
    state: "done",
    cursor: "cursor",
    csrf: "bad",
    action_id: "stale",
    title: "Valid title",
    priority: "bad",
  });
  assert.deepEqual(draft, {
    type: "task",
    role: "developer",
    priority: "2",
    title: "Valid title",
  });
  assert.equal(
    createDraft({ type: "epic", role: "human", priority: "1" }, ["human"]).type,
    "epic",
  );
  assert.equal(createDraft({}, ["human"]).role, "human");
});
