"use client";

import { useEffect, useRef, useState } from "react";
import { readPage, submitAction } from "@/lib/api";
import type { MutationResult, PageData } from "@/lib/types";
import { snapshotPayload, type Payload, type PendingAction } from "./contracts";

import { applyActionResult, applyRefreshedPage } from "./actionState";
import { withMutationDeadline } from "./deadline";

type Phase = "idle" | "sending" | "uncertain" | "blocked" | "settled";

export function useAction(
  initialPage: PageData,
  onSuccess: (result: MutationResult) => void,
  mode: "action" | "login" = "action",
  onLockChange?: (locked: boolean) => void,
) {
  const [page, setPage] = useState(initialPage);
  const [phase, setPhase] = useState<Phase>("idle");
  const [message, setMessage] = useState("");
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
    };
  }, []);
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
    pending.current = action;
    setPhase("sending");
    setMessage("");
    let result: MutationResult;
    try {
      result = await withMutationDeadline(action.path, (signal) =>
        submitAction(action.path, action.body, signal),
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
    }
    setPage(state.page);
    setPhase(state.phase);
    if (result.ok) {
      setFieldErrors({});
      onSuccess(result);
      return;
    }
    setMessage(result.error?.message ?? "Не удалось выполнить действие");
    setFieldErrors(result.field_errors ?? result.page?.field_errors ?? {});
  }

  function send(path: string, body: Payload) {
    if (
      pending.current ||
      sending.current ||
      phase === "blocked" ||
      phase === "settled"
    )
      return;
    return dispatch({ path, body: snapshotPayload(body) });
  }
  function retry() {
    if (pending.current) return dispatch(pending.current);
  }
  async function refresh(path: string) {
    if (pending.current || sending.current) return;
    sending.current = true;
    setPhase("sending");
    try {
      const fresh = await readPage(path);
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
    fieldErrors,
    send,
    retry,
    refresh,
    locked: phase !== "idle",
    busy: phase === "sending",
    setMessage,
  };
}
