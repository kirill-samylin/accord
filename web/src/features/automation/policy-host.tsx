"use client";
import { Select } from "@/components/ui/select";
import type { PolicySettings } from "@/lib/automation";
import type { ManagedHost } from "@/lib/managed-agents";
import { FieldError } from "../mutations/ActionNotice";
function earlierExpiry(current: string, limit?: string) {
  return limit && Date.parse(limit) < Date.parse(current) ? limit : current;
}
export function PolicyHost({
  draft,
  setDraft,
  hosts,
  editing,
  errors,
  selectedHost,
}: {
  draft: PolicySettings;
  setDraft: (value: PolicySettings) => void;
  hosts: ManagedHost[];
  editing: boolean;
  selectedHost?: ManagedHost;
  errors: Record<string, string>;
}) {
  const host = hosts.find((h) => h.id === draft.host_id);
  const fullHost =
    selectedHost?.id === draft.host_id ? selectedHost : undefined;
  function workspace(value: string, refreshConsent = false) {
    const scope = refreshConsent
      ? fullHost?.automation?.scopes[value]
      : undefined;
    setDraft({
      ...draft,
      workspace: value,
      host_consent_version: refreshConsent
        ? (fullHost?.automation?.version ?? 0)
        : 0,
      role_bindings: {},
      checks: scope?.checks.filter((p) => p.required) ?? [],
      limits: {
        ...draft.limits,
        step_seconds: Math.min(
          draft.limits.step_seconds,
          scope?.max_step_seconds ?? 1800,
        ),
        workflow_seconds: Math.min(
          draft.limits.workflow_seconds,
          scope?.max_workflow_seconds ?? 28800,
        ),
        native_invocations: Math.min(
          draft.limits.native_invocations,
          scope?.max_native_invocations ?? 40,
        ),
      },
      expires_at: earlierExpiry(
        earlierExpiry(draft.expires_at, host?.automation?.expires_at),
        scope?.expires_at,
      ),
    });
  }
  return (
    <>
      <div className="form-grid">
        <label className="form-field">
          Компьютер
          <Select
            disabled={editing}
            value={draft.host_id}
            onChange={(e) => {
              const h = hosts.find((entry) => entry.id === e.target.value);
              setDraft({
                ...draft,
                host_id: e.target.value,
                host_consent_version: 0,
                workspace: "",
                role_bindings: {},
                checks: [],
                expires_at: earlierExpiry(
                  draft.expires_at,
                  h?.automation?.expires_at,
                ),
              });
            }}
          >
            <option value="">Выберите компьютер</option>
            {hosts.map((h) => (
              <option
                value={h.id}
                key={h.id}
                disabled={
                  !h.can_control ||
                  !h.automation?.enabled ||
                  h.worker_capabilities?.automation !== 1 ||
                  h.state !== "active"
                }
              >
                {h.name}
                {!h.automation?.enabled
                  ? " · автоматизация не разрешена локально"
                  : ""}
              </option>
            ))}
          </Select>
          <FieldError name="host_id" errors={errors} />
        </label>
        <label className="form-field">
          Рабочая папка
          <Select
            value={draft.workspace}
            onChange={(e) => workspace(e.target.value)}
          >
            <option value="">Выберите папку</option>
            {(host?.workspaces ?? []).map((alias) => (
              <option key={alias}>{alias}</option>
            ))}
          </Select>
          <FieldError name="workspace" errors={errors} />
        </label>
      </div>
      {fullHost?.automation &&
      fullHost.automation.version !== draft.host_consent_version ? (
        <div className="notice">
          <p>
            Проверьте локальное согласие выбранной папки. После принятия
            выберите роли и проверки заново.
          </p>
          <button
            type="button"
            className="button button-secondary"
            onClick={() => workspace(draft.workspace, true)}
          >
            Принять текущее согласие · версия {fullHost.automation.version}
          </button>
        </div>
      ) : null}
    </>
  );
}
