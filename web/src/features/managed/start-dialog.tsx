"use client";
import { useState, type FormEvent } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  profileOption,
  providerName,
  selectedProfile,
  startErrors,
  startFields,
  type StartDraft,
} from "@/lib/managed-agents";
import { label } from "@/lib/format";
import { ProjectLink as Link } from "@/components/project-link";
import {
  actionBody,
  actionID,
  type MutationProps,
} from "../mutations/contracts";
import { ActionNotice, FieldError } from "../mutations/ActionNotice";
import { useAction } from "../mutations/useAction";
export function StartDialog({
  page,
  open,
  onOpenChange,
  onSuccess,
}: MutationProps & {
  open: boolean;
  onOpenChange: (value: boolean) => void;
}) {
  const [draft, setDraft] = useState<StartDraft>({
    host_id: "",
    workspace: "",
    provider: "",
    profile: "",
    prompt: "",
    task_key: "",
  });
  const [errors, setErrors] = useState<Record<string, string>>({});
  const action = useAction(page, (result) => {
    onSuccess(result);
    onOpenChange(false);
  });
  const data = action.page.managed ?? { hosts: [], runs: [], observed_at: "" };
  const profiles = action.page.profiles ?? [];
  const host = data.hosts.find((entry) => entry.id === draft.host_id);
  const profile = selectedProfile(draft.profile, profiles);
  const fields = { ...action.fieldErrors, ...errors };
  const pending = action.phase === "sending" || action.phase === "uncertain";
  const canWrite =
    action.page.project?.can_write !== false && !action.page.can_continue;
  function change(name: keyof StartDraft, value: string) {
    setDraft((previous) =>
      name === "host_id"
        ? { ...previous, host_id: value, workspace: "", provider: "" }
        : { ...previous, [name]: value },
    );
    setErrors((previous) => ({ ...previous, [name]: "" }));
  }
  function submit(event: FormEvent) {
    event.preventDefault();
    const validation = startErrors(draft, data, profiles, canWrite);
    setErrors(validation);
    if (Object.keys(validation).length || !profile) return;
    void action.send(
      "/web/managed/start",
      actionBody(action.page, "managed:start", startFields(draft, profile)),
    );
  }
  return (
    <Dialog
      open={open}
      onOpenChange={(value) => {
        if (!pending) onOpenChange(value);
      }}
    >
      <DialogContent
        className="managed-dialog"
        aria-describedby="managed-start-help"
      >
        <DialogHeader>
          <DialogTitle>Ручной запуск агента</DialogTitle>
        </DialogHeader>
        <p className="form-help" id="managed-start-help">
          Для изменения файлов нужна существующая задача и её аренда агентом.
          Новое поручение с созданием задач, разработкой, ревью и проверкой
          отправьте через{" "}
          <Link href="/app/automation">«Выполнить требование»</Link>.
        </p>
        <form className="mutation-form" onSubmit={submit}>
          <fieldset disabled={action.locked || !canWrite}>
            <label className="form-field">
              Компьютер
              <select
                value={draft.host_id}
                onChange={(event) => change("host_id", event.target.value)}
                aria-invalid={Boolean(fields.host_id)}
              >
                <option value="">Выберите компьютер</option>
                {data.hosts.map((entry) => (
                  <option
                    key={entry.id}
                    value={entry.id}
                    disabled={!entry.can_control || entry.state !== "active"}
                  >
                    {entry.name}
                    {!entry.can_control ? " · управление не разрешено" : ""}
                    {entry.state !== "active" ? " · отключён" : ""}
                  </option>
                ))}
              </select>
              <FieldError name="host_id" errors={fields} />
            </label>
            <div className="form-grid">
              <label className="form-field">
                Рабочая папка
                <select
                  value={draft.workspace}
                  onChange={(event) => change("workspace", event.target.value)}
                  aria-invalid={Boolean(fields.workspace)}
                >
                  <option value="">Выберите папку</option>
                  {host?.workspaces.map((workspace) => (
                    <option key={workspace}>{workspace}</option>
                  ))}
                </select>
                <FieldError name="workspace" errors={fields} />
              </label>
              <label className="form-field">
                Агент
                <select
                  value={draft.provider}
                  onChange={(event) => change("provider", event.target.value)}
                  aria-invalid={Boolean(fields.provider)}
                >
                  <option value="">Выберите агента</option>
                  {host?.providers.map((provider) => (
                    <option key={provider} value={provider}>
                      {providerName(provider)}
                    </option>
                  ))}
                </select>
                <FieldError name="provider" errors={fields} />
              </label>
            </div>
            <label className="form-field">
              Профиль и версия
              <select
                value={draft.profile}
                onChange={(event) => change("profile", event.target.value)}
                aria-invalid={Boolean(
                  fields.profile || (draft.profile && !profile),
                )}
              >
                <option value="">Выберите профиль</option>
                {draft.profile && !profile ? (
                  <option value={draft.profile} disabled>
                    Выбранная версия изменилась. Выберите актуальную явно.
                  </option>
                ) : null}
                {profiles
                  .filter((entry) => entry.state === "active")
                  .map((entry) => (
                    <option key={entry.id} value={profileOption(entry)}>
                      {entry.name} · {label(entry.role)} · версия{" "}
                      {entry.version}
                    </option>
                  ))}
              </select>
              <FieldError name="profile" errors={fields} />
            </label>
            <label className="form-field">
              Что нужно сделать
              <textarea
                rows={5}
                value={draft.prompt}
                onChange={(event) => change("prompt", event.target.value)}
                aria-invalid={Boolean(fields.prompt)}
                placeholder="Опишите ожидаемый результат и ограничения"
              />
              <FieldError name="prompt" errors={fields} />
            </label>
            <label className="automation-check">
              <input
                type="checkbox"
                checked={Boolean(draft.read_only)}
                onChange={(event) => {
                  const readOnly = event.target.checked;
                  setDraft((previous) => ({
                    ...previous,
                    read_only: readOnly,
                  }));
                  setErrors((previous) => ({ ...previous, task_key: "" }));
                }}
              />
              <span>
                Без задачи, только чтение и диагностика без изменения файлов
              </span>
            </label>
            {draft.read_only ? (
              <p className="form-help">
                Это ограничение в инструкции агенту, не изоляция операционной
                системы. Для создания и изменения файлов используйте выполнение
                требования.
              </p>
            ) : null}
            <label className="form-field">
              Существующая задача в Accord
              <input
                value={draft.task_key}
                disabled={Boolean(draft.read_only)}
                required={!draft.read_only}
                aria-invalid={Boolean(fields.task_key)}
                onChange={(event) => change("task_key", event.target.value)}
                placeholder="Например, T-12"
              />
              <span className="form-help">
                Ссылка задаёт контекст. Агент отдельно получает задачу с
                проверкой прав, назначения и готовности.
              </span>
              <FieldError name="task_key" errors={fields} />
            </label>
            <p className="form-help">
              Одна работа на одну папку. Модель и разрешения определяет владелец
              компьютера в локальных настройках.
            </p>
            <div className="form-actions">
              <button
                className="button"
                type="submit"
                disabled={!actionID(action.page, "managed:start")}
              >
                {action.busy
                  ? "Отправляем…"
                  : draft.read_only
                    ? "Запустить только чтение"
                    : "Запустить по задаче"}
              </button>
            </div>
          </fieldset>
          <ActionNotice
            message={action.message}
            uncertain={action.phase === "uncertain"}
            busy={action.busy}
            onRetry={() => void action.retry()}
          />
          {action.message && !pending ? (
            <button
              className="button button-ghost"
              type="button"
              onClick={() => void action.refresh("/app/managed")}
            >
              Обновить доступные компьютеры и профили
            </button>
          ) : null}
        </form>
      </DialogContent>
    </Dialog>
  );
}
