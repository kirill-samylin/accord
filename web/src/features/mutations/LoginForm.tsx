"use client";

import { useRef, useState, type FormEvent } from "react";
import type { MutationProps } from "./contracts";
import { useAction } from "./useAction";
import { ActionNotice } from "./ActionNotice";

export function LoginForm({ page, onSuccess }: MutationProps) {
  const form = useRef<HTMLFormElement>(null);
  const [source, setSource] = useState<"token" | "file">("token");
  const action = useAction(
    page,
    (result) => {
      form.current?.reset();
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
      if (!(file instanceof File) || !file.size || file.size > 4096) {
        action.setMessage("Выберите файл токена до 4096 байт");
        return;
      }
      body.append("token_file", file);
    } else {
      const token = String(values.get("token") ?? "").trim();
      if (!token || new TextEncoder().encode(token).length > 4096) {
        action.setMessage("Введите токен до 4096 байт");
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
          <label className="form-field">
            Файл токена
            <input
              name="token_file"
              type="file"
              required
              aria-describedby="login-help"
            />
          </label>
        )}
        <p id="login-help" className="form-help">
          Используйте токен своего участника. После входа поле будет очищено.
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
