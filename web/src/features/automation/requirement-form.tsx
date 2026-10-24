"use client";
import { useState, type FormEvent } from "react";
import type { MutationProps } from "../mutations/contracts";
import { actionBody, actionID } from "../mutations/contracts";
import { useAction } from "../mutations/useAction";
import { ActionNotice, FieldError } from "../mutations/ActionNotice";
import { requirementError } from "@/lib/automation";
import { date } from "@/lib/format";
export function RequirementForm({
  page,
  onSuccess,
  initialText,
  onLockChange,
}: MutationProps & {
  initialText?: string;
  onLockChange?: (locked: boolean) => void;
}) {
  const [text, setText] = useState(initialText ?? page.draft?.text ?? "");
  const [selection, setSelection] = useState("");
  const [error, setError] = useState("");
  const action = useAction(page, onSuccess, "action", onLockChange);
  const policies = action.page.automation?.policies ?? [];
  const policy = policies.find(
    (p) => `${p.id}:${p.version}` === selection && p.state === "active",
  );
  function submit(event: FormEvent) {
    event.preventDefault();
    const validation = requirementError(text);
    setError(validation);
    if (validation || (selection && !policy)) return;
    void action.send(
      "/web/automation/submit",
      actionBody(action.page, "automation:submit", {
        text,
        ...(policy
          ? { policy_id: policy.id, policy_version: String(policy.version) }
          : {}),
      }),
    );
  }
  return (
    <section className="automation-card screen-stack">
      <h2>Что нужно сделать</h2>
      <form className="mutation-form" onSubmit={submit}>
        <fieldset
          disabled={action.locked || action.page.project?.can_write === false}
        >
          <label className="form-field">
            Требование
            <textarea
              rows={7}
              placeholder="Опишите результат, ограничения и как проверить готовность. Аналитик разобьёт работу на задачи, разработчик выполнит их, ревьюер и тестировщик проверят."
              value={text}
              onChange={(e) => setText(e.target.value)}
              aria-invalid={Boolean(error)}
            />
            <FieldError name="text" errors={{ text: error }} />
          </label>
          <label className="form-field">
            Команда и разрешение
            <select
              value={selection}
              onChange={(e) => setSelection(e.target.value)}
            >
              <option value="">Сохранить черновик без запуска</option>
              {policies
                .filter((p) => p.state === "active")
                .map((p) => (
                  <option key={p.id} value={`${p.id}:${p.version}`}>
                    {p.name} · версия {p.version}
                  </option>
                ))}
              {selection && !policy ? (
                <option disabled value={selection}>
                  Разрешение изменилось — выберите актуальную версию
                </option>
              ) : null}
            </select>
          </label>
          {policy ? (
            <div className="notice">
              <strong>{policy.name}</strong>
              <p>
                Папка: {policy.workspace}. До {policy.limits.plan_tasks} задач,{" "}
                {policy.limits.fix_total} исправлений,{" "}
                {policy.limits.native_invocations} запусков. Разрешение до{" "}
                {date(policy.expires_at)}.
              </p>
              <p>
                После отправки цикл начнёт анализ, когда компьютер подтвердит
                запуск. Вопросы и причины ожидания появятся здесь.
              </p>
            </div>
          ) : (
            <p className="form-help">
              Черновик не запускает агентов. Для выполнения выберите действующее
              разрешение команды.
            </p>
          )}
          <div className="form-actions">
            <button
              className="button"
              type="submit"
              disabled={
                !actionID(action.page, "automation:submit") ||
                Boolean(selection && !policy)
              }
            >
              {action.busy
                ? "Отправляем…"
                : policy
                  ? "Начать выполнение"
                  : "Сохранить требование"}
            </button>
          </div>
        </fieldset>
        <ActionNotice
          message={action.message}
          uncertain={action.phase === "uncertain"}
          busy={action.busy}
          onRetry={() => void action.retry()}
        />
      </form>
    </section>
  );
}
