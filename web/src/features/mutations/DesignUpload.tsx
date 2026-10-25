"use client";

import { Select } from "@/components/ui/select";
import { useRef, useState, type FormEvent } from "react";
import { actionID, uploadBody, type MutationProps } from "./contracts";
import { ActionNotice, FieldError } from "./ActionNotice";
import { useAction } from "./useAction";
import { uploadDraft } from "./uploadDraft";

const LIMIT = 50 * 1024 * 1024;
export function DesignUpload({
  page,
  onSuccess,
  onLockChange,
}: MutationProps & { onLockChange?: (locked: boolean) => void }) {
  const form = useRef<HTMLFormElement>(null);
  const [draft, setDraft] = useState<Record<string, string>>(() =>
    uploadDraft(page.draft, page.designs),
  );
  const [file, setFile] = useState<File | null>(null);
  const action = useAction(
    page,
    (result) => {
      setFile(null);
      form.current?.reset();
      onSuccess(result);
    },
    "action",
    onLockChange,
  );
  function change(name: string, value: string) {
    setDraft((previous) => ({ ...previous, [name]: value }));
  }
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!file || file.size > LIMIT) {
      action.setMessage("Выберите один файл размером до 50 МиБ");
      return;
    }
    const metadata = { ...draft };
    if (metadata.scope === "project") metadata.scope_key = "";
    if (metadata.scope !== "task") metadata.epic_key = "";
    void action.send("/web/designs", uploadBody(action.page, metadata, file));
  }
  return (
    <section className="design-upload" aria-labelledby="design-upload-title">
      <div className="section-heading">
        <h2 id="design-upload-title">
          {draft.design_id ? "Новая версия материала" : "Загрузить дизайн"}
        </h2>
      </div>
      <form ref={form} className="mutation-form" onSubmit={submit}>
        <fieldset
          disabled={
            action.locked ||
            action.page.can_continue ||
            action.page.project?.can_write === false
          }
        >
          <label className="form-field">
            Название
            <input
              name="title"
              required
              maxLength={200}
              value={draft.title ?? ""}
              onChange={(event) => change("title", event.target.value)}
              aria-invalid={Boolean(action.fieldErrors.title)}
            />
            <FieldError name="title" errors={action.fieldErrors} />
          </label>
          <div className="form-grid">
            <label className="form-field">
              Связать с
              <Select
                name="scope"
                value={draft.scope}
                onChange={(event) => change("scope", event.target.value)}
              >
                <option value="project">Проектом</option>
                <option value="epic">Эпиком</option>
                <option value="task">Задачей</option>
              </Select>
            </label>
            {draft.scope !== "project" ? (
              <label className="form-field">
                Ключ {draft.scope === "epic" ? "эпика" : "задачи"}
                <input
                  name="scope_key"
                  required
                  value={draft.scope_key ?? ""}
                  placeholder={draft.scope === "epic" ? "E-…" : "T-…"}
                  onChange={(event) => change("scope_key", event.target.value)}
                />
                <FieldError name="scope_key" errors={action.fieldErrors} />
              </label>
            ) : null}
          </div>
          {draft.scope === "task" ? (
            <label className="form-field">
              Эпик задачи · необязательно
              <input
                name="epic_key"
                value={draft.epic_key ?? ""}
                placeholder="E-…"
                onChange={(event) => change("epic_key", event.target.value)}
              />
              <FieldError name="epic_key" errors={action.fieldErrors} />
            </label>
          ) : null}
          {draft.design_id ? (
            <div className="form-help">
              <p>Материал {draft.design_id}. Предыдущие версии сохранятся.</p>
              <button
                className="button button-secondary"
                type="button"
                onClick={() => change("design_id", "")}
              >
                Создать отдельный материал
              </button>
            </div>
          ) : null}
          <label className="form-field file-drop">
            Файл
            <input
              type="file"
              name="file"
              required
              accept=".png,.jpg,.jpeg,.webp,.pdf,.zip,.html,.htm,.svg"
              onChange={(event) => {
                setFile(event.target.files?.[0] ?? null);
                action.setMessage("");
              }}
              aria-describedby="upload-help"
            />
            <FieldError name="file" errors={action.fieldErrors} />
          </label>
          <p className="form-help" id="upload-help">
            PNG, JPEG, WEBP, PDF, ZIP, HTML или SVG · до 50 МиБ
            {file
              ? ` · ${(file.size / 1024 / 1024).toFixed(1)} МиБ выбрано`
              : ""}
          </p>
          <button
            className="button"
            type="submit"
            disabled={!actionID(action.page, "upload")}
          >
            {action.busy
              ? "Загружаем…"
              : draft.design_id
                ? "Загрузить новую версию"
                : "Загрузить файл"}
          </button>
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
