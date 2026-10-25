import assert from "node:assert/strict";
import { test } from "node:test";
import { myQuestions, taskTab } from "./task-groups.ts";
import type { PageData, Task } from "./types.ts";

const question = (key: string, session?: string) =>
  ({
    key,
    role: "human",
    state: "in_progress",
    lease: session
      ? { session_id: session, generation: 1, expires_at: "" }
      : undefined,
  }) as Task;

test("my questions include the ones this session claimed", () => {
  const page = {
    my_questions: [question("T-1")],
    questions: [question("T-2", "me"), question("T-3", "other")],
    tasks: [question("T-1"), question("T-2", "me")],
    session: { session_id: "me" },
  } as unknown as PageData;
  assert.deepEqual(
    myQuestions(page).map((task) => task.key),
    ["T-1", "T-2"],
  );
});

test("unknown tabs fall back to all", () => {
  assert.equal(taskTab("mine", true), "mine");
  assert.equal(taskTab("mine", false), "all");
  assert.equal(taskTab("epics", false), "epics");
  assert.equal(taskTab(null, true), "all");
});
