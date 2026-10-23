import type { ProfileSnapshot, ProfileSummary } from "./agent-profiles.ts";
export interface ManagedHost {
  automation?: import("./automation.ts").AutomationHostConsent;
  worker_capabilities?: Record<string, number>;
  id: string;
  owner_principal_id: string;
  name: string;
  version: number;
  state: string;
  controllers: string[];
  workspaces: string[];
  providers: string[];
  last_seen_at: string;
  worker_epoch: number;
  lease_expires_at: string;
  can_control: boolean;
}
export interface ManagedRun {
  id: string;
  host_id: string;
  version: number;
  provider: string;
  workspace: string;
  prompt: string;
  profile_snapshot?: ProfileSnapshot;
  task_key?: string;
  created_by: string;
  state: string;
  generation: number;
  provider_session_id?: string;
  created_at: string;
  updated_at: string;
  last_error?: string;
  output: string[];
  output_truncated: boolean;
}
export interface ManagedCommand {
  id: string;
  run_id: string;
  sequence: number;
  generation: number;
  kind: string;
  text?: string;
  created_by: string;
  created_at: string;
  status: string;
  dispatch_id?: string;
  accepted_at?: string;
  applied_at?: string;
  error?: string;
  forced?: boolean;
}
export interface ManagedPage {
  observed_at: string;
  actor_principal_id?: string;
  hosts: ManagedHost[];
  runs: ManagedRun[];
}
export interface ManagedRunPage {
  run: ManagedRun;
  commands: ManagedCommand[];
  commands_truncated?: boolean;
}
export const runLabels: Record<string, string> = {
  queued: "Ожидает запуска",
  starting: "Запускается",
  running: "Работает",
  stopping: "Останавливается",
  stopped: "Остановлен",
  completed: "Завершил шаг",
  failed: "Ошибка",
  needs_recovery: "Требует проверки на компьютере",
};
export const commandLabels: Record<string, string> = {
  start: "Запуск",
  stop: "Остановка",
  resume: "Продолжение",
  clarify: "Уточнение",
};
export const deliveryLabels: Record<string, string> = {
  queued: "В очереди",
  accepted: "Получено компьютером",
  applied: "Применено",
  failed: "Не выполнено",
  delivery_unknown: "Результат неизвестен",
  cancelled: "Отменено",
};
export function providerName(value: string) {
  return (
    ({ codex: "Codex", claude: "Claude Code" } as Record<string, string>)[
      value
    ] ?? value
  );
}
export function hostOnline(host: ManagedHost | undefined, observedAt?: string) {
  // Both timestamps belong to the server clock. Browser wall time is unrelated.
  const now = Date.parse(observedAt ?? "");
  if (!host || host.state !== "active") return false;
  const seen = Date.parse(host.last_seen_at);
  return (
    Number.isFinite(seen) &&
    now >= seen &&
    now - seen <= 30000 &&
    Date.parse(host.lease_expires_at) > now
  );
}
export function hostControlReason(
  host: ManagedHost | undefined,
  canWrite: boolean,
) {
  if (!canWrite) return "У вас доступ только для просмотра проекта.";
  if (!host) return "Компьютер недоступен в этом проекте.";
  if (!host.can_control)
    return "Владелец компьютера не разрешил вам управление.";
  if (host.state !== "active") return "Управление этим компьютером отключено.";
  return "";
}
export function commandReason(
  kind: string,
  run: ManagedRun,
  host: ManagedHost | undefined,
  canWrite: boolean,
  observedAt?: string,
) {
  const permission = hostControlReason(host, canWrite);
  if (permission) return permission;
  if (kind === "stop")
    return ["queued", "starting", "running"].includes(run.state)
      ? ""
      : "Остановка доступна для запущенной или ожидающей работы.";
  if (kind === "resume") {
    if (!["stopped", "completed"].includes(run.state))
      return "Продолжение доступно после подтверждённой остановки или завершения шага.";
    if (!run.provider_session_id)
      return "Нет подтверждённого диалога. Продолжение недоступно.";
    if (!hostOnline(host, observedAt))
      return "Подключите компьютер к Accord перед продолжением.";
    return "";
  }
  return ["running", "stopped", "completed"].includes(run.state)
    ? ""
    : "Уточнение доступно для работающего или остановленного агента.";
}
export function workspaceBusy(
  runs: ManagedRun[],
  host: string,
  workspace: string,
  except?: string,
) {
  return runs.some(
    (run) =>
      run.id !== except &&
      run.host_id === host &&
      run.workspace === workspace &&
      ["queued", "starting", "running", "stopping", "needs_recovery"].includes(
        run.state,
      ),
  );
}
export interface StartDraft {
  host_id: string;
  workspace: string;
  provider: string;
  profile: string;
  prompt: string;
  task_key: string;
}
export function profileOption(profile: ProfileSummary) {
  return `${profile.id}:${profile.version}`;
}
export function selectedProfile(value: string, profiles: ProfileSummary[]) {
  return profiles.find(
    (profile) => profile.state === "active" && profileOption(profile) === value,
  );
}
export function profileSupportsProvider(
  profile: ProfileSummary,
  provider: string,
) {
  return (
    profile.provider === "unconfigured" ||
    (profile.provider === "openai" && provider === "codex") ||
    (profile.provider === "anthropic" && provider === "claude")
  );
}
export function startErrors(
  draft: StartDraft,
  data: ManagedPage,
  profiles: ProfileSummary[],
  canWrite: boolean,
) {
  const errors: Record<string, string> = {};
  const host = data.hosts.find((entry) => entry.id === draft.host_id);
  const reason = hostControlReason(host, canWrite);
  if (reason) errors.host_id = reason;
  else if (!hostOnline(host, data.observed_at))
    errors.host_id = "По последним данным выбранный компьютер не на связи.";
  if (!host?.workspaces.includes(draft.workspace))
    errors.workspace = "Выберите разрешённую рабочую папку.";
  else if (workspaceBusy(data.runs, draft.host_id, draft.workspace))
    errors.workspace =
      "Папка занята. Дождитесь остановки или выберите другую разрешённую копию проекта.";
  if (!host?.providers.includes(draft.provider))
    errors.provider = "Выберите доступного агента.";
  const profile = selectedProfile(draft.profile, profiles);
  if (!profile) errors.profile = "Выберите действующую версию профиля.";
  else if (!profileSupportsProvider(profile, draft.provider))
    errors.profile = "Провайдер профиля не соответствует выбранному агенту.";
  if (
    !draft.prompt.trim() ||
    new TextEncoder().encode(draft.prompt).length > 16384
  )
    errors.prompt = "Опишите работу: до 16 КиБ текста.";
  return errors;
}
export function startFields(draft: StartDraft, profile: ProfileSummary) {
  return {
    host_id: draft.host_id,
    workspace: draft.workspace,
    provider: draft.provider,
    profile_id: profile.id,
    profile_version: String(profile.version),
    prompt: draft.prompt,
    ...(draft.task_key.trim() ? { task_key: draft.task_key.trim() } : {}),
  };
}
export function messageError(text: string, required = false) {
  return required && !text.trim()
    ? "Введите уточнение."
    : new TextEncoder().encode(text).length > 4096
      ? "Текст должен занимать не более 4 КиБ."
      : "";
}

export function managedPollInterval(
  path: string,
  locked: boolean,
  expired: boolean,
  errorStatus?: number,
  failures = 0,
): number | false {
  if (
    !/\/managed(?:\/|$)/.test(path) ||
    locked ||
    expired ||
    (errorStatus !== undefined && [401, 403, 404, 409].includes(errorStatus))
  )
    return false;
  return errorStatus === undefined
    ? 5000
    : Math.min(30000, 5000 * 2 ** Math.min(failures, 3));
}

// A rejected command may only adopt an actual newer detail of the same run.
export function managedConflictVersion(
  view: string,
  detail: ManagedRunPage | undefined,
  runID: string,
  submittedVersion: number | undefined,
) {
  return view === "managed-run" &&
    detail?.run.id === runID &&
    submittedVersion !== undefined &&
    detail.run.version > submittedVersion
    ? detail.run.version
    : undefined;
}
