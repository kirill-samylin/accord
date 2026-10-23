"use client";
import { useEffect, useRef, useState } from "react";
import { APIError, readPage } from "@/lib/api";
import {
  LatestHostScopeRead,
  type HostScopeResult,
} from "@/lib/automation-host-scope";
import { useProjectScope } from "../projects/project-context";
export function usePolicyScope(
  hostID: string,
  workspace: string,
  authorityKey: string,
) {
  const scope = useProjectScope(),
    reader = useRef(new LatestHostScopeRead());
  const [attempt, setAttempt] = useState(0),
    [state, setState] = useState<{
      key: string;
      result?: HostScopeResult;
      error?: string;
    }>({ key: "" });
  const key = `${scope.key}:${hostID}:${workspace}:${authorityKey}`;
  const path = scope.app(
    `/app/automation/host-scope?${new URLSearchParams({ host_id: hostID, workspace })}`,
  );
  useEffect(() => {
    const current = reader.current;
    let active = true;
    if (!hostID || !workspace) {
      current.cancel();
      return;
    }
    setState({ key });
    void current
      .read((signal) => readPage(path, signal, scope.key), hostID, workspace)
      .then((result) => {
        if (active && result) setState({ key, result });
      })
      .catch((error) => {
        if (active)
          setState({
            key,
            error:
              error instanceof APIError && error.status === 404
                ? "Для этой папки не найдено локальное согласие. Выберите другую папку или попросите владельца настроить автоматизацию."
                : error instanceof APIError && error.status === 403
                  ? "Разрешение управлять этой папкой недоступно. Обратитесь к владельцу компьютера."
                  : error instanceof Error
                    ? error.message
                    : "Не удалось получить согласие для папки. Повторите загрузку.",
          });
      });
    return () => {
      active = false;
      current.cancel();
    };
  }, [key, hostID, workspace, path, scope.key, attempt]);
  const selected = state.key === key ? state : undefined;
  return {
    result: selected?.result,
    error: selected?.error,
    loading: Boolean(
      hostID && workspace && !selected?.result && !selected?.error,
    ),
    retry: () => setAttempt((value) => value + 1),
  };
}
