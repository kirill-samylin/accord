"use client";
import { useState } from "react";
import { ProjectLink as Link } from "@/components/project-link";
import { Fact } from "@/components/common";
import {
  canCoordinate,
  transitionChoices,
  type CoordinationOperation,
  type CoordinationReason,
} from "@/lib/coordination";
import { date, label } from "@/lib/format";
import type { MutationProps } from "../mutations/contracts";
import { CoordinationDialog } from "./coordination-dialog";
import { CoordinationChecks } from "./coordination-checks";
export function CoordinationPanel({
  page,
  onSuccess,
  readOnly = false,
}: MutationProps & { readOnly?: boolean }) {
  const [dialog, setDialog] = useState<{
    operation: CoordinationOperation;
    role?: string;
  } | null>(null);
  const task = page.task;
  const coordination = page.coordination;
  if (!task) return null;
  if (!coordination)
    return (
      <p className="muted">
        Сведения о координации недоступны. Обновите страницу.
      </p>
    );
  const manage = !readOnly && canCoordinate(page);
  const liveForeign =
    task.lease && task.lease.session_id !== page.session.session_id;
  const assignable =
    task.type === "task" &&
    task.role !== "human" &&
    !task.lease &&
    ["ready", "created"].includes(task.state);
  return (
    <div className="coordination-panel screen-stack">
      {coordination.task_version !== task.version ? (
        <p className="notice" role="status">
          Задача изменилась во время чтения. Обновите страницу перед действием;
          показанные сведения относятся к разным версиям.
        </p>
      ) : null}
      <section className="coordination-section">
        <div className="section-heading">
          <h2>
            {task.type === "epic"
              ? "Готовность эпика"
              : "Назначение и готовность"}
          </h2>
          {manage && assignable ? (
            <button
              className="button button-secondary button-sm"
              onClick={() => setDialog({ operation: "assign" })}
            >
              Изменить назначение
            </button>
          ) : null}
        </div>
        {task.type === "task" ? (
          <dl>
            <Fact
              name="Профиль"
              value={
                task.assignment ? (
                  <Link
                    prefetch={false}
                    className="inline-link"
                    href={`/app/profiles/${task.assignment.profile_id}?version=${task.assignment.profile_version}`}
                  >
                    {task.assignment.profile_name} · версия{" "}
                    {task.assignment.profile_version}
                  </Link>
                ) : task.role === "human" ? (
                  "Вопрос человеку"
                ) : (
                  "Общая очередь роли"
                )
              }
            />
            <Fact name="Роль" value={label(task.role)} />
            <Fact
              name="Текущая аренда"
              value={
                task.lease ? (
                  <Link
                    prefetch={false}
                    className="mono break"
                    href={`/app/agents/${task.lease.session_id}`}
                  >
                    {task.lease.session_id}
                  </Link>
                ) : (
                  "Не взята"
                )
              }
            />
          </dl>
        ) : (
          <p>
            {coordination.epic_closable
              ? "Все условия закрытия выполнены"
              : "Условия закрытия ещё не выполнены"}
          </p>
        )}
        <Reasons reasons={coordination.routing_reasons ?? []} />
        <Reasons reasons={coordination.blockers ?? []} />
        <Reasons reasons={coordination.closure_reasons ?? []} />
        {task.type === "task" &&
        task.state === "ready" &&
        !coordination.routing_reasons?.length &&
        !coordination.blockers?.length ? (
          <p className="form-help">
            Задача готова к взятию подходящей сессией. Назначение не запускает
            исполнителя.
          </p>
        ) : null}
        {manage && transitionChoices(task).length ? (
          <button
            className="button button-secondary button-sm"
            onClick={() => setDialog({ operation: "transition" })}
          >
            Изменить состояние
          </button>
        ) : null}
      </section>
      {task.type === "task" ? (
        <section className="coordination-section">
          <div className="section-heading">
            <h2>Входящие зависимости</h2>
            {manage && !liveForeign ? (
              <button
                className="button button-secondary button-sm"
                onClick={() => setDialog({ operation: "link" })}
              >
                Управлять связью
              </button>
            ) : null}
          </div>
          {coordination.dependencies?.length ? (
            <ul className="coordination-list">
              {coordination.dependencies.map(({ link, satisfied }) => (
                <li key={`${link.from}:${link.to}:${link.type}`}>
                  <div className="coordination-line">
                    <Link
                      prefetch={false}
                      className="inline-link mono"
                      href={`/app/tasks/${link.from}`}
                    >
                      {link.from}
                    </Link>
                    <span>{linkLabels[link.type] ?? link.type}</span>
                    <span
                      className={satisfied ? "coordination-positive" : "muted"}
                    >
                      {["blocks", "contract_of"].includes(link.type)
                        ? satisfied
                          ? "Условие выполнено"
                          : "Ожидает результата"
                        : "Информационная связь"}
                    </span>
                  </div>
                  {link.condition ? (
                    <span className="form-help">
                      Условие:{" "}
                      {conditionLabels[link.condition] ?? link.condition}
                    </span>
                  ) : null}
                </li>
              ))}
            </ul>
          ) : (
            <p className="muted">Входящих зависимостей нет.</p>
          )}
          {liveForeign ? (
            <p className="form-help">
              Задача уже взята другой сессией. Изменять её зависимости вправе
              только держатель аренды.
            </p>
          ) : null}
        </section>
      ) : null}
      <CoordinationChecks
        task={task}
        coordination={coordination}
        manage={manage}
        onCreate={(role) => setDialog({ operation: "checks", role })}
      />
      {coordination.truncated ? (
        <p className="notice" role="status">
          Показаны первые 200 элементов каждой группы. Другие условия могут
          оставаться в проекте.
        </p>
      ) : null}
      <p className="form-help">
        Состояние координации на {date(coordination.observed_at)}.
      </p>
      {dialog ? (
        <CoordinationDialog
          key={`${task.key}:${dialog.operation}:${dialog.role ?? ""}`}
          page={page}
          onSuccess={onSuccess}
          operation={dialog.operation}
          checkRole={dialog.role}
          onClose={() => setDialog(null)}
        />
      ) : null}
    </div>
  );
}
function Reasons({ reasons }: { reasons: CoordinationReason[] }) {
  return reasons.length ? (
    <ul className="coordination-reasons">
      {reasons.map((reason, index) => (
        <li key={`${reason.code}:${reason.task_key ?? ""}:${index}`}>
          <span>{reason.message}</span>
          {reason.task_key ? (
            <>
              {" "}
              <Link
                prefetch={false}
                className="inline-link mono"
                href={`/app/tasks/${reason.task_key}`}
              >
                {reason.task_key}
              </Link>
            </>
          ) : null}
        </li>
      ))}
    </ul>
  ) : null;
}
const linkLabels: Record<string, string> = {
  blocks: "Блокирует",
  contract_of: "Предоставляет контракт",
  spawned_by: "Порождена",
  related: "Связана",
};
const conditionLabels: Record<string, string> = {
  completed: "Успешное завершение",
  approved: "Положительное ревью",
  passed: "Успешное QA",
};
