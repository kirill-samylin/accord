"use client";
import { useState } from "react";
import type { MutationProps } from "../mutations/contracts";
import { actionBody, actionID } from "../mutations/contracts";
import { ActionNotice } from "../mutations/ActionNotice";
import { useAction } from "../mutations/useAction";
import { workflowControlReason } from "@/lib/automation";
import { messageError } from "@/lib/managed-agents";
export function WorkflowControls({ page, onSuccess }: MutationProps) {
  const action = useAction(page, onSuccess);
  const [expectedVersion, setExpectedVersion] = useState(
    page.workflow?.workflow.version,
  );
  const [text, setText] = useState("");
  const [error, setError] = useState("");
  const [refreshPolicy, setRefreshPolicy] = useState("");
  const detail = action.page.workflow;
  if (!detail) return null;
  const flow = detail.workflow;
  const stale =
    flow.id !== page.workflow?.workflow.id || flow.version !== expectedVersion;
  function send(kind: string) {
    if (!detail) return;
    const invalid = messageError(text, kind === "clarify" || kind === "replan");
    setError(invalid);
    if (invalid) return;
    const policy = (action.page.automation?.policies ?? []).find(
      (p) => `${p.id}:${p.version}` === refreshPolicy && p.state === "active",
    );
    void action.send(
      "/web/automation/command",
      actionBody(action.page, "automation:command", {
        workflow_id: flow.id,
        version: String(expectedVersion),
        kind,
        text,
        ...(kind === "resume" && policy
          ? {
              refresh_policy_id: policy.id,
              refresh_policy_version: String(policy.version),
            }
          : {}),
      }),
    );
  }
  return (
    <section className="automation-card screen-stack">
      <h2>Управление</h2>
      <p className="form-help">
        Пауза и отмена подтверждаются после остановки процессов. Уточнение
        текущему агенту доставляется на следующем ходе; изменение требований
        потребует пересмотра плана.
      </p>
      <fieldset
        disabled={
          action.locked ||
          !detail.can_control ||
          action.page.project?.can_write === false ||
          stale
        }
      >
        <label className="form-field">
          Уточнение или причина
          <textarea
            rows={3}
            value={text}
            onChange={(e) => setText(e.target.value)}
          />
        </label>
        {error ? <p className="field-error">{error}</p> : null}
        <label className="form-field">
          Явно обновить разрешение при продолжении
          <select
            value={refreshPolicy}
            onChange={(e) => setRefreshPolicy(e.target.value)}
          >
            <option value="">
              Использовать прежнее разрешение, если оно действует
            </option>
            {action.page.automation?.policies
              .filter(
                (p) =>
                  p.state === "active" &&
                  (!flow.host_id ||
                    (p.host_id === flow.host_id &&
                      p.workspace === flow.workspace)),
              )
              .map((p) => (
                <option key={p.id} value={`${p.id}:${p.version}`}>
                  {p.name} · версия {p.version}
                </option>
              ))}
          </select>
        </label>
        <div className="form-actions">
          {[
            { kind: "pause", title: "Пауза" },
            { kind: "resume", title: "Продолжить" },
            { kind: "clarify", title: "Передать уточнение" },
            { kind: "replan", title: "Пересмотреть план" },
            { kind: "cancel", title: "Отменить цикл" },
          ].map(({ kind, title }) => {
            const reason = workflowControlReason(
              kind,
              detail,
              action.page.project?.can_write !== false,
            );
            return (
              <button
                key={kind}
                className="button button-secondary"
                type="button"
                title={reason || title}
                disabled={
                  Boolean(reason) ||
                  !actionID(action.page, "automation:command")
                }
                onClick={() => send(kind)}
              >
                {title}
              </button>
            );
          })}
        </div>
      </fieldset>
      {stale ? (
        <div className="notice">
          <p>
            Состояние изменилось. Проверьте текущий результат перед следующей
            командой.
          </p>
          <button
            type="button"
            className="button button-secondary"
            disabled={action.busy || action.phase === "uncertain"}
            onClick={() => setExpectedVersion(flow.version)}
          >
            Использовать текущую версию {flow.version}
          </button>
        </div>
      ) : null}
      <ActionNotice
        message={action.message}
        uncertain={action.phase === "uncertain"}
        busy={action.busy}
        onRetry={() => void action.retry()}
      />
      {action.message && action.phase !== "uncertain" && !action.busy ? (
        <button
          className="button button-ghost"
          onClick={() =>
            void action.refresh(`/app/automation/workflows/${flow.id}`)
          }
        >
          Обновить форму управления
        </button>
      ) : null}
    </section>
  );
}
