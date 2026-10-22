"use client";
import { useEffect, useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { TeamView } from "@/lib/types";
import { useProjectScope } from "../projects/project-context";
import { DashboardReadError, retryDelay } from "@/lib/dashboard-polling";
import { readTeam } from "@/lib/team";

// The team view is the "who is doing what" screen; it refreshes faster than
// the dashboard but keeps the same visibility, offline and backoff rules.
// Reads never renew a session or a lease.
export const TEAM_POLL_MS = 10000;

export function useTeam(initial: TeamView | undefined, onExpired: () => void) {
  const scope = useProjectScope();
  const client = useQueryClient();
  const failures = useRef(0);
  const delay = useRef(TEAM_POLL_MS);
  const stopped = useRef(false);
  const eligible = useRef(false);
  const [enabled, setEnabled] = useState(false);
  const key = ["team", scope.key];
  const live = useQuery({
    queryKey: key,
    queryFn: async ({ signal }) => {
      try {
        const data = await readTeam(signal, scope.id);
        failures.current = 0;
        delay.current = TEAM_POLL_MS;
        return data;
      } catch (error) {
        if (signal.aborted || !eligible.current) throw error;
        if (
          error instanceof DashboardReadError &&
          [401, 403, 404, 409].includes(error.status)
        ) {
          stopped.current = true;
          if (error.status === 401) {
            const event = new Event("accord:unauthorized");
            Object.assign(event, { projectScope: scope.key });
            window.dispatchEvent(event);
          } else onExpired();
        } else {
          failures.current++;
          delay.current = Math.max(
            TEAM_POLL_MS,
            retryDelay(
              failures.current,
              error instanceof DashboardReadError ? error.status : undefined,
              error instanceof DashboardReadError
                ? error.retryAfter
                : undefined,
            ),
          );
        }
        throw error;
      }
    },
    initialData: initial,
    enabled,
    refetchInterval: () =>
      enabled && !stopped.current ? delay.current : false,
    refetchIntervalInBackground: false,
    networkMode: "online",
    retry: false,
    gcTime: 0,
  });
  useEffect(() => {
    let active = true;
    const update = () => {
      eligible.current =
        active &&
        document.visibilityState === "visible" &&
        navigator.onLine &&
        !stopped.current;
      if (!eligible.current)
        void client.cancelQueries({ queryKey: key, exact: true });
      setEnabled(eligible.current);
    };
    const hide = () => {
      active = false;
      update();
    };
    const show = () => {
      active = true;
      update();
    };
    document.addEventListener("visibilitychange", update);
    window.addEventListener("offline", update);
    window.addEventListener("online", update);
    window.addEventListener("pagehide", hide);
    window.addEventListener("pageshow", show);
    update();
    return () => {
      eligible.current = false;
      void client.cancelQueries({ queryKey: key, exact: true });
      document.removeEventListener("visibilitychange", update);
      window.removeEventListener("offline", update);
      window.removeEventListener("online", update);
      window.removeEventListener("pagehide", hide);
      window.removeEventListener("pageshow", show);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [client, scope.key]);
  return {
    ...live,
    refetch: () => {
      if (eligible.current && !stopped.current)
        return live.refetch({ cancelRefetch: false });
    },
    paused: !enabled,
  };
}
