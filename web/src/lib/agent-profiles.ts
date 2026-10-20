export interface ProfileSnapshot {
  id: string;
  version: number;
  name: string;
  description: string;
  role: string;
  instructions: string;
  provider: string;
  model: string;
}
export interface AgentProfile extends ProfileSnapshot {
  state: "active" | "archived";
  created_at: string;
  updated_at: string;
  created_by: string;
  updated_by: string;
}
export type ProfileSummary = Omit<AgentProfile, "instructions">;
export interface ProfileRevision {
  version: number;
  state: "active" | "archived";
  updated_at: string;
  updated_by: string;
}
export const profileRoles = [
  "coordinator",
  "analyst",
  "architect",
  "developer",
  "reviewer",
  "tester",
];
export const providerLabels: Record<string, string> = {
  unconfigured: "Не выбран",
  openai: "OpenAI",
  anthropic: "Anthropic",
  local: "Локальный",
  custom: "Другой",
};
export type ProfileDraft = Record<
  | "name"
  | "description"
  | "role"
  | "instructions"
  | "provider"
  | "model"
  | "version",
  string
>;
export function profileDraft(profile?: AgentProfile): ProfileDraft {
  return {
    name: profile?.name ?? "",
    description: profile?.description ?? "",
    role: profile?.role ?? "developer",
    instructions: profile?.instructions ?? "",
    provider: profile?.provider ?? "unconfigured",
    model: profile?.model ?? "",
    version: profile ? String(profile.version) : "",
  };
}
export function profileFields(
  draft: ProfileDraft,
  editing: boolean,
): Record<string, string> {
  const { version, ...fields } = draft;
  return editing ? { ...fields, version } : fields;
}
export function profileDraftErrors(
  draft: ProfileDraft,
): Record<string, string> {
  const errors: Record<string, string> = {};
  if (!draft.name.trim() || Array.from(draft.name).length > 120)
    errors.name = "Введите название до 120 символов";
  const encoder = new TextEncoder();
  if (encoder.encode(draft.description).length > 4096)
    errors.description = "Описание должно занимать не более 4 КиБ";
  if (encoder.encode(draft.instructions).length > 16384)
    errors.instructions = "Инструкции должны занимать не более 16 КиБ";
  if (!profileRoles.includes(draft.role)) errors.role = "Выберите роль";
  if (!Object.hasOwn(providerLabels, draft.provider))
    errors.provider = "Выберите провайдера из списка";
  if (Array.from(draft.model).length > 200 || /[\p{Cc}]/u.test(draft.model))
    errors.model = "Укажите модель до 200 символов без управляющих знаков";
  return errors;
}
// A refreshed capability never silently rebases a draft onto another revision.
export function profileVersionChanged(
  draft: ProfileDraft,
  profile?: AgentProfile,
) {
  return Boolean(profile && draft.version !== String(profile.version));
}
