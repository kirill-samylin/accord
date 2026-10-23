"use client";
import type { ProfileSummary } from "@/lib/agent-profiles";
import {
  automationRoles,
  stageLabels,
  type PolicySettings,
} from "@/lib/automation";
import {
  profileOption,
  profileSupportsProvider,
  providerName,
  type ManagedHost,
} from "@/lib/managed-agents";
import { FieldError } from "../mutations/ActionNotice";
export function PolicyBindings({
  draft,
  setDraft,
  host,
  profiles,
  errors,
}: {
  draft: PolicySettings;
  setDraft: (value: PolicySettings) => void;
  host?: ManagedHost;
  profiles: ProfileSummary[];
  errors: Record<string, string>;
}) {
  return (
    <section className="screen-stack">
      <h3>Кто выполняет каждый этап</h3>
      {automationRoles.map((role) => {
        const binding = draft.role_bindings[role];
        const eligible = profiles.filter(
          (p) =>
            p.role === role &&
            p.state === "active" &&
            (!binding?.provider ||
              profileSupportsProvider(p, binding.provider)),
        );
        return (
          <div key={role} className="automation-role-binding">
            <strong>{stageLabels[role]}</strong>
            <label className="form-field">
              Агент
              <select
                value={binding?.provider ?? ""}
                onChange={(e) =>
                  setDraft({
                    ...draft,
                    role_bindings: {
                      ...draft.role_bindings,
                      [role]: {
                        provider: e.target.value,
                        profile_id: "",
                        profile_version: 0,
                      },
                    },
                  })
                }
              >
                <option value="">Выберите агента</option>
                {host?.automation?.scopes[draft.workspace]?.roles[role]?.map(
                  (provider) => (
                    <option value={provider} key={provider}>
                      {providerName(provider)}
                    </option>
                  ),
                )}
              </select>
            </label>
            <label className="form-field">
              Профиль и версия
              <select
                value={
                  binding?.profile_id
                    ? `${binding.profile_id}:${binding.profile_version}`
                    : ""
                }
                onChange={(e) => {
                  const p = eligible.find(
                    (profile) => profileOption(profile) === e.target.value,
                  );
                  if (p)
                    setDraft({
                      ...draft,
                      role_bindings: {
                        ...draft.role_bindings,
                        [role]: {
                          provider: binding?.provider ?? "",
                          profile_id: p.id,
                          profile_version: p.version,
                        },
                      },
                    });
                }}
              >
                <option value="">Выберите профиль</option>
                {eligible.map((profile) => (
                  <option key={profile.id} value={profileOption(profile)}>
                    {profile.name} · версия {profile.version}
                  </option>
                ))}
                {binding?.profile_id &&
                !eligible.some(
                  (p) =>
                    profileOption(p) ===
                    `${binding.profile_id}:${binding.profile_version}`,
                ) ? (
                  <option
                    value={`${binding.profile_id}:${binding.profile_version}`}
                    disabled
                  >
                    Версия изменилась — выберите заново
                  </option>
                ) : null}
              </select>
              <FieldError name={role} errors={errors} />
            </label>
          </div>
        );
      })}
    </section>
  );
}
