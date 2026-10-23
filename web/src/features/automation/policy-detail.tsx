"use client";
import { useState } from "react";
import { ProjectLink as Link } from "@/components/project-link";
import { automationRoles, stageLabels } from "@/lib/automation";
import { limitLabels } from "@/lib/automation-policy";
import { date } from "@/lib/format";
import { providerName } from "@/lib/managed-agents";
import { actionBody, type MutationProps } from "../mutations/contracts";
import { useAction } from "../mutations/useAction";
import { ActionNotice } from "../mutations/ActionNotice";
import { PolicyDialog } from "./policy-dialog";
export function PolicyDetail({ page, onSuccess }: MutationProps) {
  const [edit, setEdit] = useState(false),
    [reason, setReason] = useState("");
  const action = useAction(page, onSuccess),
    detail = action.page.automation_policy;
  if (!detail) return null;
  const policy = detail.policy;
  return (
    <section className="screen-stack">
      <div className="automation-card screen-stack">
        <h2>
          {policy.name} · версия {policy.version}
        </h2>
        <p>
          {policy.state === "active" ? "Действует" : "Отозвано"} · до{" "}
          {date(policy.expires_at)} · папка {policy.workspace}
        </p>
        <div className="automation-facts">
          {automationRoles.map((role) => {
            const binding = policy.role_bindings[role];
            const profile = page.profiles?.find(
              (p) => p.id === binding?.profile_id,
            );
            return (
              <span key={role}>
                {stageLabels[role]}: {profile?.name ?? binding?.profile_id} ·{" "}
                {providerName(binding?.provider ?? "")} · версия{" "}
                {binding?.profile_version}
              </span>
            );
          })}
        </div>
        <p className="mono break">
          Код разрешения: {policy.id} · версия {policy.version}
        </p>
        <details>
          <summary>Пределы и проверки</summary>
          <dl>
            {Object.entries(policy.limits).map(([key, value]) => (
              <div key={key}>
                <dt>{limitLabels[key as keyof typeof limitLabels]}</dt>
                <dd>{value}</dd>
              </div>
            ))}
          </dl>
          <ul>
            {policy.checks.map((check) => (
              <li key={check.preset_id}>
                {check.preset_id} · {check.kind} · версия {check.local_version}{" "}
                · {check.required ? "обязательная" : "необязательная"}
              </li>
            ))}
          </ul>
        </details>
        <p>
          Задания из Codex / Claude:{" "}
          {policy.client_intake
            ? "разрешены указанным участникам"
            : "выключены"}
          .
        </p>
        <p className="form-help">
          Изменение или отзыв останавливает дальнейшее автоматическое выполнение
          по старой версии. Продолжение требует явного обновления разрешения.
        </p>
        {page.can_manage_automation && policy.state === "active" ? (
          <>
            <button
              className="button button-secondary"
              disabled={action.locked}
              onClick={() => setEdit(true)}
            >
              Изменить разрешение
            </button>
            <fieldset disabled={action.locked}>
              <label className="form-field">
                Причина отзыва
                <input
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                />
              </label>
              <button
                className="button button-secondary"
                disabled={!reason.trim()}
                onClick={() =>
                  void action.send(
                    "/web/automation/policies/revoke",
                    actionBody(action.page, "automation:policy-revoke", {
                      policy_id: policy.id,
                      version: String(policy.version),
                      reason,
                    }),
                  )
                }
              >
                Отозвать разрешение
              </button>
            </fieldset>
          </>
        ) : null}
        <ActionNotice
          message={action.message}
          uncertain={action.phase === "uncertain"}
          busy={action.busy}
          onRetry={() => void action.retry()}
        />
      </div>
      <Link href="/app/automation" prefetch={false}>
        ← Все требования
      </Link>
      {edit ? (
        <PolicyDialog
          page={page}
          policy={policy}
          close={() => setEdit(false)}
          onSuccess={onSuccess}
        />
      ) : null}
    </section>
  );
}
