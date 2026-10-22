import { withReadDeadline } from "./read-deadline.ts";
import { projectID } from "./projects.ts";
import { DashboardReadError } from "./dashboard-polling.ts";
import type { SimilarPage, TeamView } from "./types.ts";

function record(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

export function validTeam(value: unknown): value is TeamView {
  return (
    record(value) &&
    typeof value.observed_at === "string" &&
    Number.isFinite(Date.parse(value.observed_at)) &&
    Array.isArray(value.members) &&
    value.members.every(
      (m) =>
        record(m) &&
        typeof m.owner === "string" &&
        Array.isArray(m.agents) &&
        Array.isArray(m.machines),
    ) &&
    Array.isArray(value.unclaimed) &&
    Array.isArray(value.duplicates) &&
    typeof value.duplicates_total === "number" &&
    typeof value.duplicate_mode === "string"
  );
}

export function validSimilar(value: unknown): value is SimilarPage {
  return (
    record(value) &&
    typeof value.mode === "string" &&
    Array.isArray(value.similar) &&
    value.similar.every(
      (s) =>
        record(s) &&
        typeof s.key === "string" &&
        /^[ET]-[1-9][0-9]*$/.test(s.key) &&
        typeof s.title === "string" &&
        typeof s.score === "number",
    )
  );
}

async function readData<T>(
  parent: AbortSignal,
  project: string | undefined,
  path: string,
  valid: (value: unknown) => value is T,
): Promise<T> {
  if (project && !projectID.test(project))
    throw new Error("Недопустимый проект");
  return withReadDeadline(parent, async (signal) => {
    const response = await fetch(
      project ? `/web/p/${project}/data/${path}` : `/web/data/${path}`,
      {
        headers: { Accept: "application/json" },
        credentials: "same-origin",
        cache: "no-store",
        redirect: "error",
        signal,
      },
    );
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
        response.headers.get("Retry-After"),
      );
    if (!valid(payload))
      throw new DashboardReadError(
        response.status,
        "INVALID_RESPONSE",
        "Сервер вернул неполное состояние",
        null,
      );
    signal.throwIfAborted();
    return payload;
  });
}

export function readTeam(signal: AbortSignal, project?: string) {
  return readData(signal, project, "team", validTeam);
}

export function similarQuery(fields: {
  title: string;
  description?: string;
  type?: string;
  role?: string;
  exclude_key?: string;
}) {
  const query = new URLSearchParams();
  query.set("title", fields.title.trim().slice(0, 200));
  const description = (fields.description ?? "").trim().slice(0, 1500);
  if (description) query.set("description", description);
  if (fields.type) query.set("type", fields.type);
  if (fields.role && fields.type !== "epic") query.set("role", fields.role);
  if (fields.exclude_key) query.set("exclude_key", fields.exclude_key);
  return query.toString();
}

export function readSimilar(
  signal: AbortSignal,
  project: string | undefined,
  query: string,
) {
  return readData(signal, project, `similar?${query}`, validSimilar);
}
