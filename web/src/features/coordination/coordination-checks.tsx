import { ProjectLink as Link } from "@/components/project-link";
import { Status } from "@/components/common";
import {
  latestPositiveReview,
  type TaskCoordination,
} from "@/lib/coordination";
import { label } from "@/lib/format";
import type { Task } from "@/lib/types";
export function CoordinationChecks({
  task,
  coordination,
  manage,
  onCreate,
}: {
  task: Task;
  coordination: TaskCoordination;
  manage: boolean;
  onCreate: (role: string) => void;
}) {
  if (task.role !== "developer" && !task.review && !coordination.checks?.length)
    return null;
  const checkable =
    task.role === "developer" &&
    task.state === "done" &&
    /^(?:[a-f0-9]{40}|[a-f0-9]{64})$/i.test(task.evidence.commit ?? "");
  const reviewer = latestPositiveReview(coordination);
  return (
    <section className="coordination-section">
      <div className="section-heading">
        <h2>Ревью и QA</h2>
      </div>
      {manage && checkable ? (
        <div className="coordination-actions">
          <button
            className="button button-secondary button-sm"
            onClick={() => onCreate("reviewer")}
          >
            Создать ревью
          </button>
          <button
            className="button button-secondary button-sm"
            disabled={!reviewer}
            title={
              !reviewer
                ? "Сначала нужно актуальное положительное ревью"
                : `После ревью ${reviewer.task_key}`
            }
            onClick={() => onCreate("tester")}
          >
            Создать QA
          </button>
        </div>
      ) : null}
      {task.review ? (
        <p className="form-help">
          Проверяемая задача:{" "}
          <Link
            prefetch={false}
            className="inline-link"
            href={`/app/tasks/${task.review.target_key}`}
          >
            {task.review.target_key}
          </Link>{" "}
          · поколение {task.review.target_generation}
          <br />
          <span className="mono break">{task.review.target_commit}</span>
        </p>
      ) : null}
      {coordination.checks?.length ? (
        <ul className="coordination-list">
          {coordination.checks.map((check) => (
            <li key={check.task_key}>
              <div className="coordination-line">
                <Link
                  prefetch={false}
                  className="inline-link mono"
                  href={`/app/tasks/${check.task_key}`}
                >
                  {check.task_key}
                </Link>
                <strong>{label(check.role)}</strong>
                <Status value={check.state} />
                {check.outcome ? <span>{label(check.outcome)}</span> : null}
              </div>
              <p
                className={check.successful ? "coordination-positive" : "muted"}
              >
                {check.successful
                  ? "Актуальная положительная проверка"
                  : !check.current
                    ? "Проверка прежнего результата"
                    : !check.latest
                      ? "Заменена более новой проверкой"
                      : "Положительный результат ещё не принят"}
              </p>
              <span className="mono break">
                {check.target_commit} · поколение {check.target_generation}
              </span>
            </li>
          ))}
        </ul>
      ) : (
        <p className="muted">Проверки пока не назначены.</p>
      )}
      <p className="form-help">
        Проверки относятся к точному коммиту и поколению. Их выполняют отдельные
        сессии; новый цикл сохраняет историю.
      </p>
    </section>
  );
}
