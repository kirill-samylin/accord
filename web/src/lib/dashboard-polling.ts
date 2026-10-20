import { APIError } from "./api.ts";
import type { DashboardData } from "./types.ts";
import { withReadDeadline } from "./read-deadline.ts";
export const POLL_BASE_MS = 15000;
export const POLL_CAP_MS = 120000;
export function retryDelay(
  failures: number,
  status?: number,
  header?: string | null,
  now = Date.now(),
): number {
  const backoff = Math.min(
    POLL_CAP_MS,
    POLL_BASE_MS * 2 ** Math.min(Math.max(failures, 0), 3),
  );
  if (![429, 503].includes(status ?? 0) || !header?.trim()) return backoff;
  const seconds = Number(header);
  const delay = Number.isFinite(seconds)
    ? seconds * 1000
    : Date.parse(header) - now;
  if (!Number.isFinite(delay) || delay <= 0) return backoff;
  return Math.max(
    backoff,
    Math.max(POLL_BASE_MS, Math.min(POLL_CAP_MS, delay)),
  );
}
export class DashboardReadError extends APIError {
  retryAfter: string | null;
  constructor(
    status: number,
    code: string,
    message: string,
    retryAfter: string | null,
  ) {
    super(status, { ok: false, error: { code, message } });
    this.retryAfter = retryAfter;
  }
}
const counts = [
  "tasks_total",
  "tasks_created",
  "tasks_ready",
  "tasks_in_progress",
  "tasks_done",
  "tasks_cancelled",
  "tasks_blocked",
  "open_epics",
  "active_agent_sessions",
  "active_human_sessions",
  "leased_tasks",
  "latest_event_cursor",
];
type ValidatedDashboard = Omit<DashboardData, "agents" | "activity"> & {
  agents: {
    sessions: DashboardData["agents"]["sessions"] | null;
    observed_at: string;
  };
  activity: {
    events: DashboardData["activity"]["events"] | null;
    observed_at: string;
  };
};
function record(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}
const text = (value: unknown): value is string => typeof value === "string";
const optionalText = (value: unknown) => value === undefined || text(value);
const validDate = (value: unknown) =>
  text(value) && Number.isFinite(Date.parse(value));
const sessionID = (value: unknown) =>
  text(value) && /^[a-f0-9]{64}$/.test(value);
const taskKey = (value: unknown) =>
  text(value) && /^[ET]-[1-9][0-9]*$/.test(value);
const agentRoles = new Set([
  "coordinator",
  "analyst",
  "architect",
  "developer",
  "reviewer",
  "tester",
]);
function validAgent(value: unknown): boolean {
  if (!record(value)) return false;
  const task = value.task;
  return (
    sessionID(value.session_id) &&
    text(value.role) &&
    agentRoles.has(value.role) &&
    text(value.owner) &&
    text(value.machine) &&
    text(value.run_id) &&
    value.status === "active" &&
    validDate(value.expires_at) &&
    (value.last_event_at == null || validDate(value.last_event_at)) &&
    (task == null || (record(task) && taskKey(task.key) && text(task.title)))
  );
}
function validEvent(value: unknown): boolean {
  if (!record(value)) return false;
  const data = value.data;
  return (
    Number.isSafeInteger(value.cursor) &&
    (value.cursor as number) >= 0 &&
    text(value.type) &&
    validDate(value.at) &&
    (value.task_key === undefined || taskKey(value.task_key)) &&
    (value.session_id === undefined || sessionID(value.session_id)) &&
    optionalText(value.reason) &&
    (data == null ||
      (record(data) &&
        ["owner", "role", "old_state", "new_state"].every((field) =>
          optionalText(data[field]),
        )))
  );
}
function validList(
  value: unknown,
  limit: number,
  entry: (value: unknown) => boolean,
) {
  return (
    value === null ||
    (Array.isArray(value) && value.length <= limit && value.every(entry))
  );
}
export function validDashboard(value: unknown): value is ValidatedDashboard {
  if (
    !record(value) ||
    !record(value.overview) ||
    !record(value.agents) ||
    !record(value.activity)
  )
    return false;
  const overview = value.overview;
  return (
    text(overview.project) &&
    validDate(overview.observed_at) &&
    counts.every(
      (key) =>
        typeof overview[key] === "number" &&
        Number.isSafeInteger(overview[key]) &&
        (overview[key] as number) >= 0,
    ) &&
    validDate(value.agents.observed_at) &&
    validDate(value.activity.observed_at) &&
    validList(value.agents.sessions, 6, validAgent) &&
    validList(value.activity.events, 20, validEvent)
  );
}
export async function readDashboard(
  parent: AbortSignal,
): Promise<DashboardData> {
  return withReadDeadline(parent, async (signal) => {
    const response = await fetch("/web/data/dashboard", {
      headers: { Accept: "application/json" },
      credentials: "same-origin",
      cache: "no-store",
      redirect: "error",
      signal,
    });
    const retryAfter = response.headers.get("Retry-After");
    if (response.status === 401)
      throw new DashboardReadError(401, "AUTH_REQUIRED", "Войдите снова", null);
    const payload = await response.json().catch((error) => {
      if (signal.aborted) throw error;
      return null;
    });
    if (!response.ok)
      throw new DashboardReadError(
        response.status,
        typeof payload?.code === "string" ? payload.code : "UNAVAILABLE",
        typeof payload?.error === "string"
          ? payload.error
          : "Не удалось обновить состояние",
        retryAfter,
      );
    if (!validDashboard(payload))
      throw new DashboardReadError(
        response.status,
        "INVALID_RESPONSE",
        "Сервер вернул неполное состояние",
        null,
      );
    signal.throwIfAborted();
    // Go encodes nil collections as null; consumers receive safe empty arrays.
    return {
      ...payload,
      agents: { ...payload.agents, sessions: payload.agents.sessions ?? [] },
      activity: { ...payload.activity, events: payload.activity.events ?? [] },
    };
  });
}
