import { ProjectLink as Link } from "@/components/project-link";
import { Disclosure } from "@/components/disclosure";
import type { WorkflowPage } from "@/lib/automation";
import { stageLabels, workflowLabels } from "@/lib/automation";
import { attestationOutcome } from "@/lib/automation-evidence";
import { workflowReason } from "@/lib/automation-reasons";
import { date } from "@/lib/format";
import type { ManagedRunPage, ManagedPage } from "@/lib/managed-agents";
import { runLabels, providerName, deliveryLabels } from "@/lib/managed-agents";
export function WorkflowProgress({
  detail,
  managed,
  delivery,
}: {
  detail: WorkflowPage;
  managed?: ManagedPage;
  delivery?: ManagedRunPage;
}) {
  const flow = detail.workflow;
  return (
    <>
      <section className="automation-card screen-stack">
        <h2>{workflowLabels[flow.state] ?? flow.state}</h2>
        {flow.reason_code ? (
          <p className="notice">{workflowReason(flow.reason_code)}</p>
        ) : null}
        <p className="automation-text">{flow.intake_text}</p>
        <div className="automation-facts">
          <span>
            Запусков: {flow.counters.invocations}
            {flow.grant ? ` / ${flow.grant.limits.native_invocations}` : ""}
          </span>
          <span>Подтверждённых стартов: {flow.counters.started}</span>
          <span>Задач: {flow.counters.tasks}</span>
          <span>Исправлений: {flow.counters.fix_total}</span>
          <span>Версия плана: {flow.plan_revision}</span>
        </div>
        {flow.epic_key ? (
          <Link href={`/app/tasks/${flow.epic_key}`} prefetch={false}>
            Все задачи · {flow.epic_key}
          </Link>
        ) : null}
        {flow.grant ? (
          <p className="form-help">
            Разрешение до {date(flow.grant.expires_at)}. Предел цикла:{" "}
            {date(flow.deadline_at)}. Срок разрешения и состояние процесса
            проверяются отдельно.
          </p>
        ) : (
          <p className="form-help">
            Автоматический запуск ещё не разрешён. Выберите действующую команду
            в управлении и явно продолжите.
          </p>
        )}
      </section>
      <section className="screen-stack">
        <h2>Ход выполнения</h2>
        <ol className="automation-steps">
          {detail.steps.map((step) => {
            const run = managed?.runs.find(
              (entry) => entry.id === step.managed_run_id,
            );
            return (
              <li className="automation-card screen-stack" key={step.id}>
                <div className="automation-row">
                  <strong>
                    {step.sequence}. {stageLabels[step.kind] ?? step.kind}
                  </strong>
                  <span>
                    {workflowLabels[step.status] ??
                      runLabels[step.status] ??
                      step.status}
                  </span>
                </div>
                <p className="muted">
                  {step.profile_snapshot.name} · {providerName(step.provider)} ·
                  попытка {step.attempt_no}
                </p>
                <div className="form-actions">
                  {step.task_key ? (
                    <Link href={`/app/tasks/${step.task_key}`} prefetch={false}>
                      Задача {step.task_key} · поколение {step.task_generation}
                    </Link>
                  ) : null}
                  {step.managed_run_id ? (
                    <Link
                      href={`/app/managed/runs/${step.managed_run_id}`}
                      prefetch={false}
                    >
                      Процесс:{" "}
                      {run
                        ? (runLabels[run.state] ?? run.state)
                        : "откройте актуальное состояние"}
                    </Link>
                  ) : null}
                </div>
                {delivery?.run.id === step.managed_run_id &&
                delivery.commands.length ? (
                  <p className="form-help">
                    Доставка последней команды:{" "}
                    {deliveryLabels[
                      delivery.commands[delivery.commands.length - 1].status
                    ] ?? delivery.commands[delivery.commands.length - 1].status}
                    . Подтверждение процесса показано отдельно.
                  </p>
                ) : null}
                {step.target_sha ? (
                  <p className="mono break">
                    Проверяемый SHA: {step.target_sha}
                  </p>
                ) : null}
                {step.evidence_refs?.length ? (
                  <ul>
                    {step.evidence_refs.map((ref) => (
                      <li className="break" key={ref}>
                        {ref}
                      </li>
                    ))}
                  </ul>
                ) : null}
              </li>
            );
          })}
        </ol>
        {!detail.steps.length ? (
          <p className="form-help">
            На этой странице шаги не показаны. Другие записи могут быть на
            следующих или предыдущих страницах истории.
          </p>
        ) : null}
      </section>
    </>
  );
}
export function WorkflowReports({ detail }: { detail: WorkflowPage }) {
  return (
    <section className="screen-stack">
      <h2>Результаты и подтверждения</h2>
      {detail.plans.map((plan) => (
        <Disclosure
          key={plan.revision}
          className="automation-card"
          summary={`План ${plan.revision} · ${plan.status}`}
        >
          <ol>
            {plan.tasks.map((task) => (
              <li key={task.client_key}>
                <strong>{task.title}</strong>
                <p className="automation-text">{task.description}</p>
                <p className="muted">
                  Проверки: {task.required_check_ids.join(", ")}
                </p>
                {plan.task_keys[task.client_key] ? (
                  <Link
                    href={`/app/tasks/${plan.task_keys[task.client_key]}`}
                    prefetch={false}
                  >
                    Открыть задачу {plan.task_keys[task.client_key]}
                  </Link>
                ) : null}
                <ul>
                  {task.acceptance.map((item, index) => (
                    <li key={index}>{item}</li>
                  ))}
                </ul>
              </li>
            ))}
          </ol>
        </Disclosure>
      ))}
      {detail.attestations.map((proof) => {
        const outcome = attestationOutcome(proof);
        return (
          <article className="automation-card screen-stack" key={proof.id}>
            <div className="automation-row">
              <strong>
                {proof.kind === "run_exit"
                  ? "Процесс завершён"
                  : proof.preset_id || proof.kind}
              </strong>
              <span>{outcome.label}</span>
            </div>
            <p className="mono break">
              SHA:{" "}
              {proof.observations.actual_head ||
                proof.observations.target_sha ||
                "не подтверждён"}
            </p>
            {outcome.checkSummary ? <p>{outcome.checkSummary}</p> : null}
            {proof.observations.source_clean !== undefined ? (
              <p>
                Исходники после этапа:{" "}
                {proof.observations.source_clean
                  ? "без незакоммиченных изменений"
                  : "изменены — требуется проверка"}
                .
              </p>
            ) : null}
            {proof.observations.report_sha256 ? (
              <p className="mono break">
                Хеш отчёта: {proof.observations.report_sha256}
              </p>
            ) : null}
            {proof.observations.artifact_refs?.map((ref) => (
              <p className="break" key={ref}>
                {ref}
              </p>
            ))}
            {proof.observations.outcome_bytes_truncated ? (
              <p className="notice">
                Часть результата не передана. Полные данные хранятся на
                компьютере исполнителя.
              </p>
            ) : null}
          </article>
        );
      })}
      {!detail.attestations.length ? (
        <p className="form-help">
          На этой странице нет подтверждений выполненных проверок. Просмотрите
          другие страницы истории, если они доступны.
        </p>
      ) : null}
      {detail.workflow.messages?.length ? (
        <Disclosure className="automation-card" summary="Переданные уточнения">
          <ol>
            {detail.workflow.messages.map((message, i) => (
              <li key={i} className="automation-text">
                {message}
              </li>
            ))}
          </ol>
        </Disclosure>
      ) : null}
    </section>
  );
}
