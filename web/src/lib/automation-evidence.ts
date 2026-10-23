import { workflowLabels, type WorkflowAttestation } from "./automation.ts";
// Exit codes and partial counters are observations, not a success gate. An abort
// invalidates their presentation even when the process happened to exit zero.
export function attestationOutcome(
  proof: Pick<WorkflowAttestation, "status" | "observations">,
): { label: string; checkSummary?: string } {
  const observed = proof.observations;
  if (observed.check_aborted)
    return {
      label: "Проверка прервана, результат не засчитан",
      checkSummary: "Чтобы получить результат этой проверки, запустите её заново.",
    };
  return {
    label: workflowLabels[proof.status] ?? proof.status,
    checkSummary:
      observed.check_exit_code !== undefined
        ? `Код выхода: ${observed.check_exit_code}. Тесты: ${observed.checks_passed} пройдено, ${observed.checks_failed} ошибок, ${observed.checks_skipped} пропущено.`
        : undefined,
  };
}
