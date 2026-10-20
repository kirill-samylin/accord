"use client";

import { useEffect, useState, type FormEvent } from "react";
import type { Task } from "@/lib/types";
import { actionBody, actionID, type MutationProps } from "./contracts";
import { ActionNotice, FieldError } from "./ActionNotice";
import { useAction } from "./useAction";

export function QuestionActions({
  page,
  task: suppliedTask,
  onSuccess,
  onLockChange,
}: MutationProps & { task?: Task; onLockChange?: (locked: boolean) => void }) {
  const action = useAction(page, onSuccess, "action", onLockChange);
  const task = action.page.task ?? suppliedTask;
  const [answer, setAnswer] = useState(page.draft?.answer ?? "");
  const [reason, setReason] = useState(page.draft?.reason ?? "");
  const [now, setNow] = useState(0);
  // This clock only changes presentation. Reading a question never claims it or
  // extends its lease; heartbeat always requires the explicit button below.
  useEffect(() => {
    setNow(Date.now());
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, []);
  if (!task || task.role !== "human" || task.type !== "task") return null;
  const lease = task.lease;
  const deadline = Math.min(
    Date.parse(lease?.expires_at ?? ""),
    Date.parse(action.page.session.expires_at),
  );
  const expired = now > 0 && Number.isFinite(deadline) && deadline <= now;
  const canEdit =
    action.page.project?.can_write !== false &&
    action.phase !== "blocked" &&
    action.page.can_edit &&
    !action.page.can_continue &&
    lease?.session_id === action.page.session.session_id &&
    !expired;
  const canClaim =
    action.page.project?.can_write !== false &&
    task.state === "ready" &&
    !task.blocked &&
    !action.page.can_continue &&
    !lease;
  const secondsLeft = Math.max(0, Math.ceil((deadline - now) / 1000));
  function send(
    operation: "claim" | "answer" | "release" | "heartbeat",
    fields: Record<string, string> = {},
  ) {
    if (!task) return;
    const values =
      operation === "claim"
        ? fields
        : { generation: String(task.generation), ...fields };
    if (operation === "heartbeat") values.task_key = task.key;
    void action.send(
      operation === "heartbeat"
        ? "/web/heartbeat"
        : `/web/tasks/${encodeURIComponent(task.key)}/${operation}`,
      actionBody(action.page, operation, values, task.key),
    );
  }
  function answerQuestion(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    send("answer", { answer });
  }
  return (
    <section
      className="question-actions"
      aria-labelledby="question-actions-title"
    >
      <div className="section-heading">
        <h2 id="question-actions-title">
          {canEdit ? "Ваш ответ" : "Вопрос человеку"}
        </h2>
        {canEdit && now > 0 && Number.isFinite(secondsLeft) ? (
          <span className="form-help">
            Осталось {Math.floor(secondsLeft / 60)}:
            {String(secondsLeft % 60).padStart(2, "0")}
          </span>
        ) : null}
      </div>
      {canClaim ? (
        <button
          className="button"
          type="button"
          disabled={action.locked || !actionID(action.page, "claim", task.key)}
          onClick={() => send("claim")}
        >
          {action.busy ? "Берём вопрос…" : "Взять и ответить"}
        </button>
      ) : null}
      {canEdit ? (
        <>
          <form className="mutation-form" onSubmit={answerQuestion}>
            <fieldset disabled={action.locked}>
              <label className="form-field">
                Ответ
                <textarea
                  name="answer"
                  rows={7}
                  required
                  value={answer}
                  onChange={(event) => setAnswer(event.target.value)}
                  aria-invalid={Boolean(action.fieldErrors.answer)}
                  aria-describedby="answer-error answer-help"
                />
                <FieldError name="answer" errors={action.fieldErrors} />
              </label>
              <p className="form-help" id="answer-help">
                Ответ сохранится в истории решений. Черновик хранится на этой
                странице.
              </p>
              <div className="form-actions">
                <button
                  className="button"
                  type="submit"
                  disabled={!actionID(action.page, "answer", task.key)}
                >
                  {action.busy ? "Сохраняем…" : "Ответить и завершить"}
                </button>
                <button
                  className="button button-secondary"
                  type="button"
                  disabled={!actionID(action.page, "heartbeat", task.key)}
                  onClick={() => send("heartbeat")}
                >
                  Продлить время
                </button>
              </div>
            </fieldset>
          </form>
          <details className="release-question">
            <summary>Вернуть вопрос в очередь</summary>
            <form
              className="mutation-form"
              onSubmit={(event) => {
                event.preventDefault();
                send("release", { reason });
              }}
            >
              <fieldset disabled={action.locked}>
                <label className="form-field">
                  Причина возврата
                  <textarea
                    name="reason"
                    rows={3}
                    required
                    value={reason}
                    onChange={(event) => setReason(event.target.value)}
                    aria-invalid={Boolean(action.fieldErrors.reason)}
                    aria-describedby={
                      action.fieldErrors.reason ? "reason-error" : undefined
                    }
                  />
                  <FieldError name="reason" errors={action.fieldErrors} />
                </label>
                <button
                  type="submit"
                  className="button button-secondary"
                  disabled={!actionID(action.page, "release", task.key)}
                >
                  Вернуть в очередь
                </button>
              </fieldset>
            </form>
          </details>
        </>
      ) : answer ? (
        <div className="mutation-form">
          <p className="form-help">
            {task.state === "done"
              ? "Ответ записан. Ваш текст на этой странице сохранён для копирования."
              : "Владение вопросом изменилось или истекло. Текст сохранён; обновите состояние и возьмите вопрос заново."}
          </p>
          <label className="form-field">
            Сохранённый черновик
            <textarea readOnly rows={7} value={answer} />
          </label>
          <button
            className="button button-secondary"
            disabled={action.busy || action.phase === "uncertain"}
            onClick={() =>
              void action.refresh(`/app/tasks/${encodeURIComponent(task.key)}`)
            }
          >
            Обновить состояние
          </button>
        </div>
      ) : !canClaim ? (
        <p className="form-help">
          {task.state === "done"
            ? "Ответ уже записан."
            : task.blocked
              ? "Вопрос ожидает выполнения зависимостей."
              : "Сейчас вопрос недоступен для ответа."}
        </p>
      ) : null}
      <ActionNotice
        message={action.message}
        uncertain={action.phase === "uncertain"}
        busy={action.busy}
        onRetry={() => void action.retry()}
      />
    </section>
  );
}
