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
  onSuccess: (result: CatalogMutationResult) => void;
}) {
  const action = useCatalogAction(page, onSuccess, refreshPath);
  const form = useRef<HTMLFormElement>(null);
  const autoSent = useRef(false);
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
      onSubmit={(event) => {
        event.preventDefault();
        if (confirm && !window.confirm(confirm)) return;
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
        action.send(endpoint, actionKey, values);
      }}
    >
      <fieldset
        disabled={
          disabled || action.locked || !action.page.registry.actions[actionKey]
        }
      >
        {children}
        <button className="button button-secondary" type="submit">
          {action.busy ? "Сохраняем…" : title}
        </button>
      </fieldset>
      {Object.keys(errors).length ? (
        <ul className="registry-errors" role="alert">
          {Object.entries(errors).map(([name, message]) => (
            <li key={name}>{message}</li>
          ))}
        </ul>
      ) : null}
      <ActionNotice
        message={action.message}
        busy={action.busy}
        uncertain={action.phase === "uncertain"}
        onRetry={action.retry}
      />
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
