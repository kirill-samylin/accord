import type { ProfileSnapshot } from "./agent-profiles.ts";
export const automationRoles = [
  "analyst",
  "developer",
  "reviewer",
  "tester",
] as const;
export type AutomationRole = (typeof automationRoles)[number];
export interface RoleBinding {
  provider: string;
  profile_id: string;
  profile_version: number;
}
export interface AutomationLimits {
  plan_tasks: number;
  fix_per_task: number;
  fix_total: number;
  native_invocations: number;
  continuation_per_step: number;
  step_seconds: number;
  workflow_seconds: number;
  grant_seconds: number;
}
export interface CheckPresetRef {
  preset_id: string;
  local_version: number;
  local_digest: string;
  kind: string;
  required: boolean;
  isolation: string;
}
export interface AutomationWorkspaceScope {
  version: number;
  expires_at: string;
  roles: Record<string, string[]>;
  checks: CheckPresetRef[];
  max_step_seconds: number;
  max_workflow_seconds: number;
  max_native_invocations: number;
}
export interface AutomationHostConsent {
  scopes_truncated?: boolean;
  scopes: Record<string, AutomationWorkspaceScope>;
  enabled: boolean;
  version: number;
  expires_at: string;
  workspaces: string[];
  roles: string[];
  providers: string[];
  checks: CheckPresetRef[];
  max_step_seconds: number;
}
export interface PolicySettings {
  name: string;
  host_id: string;
  host_consent_version: number;
  workspace: string;
  role_bindings: Record<string, RoleBinding>;
  intake_principals: string[];
  control_principals: string[];
  client_intake: boolean;
  checks: CheckPresetRef[];
  git_policy: string;
  limits: AutomationLimits;
  expires_at: string;
}
export interface AutomationPolicy extends PolicySettings {
  id: string;
  version: number;
  state: string;
  issuer_principal_id: string;
  issuer_epoch: number;
  host_owner_epoch: number;
  created_at: string;
  updated_at: string;
}
export interface WorkflowGrant {
  policy_id: string;
  policy_version: number;
  role_bindings: Record<string, RoleBinding>;
  profiles: Record<string, ProfileSnapshot>;
  checks: CheckPresetRef[];
  limits: AutomationLimits;
  approved_at: string;
  expires_at: string;
}
export interface Workflow {
  id: string;
  version: number;
  control_epoch: number;
  state: string;
  reason_code?: string;
  intake_text: string;
  intake_truncated?: boolean;
  intake_hash: string;
  grant?: WorkflowGrant;
  host_id?: string;
  workspace?: string;
  epic_key?: string;
  plan_revision: number;
  current_step_id?: string;
  counters: {
    invocations: number;
    started: number;
    fix_total: number;
    fix_per_task: Record<string, number>;
    tasks: number;
  };
  deadline_at: string;
  created_at: string;
  updated_at: string;
  messages?: string[];
}
export interface WorkflowStep {
  id: string;
  workflow_id: string;
  sequence: number;
  kind: string;
  status: string;
  attempt_no: number;
  task_key: string;
  task_version: number;
  task_generation: number;
  profile_snapshot: ProfileSnapshot;
  provider: string;
  target_sha?: string;
  managed_run_id: string;
  managed_generation: number;
  native_session_id?: string;
  started_at: string;
  deadline_at: string;
  ended_at?: string;
  evidence_refs?: string[];
}
export interface WorkflowQuestion {
  id: string;
  workflow_id: string;
  step_id: string;
  version: number;
  human_task_key: string;
  designated_principals: string[];
  state: string;
  question: string;
  options?: string[];
  answer_ref?: string;
  created_at: string;
}
export interface WorkflowPlanTask {
  client_key: string;
  title: string;
  description: string;
  areas: string[];
  acceptance: string[];
  prerequisites: string[];
  required_check_ids: string[];
}
export interface WorkflowPlan {
  workflow_id: string;
  revision: number;
  hash: string;
  tasks: WorkflowPlanTask[];
  task_keys: Record<string, string>;
  status: string;
  created_at: string;
}
export interface WorkflowAttestation {
  id: string;
  workflow_id: string;
  step_id: string;
  run_id: string;
  generation: number;
  kind: string;
  preset_id?: string;
  status: string;
  created_at: string;
  observations: {
    actual_head?: string;
    target_sha?: string;
    source_clean?: boolean;
    check_exit_code?: number;
    check_aborted?: boolean;
    checks_passed: number;
    checks_failed: number;
    checks_skipped: number;
    report_sha256?: string;
    artifact_refs?: string[];
    outcome_bytes_truncated: boolean;
  };
}
export interface WorkflowListPage {
  observed_at: string;
  actor_principal_id: string;
  policies: AutomationPolicy[];
  workflows: Workflow[];
  next_cursor?: string;
}
export interface WorkflowPage {
  workflow: Workflow;
  steps: WorkflowStep[];
  plans: WorkflowPlan[];
  questions: WorkflowQuestion[];
  attestations: WorkflowAttestation[];
  can_control: boolean;
  next_cursor?: string;
}
export interface AutomationPolicyPage {
  next_cursor?: string;
  policy: AutomationPolicy;
  revisions: AutomationPolicy[];
}
export const workflowLabels: Record<string, string> = {
  pending: "Ожидает выполнения",
  ready: "Готово к выполнению",
  accepted: "Получено компьютером",
  applied: "Применено",
  failed: "Ошибка",
  passed: "Проверка пройдена",
  approved: "Одобрено",
  changes_requested: "Нужны исправления",
  interrupted: "Прервано",
  queued_check: "Проверка в очереди",
  draft: "Черновик",
  awaiting_authorization: "Требуется разрешение",
  queued: "В очереди",
  running: "Выполняется",
  pausing: "Останавливаем для паузы",
  paused: "Приостановлено",
  awaiting_human: "Нужен ваш ответ",
  stopping: "Останавливаем",
  cancelled: "Отменено",
  completed: "Завершено",
  blocked: "Заблокировано",
  needs_recovery: "Нужна проверка на компьютере",
  evidence_pending: "Проверяем подтверждения",
};
export const stageLabels: Record<string, string> = {
  analysis: "Анализ",
  analyst: "Аналитик",
  develop: "Разработка",
  developer: "Разработчик",
  review: "Независимое ревью",
  reviewer: "Ревьюер",
  test: "Тестирование",
  tester: "Тестировщик",
  fix: "Исправление",
  test_design: "Подготовка тестов",
  materialize: "Создание задач",
  finish: "Результат",
};
export function requirementError(text: string) {
  return !text.trim()
    ? "Опишите требование."
    : new TextEncoder().encode(text).length > 16384
      ? "До 16 КиБ текста."
      : "";
}
export function workflowControlReason(
  kind: string,
  detail: WorkflowPage,
  canWrite: boolean,
) {
  if (!canWrite || !detail.can_control)
    return "У вас нет действующего разрешения на управление этим циклом.";
  const state = detail.workflow.state;
  if (["completed", "cancelled"].includes(state)) return "Этот цикл завершён.";
  if (kind === "pause" && !["queued", "running"].includes(state))
    return "Пауза доступна для ожидающей или выполняемой работы.";
  if (
    kind === "resume" &&
    ![
      "draft",
      "awaiting_authorization",
      "paused",
      "awaiting_human",
      "blocked",
      "needs_recovery",
    ].includes(state)
  )
    return "Дождитесь подтверждённой остановки.";
  if (
    kind === "clarify" &&
    ["pausing", "stopping", "needs_recovery"].includes(state)
  )
    return "Сначала подтвердите остановку или восстановление.";
  return "";
}
export function automationPollInterval(
  path: string,
  locked: boolean,
  expired: boolean,
  status?: number,
  failures = 0,
): number | false {
  if (
    !/\/automation(?:\/|$)/.test(path) ||
    locked ||
    expired ||
    (status !== undefined && [401, 403, 404, 409].includes(status))
  )
    return false;
  return status === undefined
    ? 5000
    : Math.min(30000, 5000 * 2 ** Math.min(failures, 3));
}
