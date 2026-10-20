import assert from "node:assert/strict";
import test from "node:test";
import type { PageData } from "../../lib/types.ts";
import {
  actionBody,
  actionID,
  snapshotPayload,
  stoppedBy,
  uploadBody,
} from "./contracts.ts";

const page = {
  csrf: "csrf-fixture",
  actions: {
    answer: "generic-old",
    "answer:T-7": "answer-seven",
    "claim:T-8": "claim-eight",
    upload: "upload-one",
  },
} as unknown as PageData;

test("a task action uses the capability issued for that exact task", () => {
  assert.equal(actionID(page, "answer", "T-7"), "answer-seven");
  const body = actionBody(
    page,
    "answer",
    { generation: "3", answer: "  Точный ответ\n" },
    "T-7",
  );
  assert.deepEqual(
    [...body],
    [
      ["csrf", "csrf-fixture"],
      ["action_id", "answer-seven"],
      ["generation", "3"],
      ["answer", "  Точный ответ\n"],
    ],
  );
  assert.equal(actionID(page, "release", "T-7"), "");
});

test("an unknown URL-encoded outcome can replay the exact original generation, action and text", () => {
  const original = actionBody(
    page,
    "answer",
    { generation: "3", answer: "Черновик" },
    "T-7",
  );
  const frozen = snapshotPayload(original);
  original.set("generation", "4");
  original.set("answer", "Исправлено после отправки");
  original.set("action_id", "fresh-id");
  assert.equal(frozen.get("generation"), "3");
  assert.equal(frozen.get("answer"), "Черновик");
  assert.equal(frozen.get("action_id"), "answer-seven");
});

test("upload sends protection and all metadata before the sole final file", () => {
  const file = new File(["fixture"], "layout.svg", { type: "image/svg+xml" });
  const original = uploadBody(
    page,
    {
      title: "Макет",
      scope: "task",
      scope_key: "T-7",
      epic_key: "E-2",
      design_id: "design-fixture",
    },
    file,
  );
  assert.deepEqual(
    [...original.keys()],
    [
      "csrf",
      "action_id",
      "title",
      "scope",
      "scope_key",
      "epic_key",
      "design_id",
      "file",
    ],
  );
  const frozen = snapshotPayload(original);
  original.set("title", "Новая версия");
  original.set("file", new File(["different"], "other.pdf"));
  assert.equal(frozen.get("title"), "Макет");
  assert.equal((frozen.get("file") as File).name, "layout.svg");
  assert.equal((frozen.get("file") as File).size, 7);
});

test("lease or authentication loss stops new writes but known validation allows correction", () => {
  for (const code of [
    "LEASE_LOST",
    "NO_SESSION",
    "UNAUTHORIZED",
    "TOKEN_EXPIRED",
    "lease_lost",
  ])
    assert.equal(
      stoppedBy({ ok: false, error: { code, message: "fixture" } }),
      true,
    );
  assert.equal(
    stoppedBy({ ok: false, error: { code: "VALIDATION", message: "fixture" } }),
    false,
  );
});

test("server-returned draft protection fields cannot override a fresh capability", () => {
  const body = actionBody(
    page,
    "answer",
    { csrf: "stale-csrf", action_id: "stale-id", answer: "Исправлено" },
    "T-7",
  );
  assert.deepEqual(body.getAll("csrf"), ["csrf-fixture"]);
  assert.deepEqual(body.getAll("action_id"), ["answer-seven"]);
});
