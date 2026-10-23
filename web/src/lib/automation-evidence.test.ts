import { test } from "node:test";
import assert from "node:assert/strict";
import { attestationOutcome } from "./automation-evidence.ts";
import type { WorkflowAttestation } from "./automation.ts";
function proof(
  status: string,
  observed: Partial<WorkflowAttestation["observations"]> = {},
) {
  return {
    status,
    observations: {
      checks_passed: 0,
      checks_failed: 0,
      checks_skipped: 0,
      outcome_bytes_truncated: false,
      ...observed,
    },
  };
}
test("abort wins over a passed status, zero exit and partial successful counters", () => {
  const observed = proof("passed", {
    check_aborted: true,
    check_exit_code: 0,
    checks_passed: 77,
  });
  const outcome = attestationOutcome(observed);
  assert.equal(outcome.label, "Проверка прервана, результат не засчитан");
  assert.equal(
    outcome.checkSummary,
    "Чтобы получить результат этой проверки, запустите её заново.",
  );
  assert.doesNotMatch(outcome.checkSummary!, /77|пройдено|Код выхода/);
  assert.equal(
    attestationOutcome(
      proof("completed", { check_aborted: true, check_exit_code: 1 }),
    ).label,
    outcome.label,
  );
});
test("zero exit and counters do not promote an unconfirmed status to success", () => {
  const observed = proof("pending", { check_exit_code: 0, checks_passed: 4 });
  const outcome = attestationOutcome(observed);
  assert.equal(outcome.label, "Ожидает выполнения");
  assert.match(outcome.checkSummary!, /Код выхода: 0/);
  assert.match(outcome.checkSummary!, /4 пройдено/);
});
test("legacy observations without an abort field retain their recorded status", () => {
  assert.equal(attestationOutcome(proof("passed")).label, "Проверка пройдена");
  assert.equal(attestationOutcome(proof("running")).checkSummary, undefined);
});
