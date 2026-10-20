import type { ProfileSummary } from "./agent-profiles.ts";
import type { PageData, Task } from "./types.ts";
export interface TaskAssignment {
  profile_id: string;
  profile_version: number;
  profile_name: string;
  role: string;
}
export interface CoordinationReason {
  code: string;
  message: string;
  task_key?: string;
}
export interface CoordinationCheck {
  task_key: string;
  role: string;
  state: string;
  outcome: string;
  target_commit: string;
  target_generation: number;
  current: boolean;
  successful: boolean;
  latest: boolean;
}
export interface TaskCoordination {
  task_key: string;
  task_version: number;
  observed_at: string;
  assignment?: TaskAssignment;
  routing_reasons: CoordinationReason[];
  blockers: CoordinationReason[];
  dependencies: {
    link: { from: string; to: string; type: string; condition?: string };
    satisfied: boolean;
  }[];
  checks: CoordinationCheck[];
  epic_closable?: boolean;
  closure_reasons: CoordinationReason[];
  truncated?: boolean;
}
export type CoordinationOperation = "assign" | "link" | "transition" | "checks";
export type CoordinationDraft = Record<
  | "version"
  | "profile"
  | "reason"
  | "from"
  | "type"
  | "condition"
  | "unlink"
  | "state"
  | "evidence_ref"
  | "role",
  string
>;
export const coordinationTitles: Record<CoordinationOperation, string> = {
  assign: "Назначение задачи",
  link: "Зависимость задачи",
  transition: "Изменить состояние",
  checks: "Создать проверку",
};
export function profileChoice(profile: { id: string; version: number }) {
  return `${profile.id}@${profile.version}`;
}
export function profilePair(choice: string): Record<string, string> {
  if (!choice) return {};
  const match = /^(ap_[a-f0-9]{32})@([1-9][0-9]*)$/.exec(choice);
  if (!match || !Number.isSafeInteger(Number(match[2])))
    throw new Error("Выберите профиль и его версию заново");
  return { profile_id: match[1], profile_version: match[2] };
}
export function currentProfileChoice(
  choice: string,
  role: string,
  profiles: ProfileSummary[] = [],
) {
  return (
    !choice ||
    profiles.some(
      (profile) =>
        profile.state === "active" &&
        profile.role === role &&
        profileChoice(profile) === choice,
    )
  );
}
export function canCoordinate(page: PageData) {
  return (
    page.can_coordinate === true &&
    !page.can_continue &&
    page.project?.can_write !== false &&
    (!page.task ||
      !page.coordination ||
      page.coordination.task_version === page.task.version)
  );
}
export function transitionChoices(
  task: Task,
): { value: string; title: string }[] {
  if (task.type === "epic")
    return task.state === "open"
      ? [
          { value: "done", title: "Закрыть эпик" },
          { value: "cancelled", title: "Отменить эпик" },
        ]
      : [{ value: "open", title: "Открыть эпик снова" }];
  if (task.lease || task.state === "in_progress") return [];
  if (task.state === "done")
    return [{ value: "ready", title: "Переоткрыть с доказательством дефекта" }];
  if (task.state === "cancelled")
    return [{ value: "ready", title: "Вернуть в очередь" }];
  if (task.state === "ready" || task.state === "created")
    return [{ value: "cancelled", title: "Отменить задачу" }];
  return [];
}
export function coordinationDraft(
  task: Task,
  role = "reviewer",
): CoordinationDraft {
  return {
    version: String(task.version),
    profile: task.assignment
      ? `${task.assignment.profile_id}@${task.assignment.profile_version}`
      : "",
    reason: "",
    from: "",
    type: "blocks",
    condition: "",
    unlink: "false",
    state: transitionChoices(task)[0]?.value ?? "",
    evidence_ref: "",
    role,
  };
}
export function coordinationFields(
  operation: CoordinationOperation,
  draft: CoordinationDraft,
  task: Task,
  sessionID: string,
): Record<string, string> {
  const generation =
    task.lease?.session_id === sessionID ? String(task.generation) : "0";
  switch (operation) {
    case "assign":
      return {
        version: draft.version,
        mode: draft.profile ? "profile" : "role_queue",
        ...profilePair(draft.profile),
        reason: draft.reason,
      };
    case "link":
      return {
        from: draft.from,
        type: draft.type,
        condition: draft.condition,
        unlink: draft.unlink,
        generation,
      };
    case "transition":
      return {
        state: draft.state,
        reason: draft.reason,
        evidence_ref: draft.evidence_ref,
        generation,
      };
    case "checks":
      return {
        target_version: draft.version,
        role: draft.role,
        ...profilePair(draft.profile),
      };
  }
}
export function latestPositiveReview(coordination?: TaskCoordination) {
  return coordination?.checks?.find(
    (check) =>
      check.role === "reviewer" &&
      check.successful &&
      check.current &&
      check.latest &&
      check.state === "done" &&
      check.outcome === "approve",
  );
}
