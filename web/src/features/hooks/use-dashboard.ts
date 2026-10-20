"use client";
import { useEffect, useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { PageData } from "@/lib/types";
import {
  DashboardReadError,
  POLL_BASE_MS,
  readDashboard,
  retryDelay,
} from "@/lib/dashboard-polling";
export function useDashboard(page: PageData, onSessionExpired: () => void) {
  const client = useQueryClient();
  const failures = useRef(0);
  const delay = useRef(POLL_BASE_MS);
  const stopped = useRef(false);
  const eligible = useRef(false);
  const [enabled, setEnabled] = useState(false);
  const live = useQuery({
    queryKey: ["dashboard"],
    queryFn: async ({ signal }) => {
      try {
        const data = await readDashboard(signal);
        failures.current = 0;
        delay.current = POLL_BASE_MS;
        return data;
      } catch (error) {
        if (signal.aborted || !eligible.current) throw error;
        if (
          error instanceof DashboardReadError &&
          [401, 409].includes(error.status)
        ) {
          stopped.current = true;
          if (error.status === 401)
            window.dispatchEvent(new Event("accord:unauthorized"));
          else onSessionExpired();
        } else {
          failures.current++;
          delay.current = retryDelay(
            failures.current,
            error instanceof DashboardReadError ? error.status : undefined,
            error instanceof DashboardReadError ? error.retryAfter : undefined,
          );
        }
        throw error;
      }
    },
    initialData: page.overview
      ? {
          overview: page.overview,
          agents: {
            sessions: page.agents ?? [],
            observed_at: page.agents_observed_at ?? "",
          },
          activity: {
            events: page.activity ?? [],
            observed_at: page.activity_observed_at ?? "",
          },
        }
      : undefined,
    enabled,
    refetchInterval: () =>
      enabled && !stopped.current ? delay.current : false,
    refetchIntervalInBackground: false,
    networkMode: "online",
    retry: false,
    gcTime: 0,
  });
  useEffect(() => {
    let pageActive = true;
    const update = () => {
      eligible.current =
        pageActive &&
        document.visibilityState === "visible" &&
        navigator.onLine &&
        !stopped.current;
      if (!eligible.current)
        void client.cancelQueries({ queryKey: ["dashboard"], exact: true });
      setEnabled(eligible.current);
    };
    const hide = () => {
      pageActive = false;
      update();
    };
    const show = () => {
      pageActive = true;
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
      void client.cancelQueries({ queryKey: ["dashboard"], exact: true });
      document.removeEventListener("visibilitychange", update);
      window.removeEventListener("offline", update);
      window.removeEventListener("online", update);
      window.removeEventListener("pagehide", hide);
      window.removeEventListener("pageshow", show);
    };
  }, [client]);
  return {
    ...live,
    refetch: () => {
      if (
        stopped.current &&
        live.error instanceof DashboardReadError &&
        live.error.status === 409
      ) {
        onSessionExpired();
        return;
      }
      if (eligible.current && !stopped.current)
        return live.refetch({ cancelRefetch: false });
    },
    paused: !enabled,
  };
}
