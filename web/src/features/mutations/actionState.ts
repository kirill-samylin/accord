import type { MutationResult, PageData } from "../../lib/types.ts";
import { stoppedBy } from "./contracts.ts";

export type ResultPhase = "idle" | "uncertain" | "blocked" | "settled";

export function applyActionResult(
  previous: PageData,
  result: MutationResult,
  mode: "action" | "login" = "action",
): { page: PageData; phase: ResultPhase; keepPending: boolean } {
  if (!result.ok && result.retry_same_action)
    return { page: previous, phase: "uncertain", keepPending: true };
  if (!result.ok && mode === "login") {
    const merged = { ...previous, ...result.page };
    const code = result.error?.code.toUpperCase() ?? "";
    // Wrong credentials do not revoke the valid preauth form. A rejected CSRF
    // or origin must bootstrap a new form explicitly before sending credentials.
    if (["FORBIDDEN", "CSRF", "ORIGIN"].includes(code))
      return {
        page: { ...merged, csrf: "" },
        phase: "blocked",
        keepPending: false,
      };
    if (
      [
        "UNAUTHORIZED",
        "UNAUTHENTICATED",
        "AUTH_REQUIRED",
        "TOKEN_EXPIRED",
      ].includes(code)
    ) {
      const csrf = result.page?.csrf || previous.csrf;
      return {
        page: { ...merged, csrf },
        phase: csrf ? "idle" : "blocked",
        keepPending: false,
      };
    }
  }
  if (!result.ok) {
    return {
      page: {
        ...previous,
        ...result.page,
        actions: result.actions ?? result.page?.actions ?? previous.actions,
      },
      phase: stoppedBy(result) ? "blocked" : "idle",
      keepPending: false,
    };
  }
  if (!result.session)
    return {
      page: previous,
      phase: result.redirect ? "settled" : "idle",
      keepPending: false,
    };
  const updatedTask = result.session.my_tasks?.find(
    (task) => task.key === previous.task?.key,
  );
  const ids = { ...previous.actions };
  if (result.next_action_id) {
    ids.heartbeat = result.next_action_id;
    if (updatedTask)
      ids[`heartbeat:${updatedTask.key}`] = result.next_action_id;
  }
  return {
    page: {
      ...previous,
      session: result.session,
      task: updatedTask ?? previous.task,
      actions: ids,
    },
    phase: result.redirect ? "settled" : "idle",
    keepPending: false,
  };
}

export function applyRefreshedPage(
  fresh: PageData,
  mode: "action" | "login" = "action",
): { page: PageData; phase: "idle" | "blocked" } {
  return {
    page: fresh,
    phase:
      mode === "login" && (fresh.name !== "login" || !fresh.csrf)
        ? "blocked"
        : "idle",
  };
}
