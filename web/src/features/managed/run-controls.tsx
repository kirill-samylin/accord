"use client";
import { useRef, useState } from "react";
import { Play, Square } from "lucide-react";
import { AnimatedCollapse } from "@/components/motion";
import {
  commandReason,
  hostOnline,
  messageError,
  managedConflictVersion,
  type ManagedRun,
} from "@/lib/managed-agents";
import {
  actionBody,
  actionID,
  type MutationProps,
} from "../mutations/contracts";
import { ActionNotice, FieldError } from "../mutations/ActionNotice";
import { useAction } from "../mutations/useAction";
import { useProjectScope } from "../projects/project-context";
export function RunControls({
  page,
  onSuccess,
  run,
}: MutationProps & { run: ManagedRun }) {
  const [text, setText] = useState("");
  const [version, setVersion] = useState<number>();
  const submittedKind = useRef<string | undefined>(undefined);
  const [validation, setValidation] = useState("");
  const [sent, setSent] = useState(false);
  const action = useAction(page, (result) => {
    if (submittedKind.current !== "stop") setText("");
    setVersion(undefined);
    setSent(true);
    onSuccess(result);
  });
  const scope = useProjectScope();
  const current = action.page.managed_run?.run ?? run;
  const host = action.page.managed?.hosts.find(
    (entry) => entry.id === current.host_id,
  );
  const canWrite =
    action.page.project?.can_write !== false && !action.page.can_continue;
  const changed =
    version !== undefined &&
    (version !== current.version ||
      action.errorCode.toLowerCase() === "conflict") &&
    Boolean(action.message);
  const conflictVersion = managedConflictVersion(
    action.page.view,
    action.page.managed_run,
    run.id,
    version,
  );
  const observedAt = action.page.managed?.observed_at;
  const locked = action.locked || scope.locked;
  const hasAction = Boolean(actionID(action.page, "managed:command"));
  function send(kind: "stop" | "resume" | "clarify") {
    setSent(false);
    const error = kind === "stop" ? "" : messageError(text, kind === "clarify");
    setValidation(error);
    if (
      changed ||
      error ||
      commandReason(kind, current, host, canWrite, observedAt)
    )
      return;
    submittedKind.current = kind;
    setVersion(current.version);
    void action.send(
      "/web/managed/command",
      actionBody(action.page, "managed:command", {
        run_id: current.id,
        version: String(current.version),
        kind,
        ...(kind !== "stop" && text.trim() ? { text } : {}),
      }),
    );
  }
  const stopReason = commandReason("stop", current, host, canWrite, observedAt);
  const resumeReason = commandReason(
    "resume",
    current,
    host,
    canWrite,
    observedAt,
  );
  const clarifyReason = commandReason(
    "clarify",
    current,
    host,
    canWrite,
    observedAt,
  );
  return (
    <section
      className="settings-section managed-controls"
      aria-labelledby="run-control-title"
    >
      <h2 id="run-control-title">Управление работой</h2>
      <div className="managed-row">
        <button
          className="button button-secondary"
          disabled={locked || changed || !hasAction || Boolean(stopReason)}
          title={stopReason || undefined}
          onClick={() => send("stop")}
        >
          <Square size={14} />
          Остановить
        </button>
        <button
          className="button"
          disabled={locked || changed || !hasAction || Boolean(resumeReason)}
          title={resumeReason || undefined}
          onClick={() => send("resume")}
        >
          <Play size={14} />
          Продолжить
        </button>
      </div>
      {resumeReason ? <p className="form-help">{resumeReason}</p> : null}
      {stopReason && stopReason !== resumeReason ? (
        <p className="form-help">{stopReason}</p>
      ) : null}
      {!hostOnline(host, observedAt) ? (
        <div className="notice">
          По последним данным сервер не видит компьютер на связи. Остановка
          будет доставлена после подключения; последняя запись о процессе не
          доказывает, что он уже остановлен. При потере связи worker
          останавливает свои процессы локально.
        </div>
      ) : null}
      {current.state === "needs_recovery" || current.state === "failed" ? (
        <div className="notice">
          Владелец должен проверить состояние на компьютере. Новая попытка без
          проверки старого процесса недоступна.
        </div>
      ) : null}
      <label className="form-field" htmlFor="managed-message">
        Уточнение или текст продолжения
        <textarea
          id="managed-message"
          rows={4}
          value={text}
          disabled={locked || !canWrite || !host?.can_control}
          onChange={(event) => {
            setText(event.target.value);
            setValidation("");
            setSent(false);
          }}
          placeholder="Опишите, что нужно изменить или учесть"
          aria-describedby="managed-message-help"
          aria-invalid={Boolean(validation || action.fieldErrors.text)}
        />
      </label>
      <p id="managed-message-help" className="form-help">
        Во время работы уточнение остаётся в очереди до завершения текущего
        шага. Чтобы прервать шаг, нажмите «Остановить». Для остановленного
        агента уточнение сохранится и будет передано после «Продолжить».
      </p>
      <FieldError
        name="text"
        errors={{
          ...action.fieldErrors,
          ...(validation ? { text: validation } : {}),
        }}
      />
      <button
        className="button button-secondary"
        disabled={
          locked ||
          changed ||
          !hasAction ||
          Boolean(clarifyReason) ||
          !text.trim()
        }
        title={clarifyReason || undefined}
        onClick={() => send("clarify")}
      >
        Сохранить уточнение
      </button>
      {clarifyReason ? <p className="form-help">{clarifyReason}</p> : null}
      {changed ? (
        <div className="notice" role="status">
          <p>
            Состояние работы изменилось. Ваш текст сохранён. Проверьте его перед
            новой отправкой.
          </p>
          <button
            className="button button-secondary"
            disabled={locked || conflictVersion === undefined}
            onClick={() => {
              if (conflictVersion !== undefined) setVersion(conflictVersion);
              action.clearErrorCode();
            }}
          >
            Использовать актуальное состояние
          </button>
          {conflictVersion === undefined ? (
            <p className="form-help">
              Актуальная версия работы ещё не получена. Сначала нажмите
              «Проверить актуальное состояние».
            </p>
          ) : null}
        </div>
      ) : null}
      {!hasAction ? (
        <p className="form-help">
          Действия сейчас недоступны. Обновите страницу или восстановите рабочую
          сессию.
        </p>
      ) : null}
      <AnimatedCollapse show={action.busy}>
        <p className="form-help" role="status">
          Отправляем команду…
        </p>
      </AnimatedCollapse>
      <AnimatedCollapse show={sent}>
        <p role="status" className="form-help">
          Команда сохранена. Её доставку и исполнение можно проверить в истории
          ниже.
        </p>
      </AnimatedCollapse>
      <ActionNotice
        message={action.message}
        uncertain={action.phase === "uncertain"}
        busy={action.busy}
        onRetry={() => void action.retry()}
      />
      {action.message && !locked ? (
        <button
          className="button button-ghost"
          onClick={() => void action.refresh(`/app/managed/runs/${current.id}`)}
        >
          Проверить актуальное состояние
        </button>
      ) : null}
    </section>
  );
}
