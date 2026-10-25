import assert from "node:assert/strict";
import { test } from "node:test";
import {
  actorLabel,
  isOpaqueSession,
  parseProfileRevision,
  profileReasonText,
} from "./event-display.ts";
import { eventFilterTypes, eventLabels, eventTitle } from "./format.ts";

const id = "ap_" + "a".repeat(32);

test("profile revision refs resolve to names, never raw ids", () => {
  assert.deepEqual(parseProfileRevision(`${id}@3`), { id, version: 3 });
  assert.equal(parseProfileRevision("обычная причина"), null);
  assert.equal(
    profileReasonText(`${id}@2`, [{ id, name: "Ревьюер" }]),
    "«Ревьюер» · версия 2",
  );
  assert.equal(profileReasonText(`${id}@1`), "Профиль агента · версия 1");
  assert.equal(profileReasonText(undefined), null);
});

test("session hashes become a neutral label", () => {
  const hash = "d1e0".repeat(16);
  assert.ok(isOpaqueSession(hash));
  assert.equal(actorLabel(undefined, undefined, hash), "Сессия агента");
  assert.equal(actorLabel("Анна", "mac", hash), "Анна · mac");
  assert.equal(actorLabel(undefined, "mac", hash), "mac");
  assert.equal(actorLabel(), "");
});

test("every event code has a Russian title", () => {
  for (const type of [
    "workflow_submitted",
    "profile_created",
    "profile_updated",
    "profile_state_changed",
    "automation_history_archived",
    "member_grant",
    "member_revoke",
    "managed_report_started",
    ...eventFilterTypes,
  ])
    assert.match(eventLabels[type] ?? "", /[А-Яа-яЁё]/, type);
  assert.equal(eventTitle("something_new"), "Событие");
});
