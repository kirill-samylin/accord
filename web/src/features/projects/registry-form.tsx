"use client";
import { useEffect, useRef, useState, type ReactNode } from "react";
import type { CatalogPage } from "@/lib/projects";
import type { CatalogMutationResult } from "@/lib/types";
import { ActionNotice } from "../mutations/ActionNotice";
import { useCatalogAction } from "./use-catalog-action";
export function RegistryForm({
  page,
  actionKey,
  endpoint,
  children,
  title,
  fields = {},
  confirm,
  disabled,
  onSuccess,
  refreshPath,
  validate,
  autoSubmit,
  savedMessage,
}: {
  page: CatalogPage;
  actionKey: string;
  endpoint: string;
  children?: ReactNode;
  title: string;
  fields?: Record<string, string>;
  confirm?: string;
  disabled?: boolean;
  refreshPath?: string;
  validate?: (values: URLSearchParams) => Record<string, string>;
  autoSubmit?: boolean;
  /** Shown after a successful send until the form is edited again. */
  savedMessage?: string;
  onSuccess: (result: CatalogMutationResult) => void;
}) {
  const [saved, setSaved] = useState(false);
  const action = useCatalogAction(
    page,
    (result) => {
      setSaved(true);
      onSuccess(result);
    },
    refreshPath,
  );
  const form = useRef<HTMLFormElement>(null);
  const autoSent = useRef(false);
  const submit = useRef<HTMLButtonElement>(null);
  const status = useRef<HTMLDivElement>(null);
  const wasBusy = useRef(false);
  // The fieldset is disabled while sending, which drops keyboard focus to
  // <body>. When the send settles, focus returns to the submit button, or to
  // the result notice when the form stays locked (or the button is gone).
  useEffect(() => {
    if (action.busy) {
      wasBusy.current = true;
      return;
    }
    if (!wasBusy.current) return;
    wasBusy.current = false;
    const active = document.activeElement;
    if (active && active !== document.body) return;
    if (submit.current && !submit.current.closest("fieldset")?.disabled)
      submit.current.focus();
    else status.current?.focus();
  }, [action.busy, action.phase]);
  const [localErrors, setLocalErrors] = useState<Record<string, string>>({});
  const ready = Boolean(action.page.registry.actions[actionKey]);
  useEffect(() => {
    if (!autoSubmit || autoSent.current || !ready || action.locked) return;
    autoSent.current = true;
    form.current?.requestSubmit();
  }, [autoSubmit, ready, action.locked]);
  const errors = { ...action.errors, ...localErrors };
  return (
    <form
      ref={form}
      noValidate={Boolean(validate)}
      className="mutation-form registry-form"
      onInput={(event) => {
        const name = (event.target as HTMLInputElement).name;
        setSaved(false);
        if (!name) return;
        action.clearError(name);
        setLocalErrors((current) => {
          if (!(name in current)) return current;
          const next = { ...current };
          delete next[name];
          return next;
        });
      }}
      onSubmit={(event) => {
        event.preventDefault();
        if (confirm && !window.confirm(confirm)) return;
        setSaved(false);
        const values = new URLSearchParams();
        new FormData(event.currentTarget).forEach((value, key) => {
          if (typeof value === "string") values.set(key, value);
        });
        for (const [name, value] of Object.entries(fields))
          values.set(name, value);
        if (validate) {
          const found = validate(values);
          setLocalErrors(found);
          if (Object.keys(found).length) return;
        }
        void action.send(endpoint, actionKey, values);
      }}
    >
      <fieldset
        disabled={
          disabled || action.locked || !action.page.registry.actions[actionKey]
        }
      >
        {children}
        <button ref={submit} className="button button-secondary" type="submit">
          {action.busy ? "Сохраняем…" : title}
        </button>
      </fieldset>
      <div ref={status} tabIndex={-1} className="registry-status">
        {Object.keys(errors).length ? (
          <ul className="registry-errors" role="alert">
            {Object.entries(errors).map(([name, message]) => (
              <li key={name}>{message}</li>
            ))}
          </ul>
        ) : null}
        {saved && savedMessage && !action.message ? (
          <p className="notice" role="status">
            {savedMessage}
          </p>
        ) : null}
        <ActionNotice
          message={action.message}
          busy={action.busy}
          uncertain={action.phase === "uncertain"}
          onRetry={action.retry}
        />
      </div>
      {action.phase === "blocked" ? (
        <button
          type="button"
          className="button button-ghost"
          onClick={() => void action.refresh()}
        >
          Попробовать снова
        </button>
      ) : null}
    </form>
  );
}
