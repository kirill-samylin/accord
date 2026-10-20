"use client";

import { useState, type FormEvent } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { actionBody, actionID, type MutationProps } from "./contracts";
import { ActionNotice, FieldError } from "./ActionNotice";
import { useAction } from "./useAction";

export function EditTaskDialog({
  page,
  onSuccess,
  open,
  onOpenChange,
}: MutationProps & { open: boolean; onOpenChange: (open: boolean) => void }) {
  const [title, setTitle] = useState(page.task?.title ?? "");
  const [description, setDescription] = useState(page.task?.description ?? "");
  const action = useAction(page, (result) => {
    onSuccess(result);
    onOpenChange(false);
  });
  const task = action.page.task;
  if (!task || !page.can_update || page.project?.can_write === false)
    return null;
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!task) return;
    // A live lease's generation is fenced by Go. Unleased requester edits use
    // zero; metadata editing never takes ownership or changes task status.
    const generation =
      task.lease?.session_id === action.page.session.session_id
        ? task.generation
        : 0;
    void action.send(
      `/web/tasks/${encodeURIComponent(task.key)}/update`,
      actionBody(
        action.page,
        "update",
        { title, description, generation: String(generation) },
        task.key,
      ),
    );
  }
  return (
    <Dialog
      open={open}
      onOpenChange={(value) => {
        if (!action.locked) onOpenChange(value);
      }}
    >
      <DialogContent aria-describedby={undefined}>
        <DialogHeader>
          <DialogTitle>Изменить {task.key}</DialogTitle>
        </DialogHeader>
        <form className="mutation-form" onSubmit={submit}>
          <fieldset disabled={action.locked || !action.page.can_update}>
            <label className="form-field">
              Название
              <input
                name="title"
                maxLength={200}
                required
                value={title}
                onChange={(event) => setTitle(event.target.value)}
                aria-invalid={Boolean(action.fieldErrors.title)}
                aria-describedby={
                  action.fieldErrors.title ? "title-error" : undefined
                }
              />
              <FieldError name="title" errors={action.fieldErrors} />
            </label>
            <label className="form-field">
              Описание
              <textarea
                name="description"
                rows={8}
                value={description}
                onChange={(event) => setDescription(event.target.value)}
                aria-invalid={Boolean(action.fieldErrors.description)}
                aria-describedby={
                  action.fieldErrors.description
                    ? "description-error"
                    : undefined
                }
              />
              <FieldError name="description" errors={action.fieldErrors} />
            </label>
            <div className="form-actions">
              <button
                className="button"
                disabled={!actionID(action.page, "update", task.key)}
              >
                {action.busy ? "Сохраняем…" : "Сохранить изменения"}
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
      </DialogContent>
    </Dialog>
  );
}
