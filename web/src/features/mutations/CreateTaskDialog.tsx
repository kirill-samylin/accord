"use client";

import { useState, type FormEvent } from "react";
import { currentProfileChoice, profilePair } from "@/lib/coordination";
import { ProfileChoice } from "../coordination/profile-choice";
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
import { CopyX } from "lucide-react";
import { useSimilar } from "../duplicates/use-similar";
import { SimilarList } from "../duplicates/similar-list";

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
  const [profile, setProfile] = useState(() =>
    page.draft.profile_id && page.draft.profile_version
      ? `${page.draft.profile_id}@${page.draft.profile_version}`
      : "",
  );
  const action = useAction(page, (result) => {
    setProfile("");
    setDraft(createDraft({}, page.roles));
    onSuccess(result);
    onOpenChange(false);
  });
  const isTask = draft.type !== "epic";
  const isReview = isTask && ["reviewer", "tester"].includes(draft.role);
  const similar = useSimilar({
    title: draft.title ?? "",
    description: draft.description,
    type: draft.type,
    role: draft.role,
    enabled: open && !isReview,
  });
  const matches = similar.data?.similar ?? [];
  const strict = similar.data?.mode === "strict";
  const blocking = matches.some(
    (m) => m.score >= (similar.data?.threshold ?? 1),
  );
  const [force, setForce] = useState(false);
  const [forceReason, setForceReason] = useState("");
  function change(name: string, value: string) {
    if (name === "role") setProfile("");
    setDraft((previous) => ({ ...previous, [name]: value }));
  }
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const useProfile = isTask && draft.role !== "human";
    if (
      useProfile &&
      !currentProfileChoice(
        profile,
        draft.role,
        action.page.assignment_profiles,
      )
    )
      return;
    const fields: Record<string, string> = useProfile
      ? profilePair(profile)
      : {};
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
    if (force) {
      if (!forceReason.trim()) return;
      fields.force = "true";
      fields.force_reason = forceReason.trim();
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
            {matches.length ? (
              <section className="similar-callout" aria-live="polite">
                <div className="similar-callout-head">
                  <CopyX size={15} aria-hidden />
                  <span>Похоже, такая работа уже есть</span>
                </div>
                <p className="muted">
                  Откройте задачу и проверьте, прежде чем создавать новую — так
                  агенты не сделают одно и то же дважды.
                </p>
                <SimilarList items={matches.slice(0, 4)} compact />
                {strict && blocking ? (
                  <label className="checkbox-row">
                    <input
                      type="checkbox"
                      checked={force}
                      onChange={(event) => setForce(event.target.checked)}
                    />
                    Это другая задача — создать всё равно
                  </label>
                ) : null}
                {force ? (
                  <label className="form-field">
                    Чем она отличается
                    <input
                      required
                      maxLength={1000}
                      value={forceReason}
                      placeholder="Например: отдельный экран для администратора"
                      onChange={(event) => setForceReason(event.target.value)}
                    />
                  </label>
                ) : null}
              </section>
            ) : null}
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
            {isTask && draft.role !== "human" ? (
              <ProfileChoice
                value={profile}
                role={draft.role}
                profiles={action.page.assignment_profiles}
                onChange={setProfile}
                id="create-profile"
              />
            ) : null}
            <div className="form-grid">
              {!isTask ? field("owner", "Ответственный") : null}
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
                disabled={
                  !actionID(action.page, "create") ||
                  (strict && blocking && !force) ||
                  (isTask &&
                    draft.role !== "human" &&
                    !currentProfileChoice(
                      profile,
                      draft.role,
                      action.page.assignment_profiles,
                    ))
                }
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
