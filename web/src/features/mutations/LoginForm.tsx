"use client";

import { useRef, useState, type FormEvent } from "react";
import type { MutationProps } from "./contracts";
import { useAction } from "./useAction";
import { ActionNotice } from "./ActionNotice";
import styles from "./LoginForm.module.css";

export function LoginForm({ page, onSuccess }: MutationProps) {
  const form = useRef<HTMLFormElement>(null);
  const fileInput = useRef<HTMLInputElement>(null);
  const fileButton = useRef<HTMLButtonElement>(null);
  const [source, setSource] = useState<"token" | "file">("token");
  const [fileName, setFileName] = useState("");
  const action = useAction(
    page,
    (result) => {
      form.current?.reset();
      setFileName("");
      onSuccess(result);
    },
    "login",
  );
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const values = new FormData(event.currentTarget);
    const body = new FormData();
    body.append("csrf", action.page.csrf ?? "");
    if (source === "file") {
      const file = values.get("token_file");
      if (!(file instanceof File) || !file.size) {
        action.setMessage("Выберите файл с токеном — файл не выбран или пуст");
        return;
      }
      if (file.size > 4096) {
        action.setMessage(
          "Файл слишком большой для токена (больше 4096 байт). Выберите файл, который выдал владелец проекта.",
        );
        return;
      }
      body.append("token_file", file);
    } else {
      const token = String(values.get("token") ?? "").trim();
      if (!token) {
        action.setMessage("Вставьте токен — поле пустое");
        return;
      }
      if (new TextEncoder().encode(token).length > 4096) {
        action.setMessage(
          "Это не похоже на токен: строка длиннее 4096 байт. Вставьте только сам токен.",
        );
        return;
      }
      body.append("token", token);
    }
    void action.send("/web/login", body);
  }
  return (
    <form
      ref={form}
      className="mutation-form"
      onSubmit={submit}
      autoComplete="off"
    >
      <fieldset disabled={action.locked}>
        <div className="form-grid" role="group" aria-label="Способ входа">
          <button
            className={`button ${source === "token" ? "" : "button-secondary"}`}
            type="button"
            aria-pressed={source === "token"}
            onClick={() => {
              form.current?.reset();
              setFileName("");
              action.setMessage("");
              setSource("token");
            }}
          >
            Токен
          </button>
          <button
            className={`button ${source === "file" ? "" : "button-secondary"}`}
            type="button"
            aria-pressed={source === "file"}
            onClick={() => {
              form.current?.reset();
              setFileName("");
              action.setMessage("");
              setSource("file");
            }}
          >
            Файл токена
          </button>
        </div>
        {source === "token" ? (
          <label className="form-field">
            Токен доступа
            <input
              name="token"
              type="password"
              required
              autoComplete="off"
              spellCheck={false}
              aria-describedby="login-help"
            />
          </label>
        ) : (
          <div
            className={`form-field ${styles.filePicker}`}
            role="group"
            aria-labelledby="token-file-label"
          >
            <span id="token-file-label">Файл токена</span>
            <input
              ref={fileInput}
              name="token_file"
              type="file"
              hidden
              onChange={(event) => {
                setFileName(event.currentTarget.files?.[0]?.name ?? "");
                action.setMessage("");
              }}
            />
            <div className={styles.fileActions}>
              <button
                ref={fileButton}
                type="button"
                className={`button button-secondary ${styles.fileButton}`}
                aria-describedby="token-file-status token-file-help"
                onClick={() => fileInput.current?.click()}
              >
                {fileName ? "Заменить файл токена" : "Выбрать файл токена"}
              </button>
              {fileName ? (
                <button
                  type="button"
                  className="button button-link"
                  onClick={() => {
                    if (fileInput.current) fileInput.current.value = "";
                    setFileName("");
                    action.setMessage("");
                    fileButton.current?.focus();
                  }}
                >
                  Убрать файл
                </button>
              ) : null}
            </div>
            <span
              id="token-file-status"
              className={styles.fileStatus}
              role="status"
            >
              {fileName ? `Выбран файл: ${fileName}` : "Файл не выбран"}
            </span>
            <span id="token-file-help" className="form-help">
              Выберите файл с токеном, который прислал владелец или
              администратор Accord.
            </span>
          </div>
        )}
        <p id="login-help" className="form-help">
          Токен — ваш личный ключ доступа. Его выдаёт владелец или администратор
          Accord (строка или файл). Токен не сохраняется в браузере; после входа
          поле очищается.
        </p>
        <button className="button" type="submit" disabled={!action.page.csrf}>
          {action.busy ? "Входим…" : "Войти в Accord"}
        </button>
      </fieldset>
      {action.phase === "blocked" || !action.page.csrf ? (
        <button
          type="button"
          className="button button-secondary"
          disabled={action.busy}
          onClick={() => void action.refresh("/app/login")}
        >
          Обновить форму входа
        </button>
      ) : null}
      <ActionNotice
        message={action.message}
        uncertain={action.phase === "uncertain"}
        busy={action.busy}
        onRetry={() => void action.retry()}
      />
    </form>
  );
}
