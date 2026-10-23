import type { ProfileSummary } from "./agent-profiles.ts";
import {
  automationRoles,
  type AutomationLimits,
  type AutomationPolicy,
  type PolicySettings,
} from "./automation.ts";
import {
  hostOnline,
  profileSupportsProvider,
  type ManagedHost,
} from "./managed-agents.ts";
export const defaultAutomationLimits: AutomationLimits = {
  plan_tasks: 8,
  fix_per_task: 2,
  fix_total: 2,
  native_invocations: 40,
  continuation_per_step: 1,
  step_seconds: 1800,
  workflow_seconds: 28800,
  grant_seconds: 86400,
};
export const limitLabels: Record<keyof AutomationLimits, string> = {
  plan_tasks: "Задач в плане",
  fix_per_task: "Исправлений одной задачи",
  fix_total: "Исправлений всего",
  native_invocations: "Запусков агентов",
  continuation_per_step: "Продолжений шага",
  step_seconds: "Лимит шага, секунд",
  workflow_seconds: "Лимит цикла, секунд",
  grant_seconds: "Срок разрешения, секунд",
};
export function newPolicy(
  observedAt: string,
  principal: string,
  policy?: AutomationPolicy,
): PolicySettings {
  if (policy) {
    const {
      name,
      host_id,
      host_consent_version,
      workspace,
      role_bindings,
      intake_principals,
      control_principals,
      client_intake,
      checks,
      git_policy,
      limits,
      expires_at,
    } = policy;
    return {
      name,
      host_id,
      host_consent_version,
      workspace,
      role_bindings,
      intake_principals,
      control_principals,
      client_intake,
      checks,
      git_policy,
      limits,
      expires_at,
    };
  }
  const now = Date.parse(observedAt);
  return {
    name: "Команда проекта",
    host_id: "",
    host_consent_version: 0,
    workspace: "",
    role_bindings: {},
    intake_principals: principal ? [principal] : [],
    control_principals: principal ? [principal] : [],
    client_intake: false,
    checks: [],
    git_policy: "local_commits",
    limits: { ...defaultAutomationLimits },
    expires_at: Number.isFinite(now)
      ? new Date(now + 86400000).toISOString()
      : "",
  };
}
export function policyErrors(
  draft: PolicySettings,
  hosts: ManagedHost[],
  profiles: ProfileSummary[],
  observedAt: string,
) {
  const errors: Record<string, string> = {};
  const host = hosts.find((entry) => entry.id === draft.host_id);
  const consent = host?.automation;
  const scope = consent?.scopes[draft.workspace];
  if (!draft.name.trim()) errors.name = "Введите название разрешения.";
  if (!host?.can_control || !hostOnline(host, observedAt))
    errors.host_id = "Нужен подключённый компьютер с правом управления.";
  if (
    host?.worker_capabilities?.automation !== 1 ||
    !consent?.enabled ||
    consent.version !== draft.host_consent_version ||
    Date.parse(consent.expires_at) <= Date.parse(observedAt)
  )
    errors.host_id =
      "Владелец должен включить локальное согласие на автоматизацию.";
  if (!scope || consent?.scopes_truncated)
    errors.workspace = "Выберите папку из локального согласия.";
  for (const role of automationRoles) {
    const binding = draft.role_bindings[role];
    const profile = profiles.find(
      (entry) =>
        entry.id === binding?.profile_id &&
        entry.version === binding.profile_version &&
        entry.role === role &&
        entry.state === "active",
    );
    if (
      !binding ||
      !profile ||
      !scope?.roles[role]?.includes(binding.provider) ||
      !profileSupportsProvider(profile, binding.provider)
    )
      errors[role] =
        "Выберите действующий профиль этой роли и разрешённого агента.";
  }
  if (!draft.checks.some((entry) => entry.required && entry.kind === "test"))
    errors.checks =
      "Нужна хотя бы одна обязательная проверка с реальными тестами.";
  if (
    draft.checks.some(
      (entry) =>
        !scope?.checks.some(
          (allowed) =>
            allowed.preset_id === entry.preset_id &&
            allowed.local_version === entry.local_version &&
            allowed.local_digest === entry.local_digest,
        ),
    )
  )
    errors.checks = "Проверки изменились. Выберите их заново.";
  if (
    scope?.checks.some(
      (entry) =>
        entry.required &&
        !draft.checks.some(
          (selected) => selected.preset_id === entry.preset_id,
        ),
    )
  )
    errors.checks = "Нельзя исключить обязательные проверки владельца.";
  const expiry = Date.parse(draft.expires_at),
    now = Date.parse(observedAt);
  if (
    !Number.isFinite(expiry) ||
    expiry <= now ||
    expiry > now + 604800000 ||
    expiry > Date.parse(consent?.expires_at ?? "") ||
    expiry > Date.parse(scope?.expires_at ?? "")
  )
    errors.expires_at =
      "Срок должен быть в будущем, до 7 дней и не позднее локального согласия.";
  if (
    draft.limits.workflow_seconds < draft.limits.step_seconds ||
    draft.limits.grant_seconds < draft.limits.workflow_seconds
  )
    errors.limits =
      "Лимит цикла должен покрывать шаг, срок разрешения — весь цикл.";
  if (
    scope &&
    (draft.limits.step_seconds > scope.max_step_seconds ||
      draft.limits.workflow_seconds > scope.max_workflow_seconds ||
      draft.limits.native_invocations > scope.max_native_invocations)
  )
    errors.limits = "Лимиты превышают локальное согласие владельца.";
  const maxima: AutomationLimits = {
    plan_tasks: 16,
    fix_per_task: 5,
    fix_total: 10,
    native_invocations: 128,
    continuation_per_step: 1,
    step_seconds: 7200,
    workflow_seconds: 86400,
    grant_seconds: 604800,
  };
  for (const key of Object.keys(maxima) as (keyof AutomationLimits)[]) {
    const value = draft.limits[key];
    if (
      !Number.isSafeInteger(value) ||
      value <
        (["fix_per_task", "fix_total", "continuation_per_step"].includes(key)
          ? 0
          : 1) ||
      value > maxima[key]
    )
      errors.limits =
        "Проверьте пределы работы: нужны целые значения в допустимых границах.";
  }
  if (!draft.intake_principals.length || !draft.control_principals.length)
    errors.principals =
      "Укажите участников, которым разрешены запуск и управление.";
  return errors;
}
