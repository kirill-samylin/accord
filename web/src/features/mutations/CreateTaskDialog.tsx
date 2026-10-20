"use client";

import { useState, type FormEvent } from "react";
import { actionBody, actionID, type MutationProps } from "./contracts";
import { createDraft } from "./createDraft";
import { useAction } from "./useAction";
import { ActionNotice, FieldError } from "./ActionNotice";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

export type CreateTaskDialogProps = MutationProps & {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};
const fallbackRoles = [
  "coordinator",
  "analyst",
  "architect",
  "developer",
  "reviewer",
  "tester",
  "human",
];
const roleLabels: Record<string, string> = {
  coordinator: "Координатор",
  analyst: "Аналитик",
  architect: "Архитектор",
  developer: "Разработчик",
  reviewer: "Ревьюер",
  tester: "Тестировщик",
  human: "Человек",
};

export function CreateTaskDialog({
  page,
  onSuccess,
  open,
  onOpenChange,
}: CreateTaskDialogProps) {
  const [draft, setDraft] = useState<Record<string, string>>(() =>
    createDraft(page.draft, page.roles),
  );
  const action = useAction(page, (result) => {
    setDraft(createDraft({}, page.roles));
    onSuccess(result);
    onOpenChange(false);
  });
  const isTask = draft.type !== "epic";
  const isReview = isTask && ["reviewer", "tester"].includes(draft.role);
  function change(name: string, value: string) {
    setDraft((previous) => ({ ...previous, [name]: value }));
  }
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const fields: Record<string, string> = {};
    for (const name of [
      "type",
      "title",
      "description",
      "priority",
      "epic_key",
      "role",
      "owner",
      "areas",
      "target_key",
      "target_commit",
    ])
      fields[name] = draft[name] ?? "";
    if (!isTask) {
      fields.epic_key = "";
      fields.role = "";
    }
    if (!isReview) {
      fields.target_key = "";
      fields.target_commit = "";
    }
    void action.send("/web/tasks", actionBody(action.page, "create", fields));
  }
  function field(
    name: string,
    label: string,
    options: {
      required?: boolean;
      placeholder?: string;
      maxLength?: number;
    } = {},
  ) {
    return (
      <label className="form-field">
        {label}
        <input
          name={name}
          value={draft[name] ?? ""}
          onChange={(event) => change(name, event.target.value)}
          {...options}
          aria-invalid={Boolean(action.fieldErrors[name])}
          aria-describedby={
            action.fieldErrors[name] ? `${name}-error` : undefined
          }
        />
        <FieldError name={name} errors={action.fieldErrors} />
      </label>
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
          <DialogTitle>{isTask ? "Новая задача" : "Новый эпик"}</DialogTitle>
        </DialogHeader>
        <form className="mutation-form" onSubmit={submit}>
          <fieldset
            disabled={
              action.locked ||
              action.page.can_continue ||
              action.page.project?.can_write === false
            }
          >
            <div className="form-grid">
              <label className="form-field">
                Что создать
                <select
                  name="type"
                  value={draft.type}
                  onChange={(event) => change("type", event.target.value)}
                >
                  <option value="task">Задача</option>
                  <option value="epic">Эпик</option>
                </select>
              </label>
              <label className="form-field">
                Приоритет
                <input
                  type="number"
                  min="0"
                  max="100"
                  required
                  value={draft.priority}
                  onChange={(event) => change("priority", event.target.value)}
                />
              </label>
            </div>
            {field("title", "Название", {
              required: true,
              maxLength: 200,
              placeholder: "Что нужно сделать?",
            })}
            <label className="form-field">
              Описание
              <textarea
                name="description"
                rows={5}
                required={isTask}
                value={draft.description ?? ""}
                onChange={(event) => change("description", event.target.value)}
                aria-invalid={Boolean(action.fieldErrors.description)}
                aria-describedby={
                  action.fieldErrors.description
                    ? "description-error"
                    : undefined
                }
              />
              <FieldError name="description" errors={action.fieldErrors} />
            </label>
            {isTask ? (
              <div className="form-grid">
                <label className="form-field">
                  Эпик
                  <input
                    name="epic_key"
                    list="create-epics"
                    required
                    value={draft.epic_key ?? ""}
                    placeholder="E-…"
                    onChange={(event) => change("epic_key", event.target.value)}
                    aria-invalid={Boolean(action.fieldErrors.epic_key)}
                  />
                  <datalist id="create-epics">
                    {action.page.epics?.map((epic) => (
                      <option key={epic.key} value={epic.key}>
                        {epic.title}
                      </option>
                    ))}
                  </datalist>
                  <FieldError name="epic_key" errors={action.fieldErrors} />
                </label>
                <label className="form-field">
                  Роль
                  <select
                    name="role"
                    value={draft.role}
                    onChange={(event) => change("role", event.target.value)}
                  >
                    {(action.page.roles?.length
                      ? action.page.roles
                      : fallbackRoles
                    ).map((role) => (
                      <option key={role} value={role}>
                        {roleLabels[role] ?? role}
                      </option>
                    ))}
                  </select>
                  <FieldError name="role" errors={action.fieldErrors} />
                </label>
              </div>
            ) : null}
            <div className="form-grid">
              {field("owner", "Ответственный")}
              {field("areas", "Области работы", {
                placeholder: "Через запятую",
              })}
            </div>
            {isReview ? (
              <div className="form-grid">
                {field("target_key", "Проверяемая задача", {
                  required: true,
                  placeholder: "T-…",
                })}
                {field("target_commit", "Полный commit SHA", {
                  required: true,
                })}
              </div>
            ) : null}
            <div className="form-actions">
              <button
                className="button"
                type="submit"
                disabled={!actionID(action.page, "create")}
              >
                {action.busy ? "Создаём…" : "Создать"}
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
