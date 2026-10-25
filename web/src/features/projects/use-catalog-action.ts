"use client";
import { useEffect, useRef, useState } from "react";
import { usePathname } from "next/navigation";
import { pageQueryKey, readPage, submitAction } from "@/lib/api";
import { useQueryClient } from "@tanstack/react-query";
import type { CatalogPage } from "@/lib/projects";
import type { CatalogMutationResult } from "@/lib/types";
import { catalogBody } from "@/lib/catalog-action";
import { withMutationDeadline } from "../mutations/deadline";
import { useProjectScope } from "./project-context";
type Phase = "idle" | "sending" | "uncertain" | "blocked" | "settled";
// A definitive field rejection (400 VALIDATION, a well-formed id that does
// not exist, or a conflict the server marks editable such as a duplicate)
// leaves the form editable with the message shown. The action id may already
// be bound to the rejected payload, so the next send first reads a fresh page
// (new action ids) instead of replaying the old one.
function definitiveFieldError(result: CatalogMutationResult) {
  const code = result.error?.code?.toUpperCase() ?? "";
  return (
    result.editable === true || code === "VALIDATION" || code === "NOT_FOUND"
  );
}
export function useCatalogAction(
  initialPage: CatalogPage,
  onSuccess: (value: CatalogMutationResult) => void,
  refreshPath?: string,
  onRefresh?: () => void,
) {
  const path = usePathname();
  const scope = useProjectScope();
  const client = useQueryClient();
  const [page, setPage] = useState(initialPage);
  const [phase, setPhase] = useState<Phase>("idle");
  const [message, setMessage] = useState("");
  const [errors, setErrors] = useState<Record<string, string>>({});
  const owner = useRef({});
  const mounted = useRef(true);
  const sending = useRef(false);
  const pending = useRef<{ path: string; body: URLSearchParams } | null>(null);
  const staleAction = useRef(false);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      scope.lock(owner.current, false);
    };
  }, [scope.lock]);
  useEffect(() => {
    if (!pending.current && !sending.current) {
      setPage(initialPage);
      setPhase((previous) => (previous === "settled" ? "idle" : previous));
    }
  }, [initialPage]);
  async function dispatch(action: { path: string; body: URLSearchParams }) {
    if (sending.current) return;
    sending.current = true;
    pending.current = action;
    scope.lock(owner.current, true);
    setPhase("sending");
    setMessage("");
    let result: CatalogMutationResult;
    try {
      result = await withMutationDeadline(action.path, (signal) =>
        submitAction<CatalogMutationResult>(
          action.path,
          action.body,
          signal,
          scope.key,
        ),
      );
    } catch {
      sending.current = false;
      if (mounted.current) {
        setPhase("uncertain");
        setMessage("Результат отправки неизвестен. Повторите ту же отправку.");
      }
      return;
    }
    sending.current = false;
    if (!mounted.current) return;
    if (result.ok || !result.retry_same_action) {
      pending.current = null;
      scope.lock(owner.current, false);
    }
    if (result.ok) {
      setPhase("settled");
      setErrors({});
      onSuccess(result);
      return;
    }
    const editable = !result.retry_same_action && definitiveFieldError(result);
    if (editable) staleAction.current = true;
    setPhase(
      result.retry_same_action ? "uncertain" : editable ? "idle" : "blocked",
    );
    setErrors(result.field_errors ?? result.page?.field_errors ?? {});
    setMessage(result.error?.message ?? "Не удалось сохранить изменения");
  }
  async function readFresh() {
    const fresh = await readPage<CatalogPage>(
      refreshPath ?? path,
      undefined,
      scope.key,
    );
    setPage(fresh);
    client.setQueryData(pageQueryKey(refreshPath ?? path), fresh);
    return fresh;
  }
  async function send(endpoint: string, key: string, values: URLSearchParams) {
    if (phase !== "idle" || scope.locked || pending.current || sending.current)
      return;
    let source = page;
    if (staleAction.current) {
      setPhase("sending");
      try {
        source = await readFresh();
        staleAction.current = false;
      } catch {
        if (!mounted.current) return;
        setPhase("idle");
        setMessage("Не удалось обновить форму. Ваши поля сохранены.");
        return;
      }
      if (!mounted.current) return;
      setPhase("idle");
    }
    const body = catalogBody(source, key, values);
    if (!body.get("action_id") || !body.get("csrf")) {
      setMessage("Действие сейчас недоступно. Обновите страницу.");
      return;
    }
    void dispatch({ path: endpoint, body });
  }
  async function refresh() {
    if (pending.current || sending.current) return;
    try {
      setPhase("sending");
      await readFresh();
      staleAction.current = false;
      void client.invalidateQueries({ queryKey: ["project-catalog"] });
      onRefresh?.();
      setPhase("idle");
      setMessage("");
      setErrors({});
    } catch {
      setPhase("blocked");
      setMessage("Не удалось обновить форму. Ваши поля сохранены.");
    }
  }
  return {
    page,
    phase,
    message,
    errors,
    send,
    refresh,
    busy: phase === "sending",
    locked: phase !== "idle" || scope.locked,
    retry: () => {
      if (pending.current) void dispatch(pending.current);
    },
    // Editing a field withdraws its stale rejection.
    clearError: (name: string) =>
      setErrors((current) => {
        if (!(name in current)) return current;
        const next = { ...current };
        delete next[name];
        return next;
      }),
  };
}
