"use client";

import { useEffect, useRef, useState } from "react";
import { readPage, submitAction } from "@/lib/api";
import type { MutationResult, PageData } from "@/lib/types";
import { snapshotPayload, type Payload, type PendingAction } from "./contracts";

import { applyActionResult, applyRefreshedPage } from "./actionState";
import { withMutationDeadline } from "./deadline";
import { useProjectScope } from "../projects/project-context";

type Phase = "idle" | "sending" | "uncertain" | "blocked" | "settled";

export function useAction(
  initialPage: PageData,
  onSuccess: (result: MutationResult) => void,
  mode: "action" | "login" = "action",
  onLockChange?: (locked: boolean) => void,
) {
  const scope = useProjectScope();
  const lockOwner = useRef({});
  const [page, setPage] = useState(initialPage);
  const [phase, setPhase] = useState<Phase>("idle");
  const [message, setMessage] = useState("");
  const [errorCode, setErrorCode] = useState("");
  const [fieldErrors, setFieldErrors] = useState(
    initialPage.field_errors ?? {},
  );
  const pending = useRef<PendingAction | null>(null);
  const sending = useRef(false);
  const mounted = useRef(true);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      scope.lock(lockOwner.current, false);
    };
  }, [scope.lock]);
  useEffect(() => {
    // Background read results may refresh capabilities, never an uncertain body.
    if (!pending.current && !sending.current) {
      setPage(initialPage);
      setPhase((previous) => (previous === "settled" ? "idle" : previous));
    }
  }, [initialPage]);

  async function dispatch(action: PendingAction) {
    if (sending.current) return;
    sending.current = true;
    onLockChange?.(true);
    scope.lock(lockOwner.current, true);
    pending.current = action;
    setPhase("sending");
    setMessage("");
    setErrorCode("");
    let result: MutationResult;
    try {
      result = await withMutationDeadline(action.path, (signal) =>
        submitAction(action.path, action.body, signal, scope.key),
      );
    } catch {
      sending.current = false;
      if (mounted.current) {
        setPhase("uncertain");
        setMessage(
          "Результат отправки неизвестен. Повторите эту же отправку, чтобы проверить результат.",
        );
      }
      return;
    }
    sending.current = false;
    if (!mounted.current) return;
    const state = applyActionResult(page, result, mode);
    if (!state.keepPending) {
      pending.current = null;
      onLockChange?.(false);
      scope.lock(lockOwner.current, false);
    }
    setPage(state.page);
    setPhase(state.phase);
    // Only a confirmed rejection may enable an explicit override. Unknown
    // outcomes keep the original action frozen for an exact retry.
    setErrorCode(
      !result.ok && !state.keepPending ? (result.error?.code ?? "") : "",
    );
    if (result.ok) {
      setFieldErrors({});
      onSuccess(result);
      return;
    }
    setMessage(friendlyMessage(result, mode));
    setFieldErrors(result.field_errors ?? result.page?.field_errors ?? {});
  }

  function send(path: string, body: Payload) {
    if (
      page.project?.can_write === false &&
      !["/web/logout", "/web/session/continue"].includes(path)
    )
      return;
    if (
      pending.current ||
      sending.current ||
      phase === "blocked" ||
      phase === "settled" ||
      scope.locked
    )
      return;
    return dispatch({ path: scope.web(path), body: snapshotPayload(body) });
  }
  function retry() {
    if (pending.current) return dispatch(pending.current);
  }
  async function refresh(path: string) {
    if (pending.current || sending.current) return;
    sending.current = true;
    setPhase("sending");
    try {
      const fresh = await readPage(scope.app(path));
      if (mounted.current) {
        const state = applyRefreshedPage(fresh, mode);
        setPage(state.page);
        setPhase(state.phase);
        setMessage(
          state.phase === "idle"
            ? ""
            : "Не удалось обновить форму входа. Повторите попытку.",
        );
      }
    } catch {
      if (mounted.current) {
        setPhase("blocked");
        setMessage("Не удалось обновить состояние. Черновик сохранён.");
      }
    } finally {
      sending.current = false;
    }
  }
  return {
    page,
    phase,
    message,
    errorCode,
    clearErrorCode: () => setErrorCode(""),
    fieldErrors,
    send,
    retry,
    refresh,
    locked: phase !== "idle",
    busy: phase === "sending",
    setMessage,
  };
}

const cyrillic = /[А-Яа-яЁё]/;
// Server messages are public but may be terse or English; users get a
// concrete next step instead of a generic failure.
export function friendlyMessage(
  result: MutationResult,
  mode: "action" | "login",
) {
  const code = result.error?.code ?? "";
  const raw = result.error?.message || result.page?.error || "";
  if (mode === "login") {
    if (code === "UNAVAILABLE" || result.retry_same_action)
      return raw || "Сервис временно недоступен. Повторите попытку.";
    if (code === "RATE_LIMITED" || /Слишком много/.test(raw))
      return "Слишком много попыток входа. Подождите минуту и попробуйте снова.";
    if (code === "FORBIDDEN" || /Обновите форму/.test(raw))
      return "Форма входа устарела. Нажмите «Обновить форму входа» и попробуйте снова.";
    return "Токен не подошёл. Проверьте, что вставили его целиком, без лишних символов, или попросите новый токен у владельца проекта.";
  }
  if (raw && cyrillic.test(raw)) return raw;
  if (code === "VALIDATION" || code === "INVALID_INPUT" || /invalid/i.test(raw))
    return "Проверьте заполненные поля: какое-то значение не подходит.";
  return raw || "Не удалось выполнить действие. Попробуйте ещё раз.";
}
