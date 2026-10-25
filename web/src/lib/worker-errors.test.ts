import assert from "node:assert/strict";
import test from "node:test";
import { workerErrorHint } from "./worker-errors.ts";

test("changed provider or workspace identity explains the owner's consent step", () => {
  for (const message of [
    "local workspace or provider identity changed",
    "local path identity changed; explicit worker consent is required",
    "local executable/workspace identity changed; explicit local consent is required",
  ]) {
    const hint = workerErrorHint(message);
    assert.ok(hint, message);
    assert.match(hint, /worker consent/);
  }
});

test("unknown or empty worker errors get no invented hint", () => {
  assert.equal(workerErrorHint("disk is full"), undefined);
  assert.equal(workerErrorHint(""), undefined);
  assert.equal(workerErrorHint(undefined), undefined);
});
