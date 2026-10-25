import assert from "node:assert/strict";
import { test } from "node:test";
import { createTaskBlock } from "./projects.ts";

test("new task is explained when disabled", () => {
  const active = { state: "active", can_write: true } as const;
  assert.equal(createTaskBlock(active, false, false), "");
  assert.match(
    createTaskBlock({ state: "archived", can_write: false }, false, false),
    /архиве/,
  );
  assert.match(
    createTaskBlock({ ...active, can_write: false }, false, false),
    /чтения/,
  );
  assert.match(createTaskBlock(active, true, false), /Сессия/);
  assert.match(createTaskBlock(undefined, false, true), /отправки/);
});
