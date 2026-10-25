import assert from "node:assert/strict";
import test from "node:test";
import { readPage, submitAction } from "./api.ts";
import {
  answerDraftKey,
  clearSessionDrafts,
  loadAnswerDraft,
  saveAnswerDraft,
} from "./session-drafts.ts";

const privateKeys = [
  "accord:create-draft:default",
  "accord:create-draft:p_" + "a".repeat(32),
  "accord:pending-task:default",
  "accord:create-project-name",
  "accord:answer-draft:default:T-1",
];

async function withDrafts(run: (values: Map<string, string>) => Promise<void>) {
  const previousWindow = Object.getOwnPropertyDescriptor(globalThis, "window");
  const previousFetch = globalThis.fetch;
  const values = new Map(privateKeys.map((key) => [key, "Person A's draft"]));
  values.set("another-feature", "keep");
  values.set("accord:create-draft-preference", "keep");
  Object.defineProperty(globalThis, "window", {
    configurable: true,
    value: {
      sessionStorage: {
        get length() {
          return values.size;
        },
        key: (index: number) => [...values.keys()][index] ?? null,
        removeItem: (key: string) => values.delete(key),
      },
      dispatchEvent: () => true,
    },
  });
  try {
    await run(values);
  } finally {
    globalThis.fetch = previousFetch;
    if (previousWindow)
      Object.defineProperty(globalThis, "window", previousWindow);
    else Reflect.deleteProperty(globalThis, "window");
  }
}

function assertCleared(values: Map<string, string>) {
  for (const key of privateKeys) assert.equal(values.has(key), false, key);
  assert.equal(values.get("another-feature"), "keep");
  assert.equal(values.get("accord:create-draft-preference"), "keep");
}

test("confirmed logout and subsequent login cannot restore the previous person's drafts", async () => {
  for (const path of [
    "/web/logout",
    "/web/login",
    "/web/p/p_" + "a".repeat(32) + "/logout",
  ]) {
    await withDrafts(async (values) => {
      globalThis.fetch = async () =>
        Response.json({ ok: true, redirect: "/web/login" });
      assert.equal((await submitAction(path, new URLSearchParams())).ok, true);
      assertCleared(values);
    });
  }
});

test("a lost authenticated session clears all owned drafts before another login", async () => {
  await withDrafts(async (values) => {
    globalThis.fetch = async () =>
      Response.json({ ok: false }, { status: 401 });
    await assert.rejects(readPage("/app/tasks"));
    assertCleared(values);
  });
});

test("a successful fallback login page clears drafts from the former session", async () => {
  await withDrafts(async (values) => {
    globalThis.fetch = async () =>
      Response.json({ ok: true, page: { name: "login" } });
    assert.equal((await readPage("/app/login")).name, "login");
    assertCleared(values);
  });
});

test("an uncertain logout keeps drafts until the server confirms the auth change", async () => {
  await withDrafts(async (values) => {
    globalThis.fetch = async () =>
      Response.json({ ok: false, retry_same_action: true });
    await submitAction("/web/logout", new URLSearchParams());
    for (const key of privateKeys) assert.equal(values.has(key), true);
  });
});

test("storage denial does not interrupt authentication cleanup", () => {
  const previousWindow = Object.getOwnPropertyDescriptor(globalThis, "window");
  Object.defineProperty(globalThis, "window", {
    configurable: true,
    value: {
      get sessionStorage() {
        throw new Error("blocked");
      },
    },
  });
  try {
    assert.doesNotThrow(clearSessionDrafts);
  } finally {
    if (previousWindow)
      Object.defineProperty(globalThis, "window", previousWindow);
    else Reflect.deleteProperty(globalThis, "window");
  }
});

test("answer drafts are scoped by project and question", () => {
  const previous = Object.getOwnPropertyDescriptor(globalThis, "window");
  const values = new Map<string, string>();
  Object.defineProperty(globalThis, "window", {
    configurable: true,
    value: {
      sessionStorage: {
        getItem: (key: string) => values.get(key) ?? null,
        setItem: (key: string, value: string) => values.set(key, value),
        removeItem: (key: string) => values.delete(key),
      },
    },
  });
  try {
    const a = "p_" + "a".repeat(32);
    saveAnswerDraft(a, "T-1", "да");
    assert.equal(loadAnswerDraft(a, "T-1"), "да");
    assert.equal(loadAnswerDraft("default", "T-1"), "");
    assert.equal(answerDraftKey(a, "T-2") === answerDraftKey(a, "T-1"), false);
    saveAnswerDraft(a, "T-1", "");
    assert.equal(values.size, 0);
  } finally {
    if (previous) Object.defineProperty(globalThis, "window", previous);
    else Reflect.deleteProperty(globalThis, "window");
  }
});
