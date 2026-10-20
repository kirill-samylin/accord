"use client";
import { useState, type FormEvent } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  canCoordinate,
  coordinationDraft,
  coordinationFields,
  coordinationTitles,
  currentProfileChoice,
  latestPositiveReview,
  type CoordinationDraft,
  type CoordinationOperation,
} from "@/lib/coordination";
import {
  actionBody,
  actionID,
  type MutationProps,
} from "../mutations/contracts";
import { ActionNotice } from "../mutations/ActionNotice";
import { useAction } from "../mutations/useAction";
import { CoordinationFields } from "./coordination-fields";
export function CoordinationDialog({
  page,
  onSuccess,
  operation,
  checkRole,
  onClose,
}: MutationProps & {
  operation: CoordinationOperation;
  checkRole?: string;
  onClose: () => void;
}) {
  const [baseTask, setBaseTask] = useState(page.task!);
  const [draft, setDraft] = useState(() => {
    const value = coordinationDraft(page.task!, checkRole);
    if (operation === "checks") value.profile = "";
    return value;
  });
  const [localError, setLocalError] = useState("");
  const action = useAction(page, (result) => {
    onClose();
    onSuccess(result);
  });
  const currentTask = action.page.task;
  const changed = currentTask && String(currentTask.version) !== draft.version;
  const role = operation === "checks" ? draft.role : baseTask.role;
  const profileCurrent = currentProfileChoice(
    draft.profile,
    role,
    action.page.assignment_profiles,
  );
  const pending = action.phase === "sending" || action.phase === "uncertain";
  const permitted = canCoordinate(action.page);
  const qualified =
    operation !== "checks" ||
    draft.role !== "tester" ||
    Boolean(latestPositiveReview(action.page.coordination));
  const op = `coord:${operation}`;
  function change(name: keyof CoordinationDraft, value: string) {
    setDraft((previous) => ({ ...previous, [name]: value }));
    setLocalError("");
  }
  function submit(event: FormEvent) {
    event.preventDefault();
    if (
      !permitted ||
      changed ||
      !qualified ||
      ((operation === "assign" || operation === "checks") && !profileCurrent)
    )
      return;
    try {
      void action.send(
        `/web/tasks/${baseTask.key}/${operation}`,
        actionBody(
          action.page,
          op,
          coordinationFields(
            operation,
            draft,
            baseTask,
            action.page.session.session_id,
          ),
          baseTask.key,
        ),
      );
    } catch (error) {
      setLocalError(
        error instanceof Error ? error.message : "Проверьте поля формы",
      );
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !pending) onClose();
      }}
    >
      <DialogContent
        className="coordination-dialog"
        aria-describedby="coordination-dialog-help"
      >
        <DialogHeader>
          <DialogTitle>
            {coordinationTitles[operation]} · {baseTask.key}
          </DialogTitle>
        </DialogHeader>
        <p className="form-help" id="coordination-dialog-help">
          Действие относится к текущему проекту. Владение задачей определяется
          арендой сессии.
        </p>
        <form className="mutation-form" onSubmit={submit}>
          <fieldset disabled={action.locked || !permitted}>
            <CoordinationFields
              operation={operation}
              draft={draft}
              task={baseTask}
              page={action.page}
              change={change}
            />
            {changed ? (
              <div className="notice" role="status">
                <p>
                  Задача изменена: версия {currentTask!.version}. Ваш черновик
                  версии {draft.version} сохранён.
                </p>
                <button
                  className="button button-secondary"
                  type="button"
                  onClick={() => {
                    setBaseTask(currentTask!);
                    change("version", String(currentTask!.version));
                  }}
                >
                  Применить черновик к версии {currentTask!.version}
                </button>
              </div>
            ) : null}
            {!qualified ? (
              <p className="form-help" role="status">
                Для QA нужно актуальное положительное ревью. Обновите состояние.
              </p>
            ) : null}
            <div className="form-actions">
              <span className="form-help">Основа: версия {draft.version}</span>
              <button
                className="button"
                disabled={
                  Boolean(changed) ||
                  !qualified ||
                  ((operation === "assign" || operation === "checks") &&
                    !profileCurrent) ||
                  !actionID(action.page, op, baseTask.key)
                }
              >
                {action.busy
                  ? "Сохраняем…"
                  : operation === "checks"
                    ? "Создать проверку"
                    : "Сохранить"}
              </button>
            </div>
          </fieldset>
          <ActionNotice
            message={localError || action.message}
            uncertain={action.phase === "uncertain"}
            busy={action.busy}
            onRetry={() => void action.retry()}
          />
          {!pending ? (
            <button
              type="button"
              className="button button-ghost"
              onClick={() => void action.refresh(`/app/tasks/${baseTask.key}`)}
            >
              Проверить актуальное состояние
            </button>
          ) : null}
        </form>
      </DialogContent>
    </Dialog>
  );
}
