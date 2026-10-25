"use client";

import { Select } from "@/components/ui/select";
import { useEffect, useState, type FormEvent, type ReactNode } from "react";
import { usePathname } from "next/navigation";
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
import { CopyX, Layers } from "lucide-react";
import { AnimatedCollapse } from "@/components/motion";
import { useSimilar } from "../duplicates/use-similar";
import { SimilarList } from "../duplicates/similar-list";
import { useProjectScope } from "../projects/project-context";
import {
  epicFromPath,
  loadCreateDraft,
  loadPendingTask,
  saveCreateDraft,
  savePendingTask,
} from "./taskDraftStore";

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
const draftFields = [
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
];

function Required() {
  return (
    <span className="required-mark" aria-hidden>
      {" "}
      *
    </span>
  );
}

export function CreateTaskDialog({
  page,
  onSuccess,
  open,
  onOpenChange,
}: CreateTaskDialogProps) {
  const scope = useProjectScope();
  const path = usePathname();
  const [initial] = useState(() => {
    const base = createDraft(page.draft, page.roles);
    if (page.draft.title) return { draft: base, restored: false };
    const pending = loadPendingTask(scope.key);
    if (pending?.epic_key)
      return {
        draft: { ...base, ...pending, type: "task" },
        restored: false,
        fromPending: true,
      };
    if (pending && (pending.title?.trim() || pending.description?.trim()))
      return {
        draft: { ...base, ...pending, type: "task" },
        restored: true,
        fromPending: true,
      };
    const stored = loadCreateDraft(scope.key);
    if (stored && (stored.title?.trim() || stored.description?.trim()))
      return {
        draft: { ...base, ...createDraft(stored, page.roles), ...stored },
        restored: true,
      };
    const fromPath = epicFromPath(path);
    const epics = page.epics ?? [];
    if (fromPath)
      return { draft: { ...base, epic_key: fromPath }, restored: false };
    if (epics.length === 1)
      return { draft: { ...base, epic_key: epics[0].key }, restored: false };
    return { draft: base, restored: false };
  });
  const [restored, setRestored] = useState(initial.restored);
  useEffect(() => {
    // The parked task is now in this form; never reopen it a second time.
    if ("fromPending" in initial) savePendingTask(scope.key, null);
  }, [initial, scope.key]);
  const [pendingTitle, setPendingTitle] = useState("");
  const [draft, setDraft] = useState<Record<string, string>>(initial.draft);
  const [profile, setProfile] = useState(() =>
    page.draft.profile_id && page.draft.profile_version
      ? `${page.draft.profile_id}@${page.draft.profile_version}`
      : "",
  );
  const [localErrors, setLocalErrors] = useState<Record<string, string>>({});
  const action = useAction(page, (result) => {
    const createdEpic = draft.type === "epic";
    setProfile("");
    setForce(false);
    setForceReason("");
    saveCreateDraft(scope.key, null);
    const parked = loadPendingTask(scope.key);
    if (createdEpic && parked) {
      const epic = epicFromPath(result.redirect);
      savePendingTask(scope.key, epic ? { ...parked, epic_key: epic } : null);
    }
    setDraft(createDraft({}, page.roles));
    onSuccess(result);
    onOpenChange(false);
  });
  useEffect(() => {
    if (!open) return;
    const meaningful = draft.title?.trim() || draft.description?.trim();
    saveCreateDraft(scope.key, meaningful ? draft : null);
  }, [draft, open, scope.key]);
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
  const serverDuplicate = action.errorCode === "DUPLICATE_SUSPECTED";
  const needsConsent = serverDuplicate || (strict && blocking);
  const epics = action.page.epics ?? [];
  const epicsComplete = !action.page.epics_next_cursor;
  const noEpics = isTask && epicsComplete && epics.length === 0;
  const errors = { ...action.fieldErrors, ...localErrors };
  function change(name: string, value: string) {
    if (draft[name] !== value) {
      setForce(false);
      setForceReason("");
      action.clearErrorCode();
      if (action.message) action.setMessage("");
    }
    if (localErrors[name])
      setLocalErrors((previous) => {
        const next = { ...previous };
        delete next[name];
        return next;
      });
    if (name === "type") setLocalErrors({});
    if (name === "role") setProfile("");
    setDraft((previous) => ({ ...previous, [name]: value }));
  }
  function startEpicFirst() {
    savePendingTask(scope.key, {
      type: "task",
      title: draft.title ?? "",
      description: draft.description ?? "",
      priority: draft.priority ?? "2",
      role: draft.role ?? "",
      areas: draft.areas ?? "",
    });
    setPendingTitle(draft.title?.trim() ?? "");
    setLocalErrors({});
    action.setMessage("");
    action.clearErrorCode();
    setDraft((previous) => ({
      ...previous,
      type: "epic",
      title: "",
      description: "",
    }));
  }
  function validate() {
    const found: Record<string, string> = {};
    if (!draft.title?.trim()) found.title = "Введите название";
    else if (draft.title.trim().length > 200)
      found.title = "Название длиннее 200 символов — сократите его";
    // Empty priority means the server default (2); "0" is P0.
    const priority = (draft.priority ?? "").trim();
    if (priority && (!/^\d+$/.test(priority) || Number(priority) > 100))
      found.priority = "Приоритет — целое число от 0 до 100";
    if (isTask && !draft.description?.trim())
      found.description = "Опишите, что нужно сделать и каким будет результат";
    if (isTask && !draft.epic_key?.trim())
      found.epic_key = noEpics
        ? "Сначала создайте эпик — кнопка выше"
        : "Выберите эпик, в который входит задача";
    setLocalErrors(found);
    return Object.keys(found).length === 0;
  }
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!validate()) return;
    if (needsConsent && !force) return;
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
    for (const name of draftFields) fields[name] = draft[name] ?? "";
    fields.title = fields.title.trim();
    fields.description = fields.description.trim();
    fields.priority = (fields.priority ?? "").trim();
    if (!fields.priority) delete fields.priority;
    else fields.priority = String(Number(fields.priority));
    fields.epic_key = fields.epic_key.trim().toUpperCase();
    if (!isTask) {
      fields.epic_key = "";
      fields.role = "";
    } else {
      // owner belongs to epics only; the server rejects it on tasks, so a
      // value typed before switching the type must not be sent.
      fields.owner = "";
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
    label: ReactNode,
    options: {
      required?: boolean;
      placeholder?: string;
      maxLength?: number;
    } = {},
  ) {
    return (
      <label className="form-field">
        <span>
          {label}
          {options.required ? <Required /> : null}
        </span>
        <input
          name={name}
          value={draft[name] ?? ""}
          onChange={(event) => change(name, event.target.value)}
          {...options}
          required={false}
          aria-required={options.required || undefined}
          aria-invalid={Boolean(errors[name])}
          aria-describedby={errors[name] ? `${name}-error` : undefined}
        />
        <FieldError name={name} errors={errors} />
      </label>
    );
  }
  const notFound =
    action.errorCode === "NOT_FOUND" && isTask && draft.epic_key
      ? `Эпик «${draft.epic_key}» не найден в этом проекте. Выберите эпик из списка.`
      : "";
  const epicOptions = epics.some((epic) => epic.key === draft.epic_key)
    ? epics
    : draft.epic_key
      ? [{ key: draft.epic_key, title: "" }, ...epics]
      : epics;
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
        <form className="mutation-form" onSubmit={submit} noValidate>
          <fieldset
            disabled={
              action.locked ||
              action.page.can_continue ||
              action.page.project?.can_write === false
            }
          >
            <p className="form-help">
              Поля со звёздочкой (*) обязательны.
              {!isTask
                ? " Эпик — крупная цель или этап проекта; задачи входят в эпики."
                : ""}
            </p>
            {restored ? (
              <p className="notice notice-row" role="status">
                <span>Восстановлен несохранённый черновик.</span>
                <button
                  type="button"
                  className="button button-ghost button-sm"
                  onClick={() => {
                    saveCreateDraft(scope.key, null);
                    setRestored(false);
                    setLocalErrors({});
                    setDraft(createDraft({}, page.roles));
                  }}
                >
                  Очистить
                </button>
              </p>
            ) : null}
            {!isTask && pendingTitle ? (
              <p className="notice" role="status">
                После создания эпика форма вернётся к задаче «{pendingTitle}».
              </p>
            ) : null}
            {noEpics ? (
              <section className="similar-callout" aria-live="polite">
                <div className="similar-callout-head">
                  <Layers size={15} aria-hidden />
                  <span>Сначала нужен эпик</span>
                </div>
                <p className="muted">
                  Каждая задача входит в эпик — крупную цель проекта, например
                  «Запуск продукта». В этом проекте эпиков пока нет. Создайте
                  эпик — введённая задача сохранится, и форма вернётся к ней.
                </p>
                <div>
                  <button
                    type="button"
                    className="button button-sm"
                    onClick={startEpicFirst}
                  >
                    Создать эпик
                  </button>
                </div>
              </section>
            ) : null}
            <div className="form-grid">
              <label className="form-field">
                Что создать
                <Select
                  name="type"
                  value={draft.type}
                  onChange={(event) => change("type", event.target.value)}
                >
                  <option value="task">Задача</option>
                  <option value="epic">Эпик</option>
                </Select>
              </label>
              <label className="form-field">
                Приоритет
                <input
                  type="number"
                  min="0"
                  max="100"
                  aria-describedby="priority-help"
                  value={draft.priority}
                  onChange={(event) => change("priority", event.target.value)}
                />
                <span id="priority-help" className="form-help">
                  0–100: меньшее число берут в работу раньше; пусто — 2
                </span>
                <FieldError name="priority" errors={errors} />
              </label>
            </div>
            {field("title", "Название", {
              required: true,
              maxLength: 200,
              placeholder: isTask
                ? "Что нужно сделать?"
                : "Например: Запуск продукта",
            })}
            <label className="form-field">
              <span>
                Описание
                {isTask ? <Required /> : " · необязательно"}
              </span>
              <textarea
                name="description"
                rows={5}
                aria-required={isTask || undefined}
                value={draft.description ?? ""}
                placeholder={
                  isTask ? "Что сделать и как понять, что готово" : undefined
                }
                onChange={(event) => change("description", event.target.value)}
                aria-invalid={Boolean(errors.description)}
                aria-describedby={
                  errors.description ? "description-error" : undefined
                }
              />
              <FieldError name="description" errors={errors} />
            </label>
            <AnimatedCollapse
              show={Boolean(matches.length || serverDuplicate || force)}
            >
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
                <AnimatedCollapse show={Boolean(needsConsent || force)}>
                  <label className="checkbox-row">
                    <input
                      type="checkbox"
                      checked={force}
                      onChange={(event) => setForce(event.target.checked)}
                    />
                    Это другая задача — создать всё равно
                  </label>
                </AnimatedCollapse>
                <AnimatedCollapse show={force}>
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
                </AnimatedCollapse>
              </section>
            </AnimatedCollapse>
            {isTask ? (
              <div className="form-grid">
                <label className="form-field">
                  <span>
                    Эпик
                    <Required />
                  </span>
                  {epicsComplete ? (
                    <Select
                      name="epic_key"
                      value={draft.epic_key ?? ""}
                      disabled={noEpics}
                      onChange={(event) =>
                        change("epic_key", event.target.value)
                      }
                      aria-invalid={Boolean(errors.epic_key)}
                      aria-describedby={
                        errors.epic_key ? "epic_key-error" : undefined
                      }
                    >
                      <option value="">
                        {noEpics ? "Эпиков пока нет" : "Выберите эпик"}
                      </option>
                      {epicOptions.map((epic) => (
                        <option key={epic.key} value={epic.key}>
                          {epic.title
                            ? `${epic.key} · ${epic.title}`
                            : epic.key}
                        </option>
                      ))}
                    </Select>
                  ) : (
                    <>
                      <input
                        name="epic_key"
                        list="create-epics"
                        value={draft.epic_key ?? ""}
                        placeholder="E-1"
                        onChange={(event) =>
                          change("epic_key", event.target.value)
                        }
                        aria-invalid={Boolean(errors.epic_key)}
                        aria-describedby={
                          errors.epic_key ? "epic_key-error" : undefined
                        }
                      />
                      <datalist id="create-epics">
                        {epics.map((epic) => (
                          <option key={epic.key} value={epic.key}>
                            {epic.title}
                          </option>
                        ))}
                      </datalist>
                    </>
                  )}
                  <FieldError name="epic_key" errors={errors} />
                </label>
                <label className="form-field">
                  Роль исполнителя
                  <Select
                    name="role"
                    value={draft.role}
                    aria-describedby="role-help"
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
                  </Select>
                  <span id="role-help" className="form-help">
                    {draft.role === "human"
                      ? "Задача для человека попадёт в «Вопросы»: её возьмёт участник-человек."
                      : "Задачу возьмёт агент с этой ролью."}
                  </span>
                  <FieldError name="role" errors={errors} />
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
              {!isTask ? field("owner", "Ответственный · необязательно") : null}
              {field("areas", "Области работы · необязательно", {
                placeholder: "Например: web, api — через запятую",
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
                  (needsConsent && !force) ||
                  (isTask &&
                    draft.role !== "human" &&
                    !currentProfileChoice(
                      profile,
                      draft.role,
                      action.page.assignment_profiles,
                    ))
                }
              >
                {action.busy
                  ? "Создаём…"
                  : isTask
                    ? "Создать задачу"
                    : "Создать эпик"}
              </button>
            </div>
          </fieldset>
          <ActionNotice
            message={notFound || action.message}
            uncertain={action.phase === "uncertain"}
            busy={action.busy}
            onRetry={() => void action.retry()}
          />
        </form>
      </DialogContent>
    </Dialog>
  );
}
