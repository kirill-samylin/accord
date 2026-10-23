"use client";
import { useState, type FormEvent } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { MutationResult, PageData } from "@/lib/types";
import type { AutomationPolicy, PolicySettings } from "@/lib/automation";
import { newPolicy, policyErrors } from "@/lib/automation-policy";
import { actionBody, actionID } from "../mutations/contracts";
import { useAction } from "../mutations/useAction";
import { ActionNotice, FieldError } from "../mutations/ActionNotice";
import { usePolicyScope } from "./use-policy-scope";
import { PolicyHost } from "./policy-host";
import { PolicyBindings } from "./policy-bindings";
import { PolicyLimits } from "./policy-limits";
export function PolicyDialog({
  page,
  policy,
  close,
  onSuccess,
}: {
  page: PageData;
  policy?: AutomationPolicy;
  close: () => void;
  onSuccess: (result: MutationResult) => void;
}) {
  const [draft, setDraft] = useState<PolicySettings>(() =>
    newPolicy(
      page.automation?.observed_at ?? "",
      page.automation?.actor_principal_id ?? "",
      policy,
    ),
  );
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [consented, setConsented] = useState(false);
  const action = useAction(page, (result) => {
    close();
    onSuccess(result);
  });
  const hosts = action.page.managed?.hosts ?? [],
    profiles = action.page.profiles ?? [];
  const summaryHost = hosts.find((h) => h.id === draft.host_id);
  const scopeRead = usePolicyScope(
    draft.host_id,
    draft.workspace,
    `${summaryHost?.version}:${summaryHost?.automation?.version}:${summaryHost?.can_control}:${summaryHost?.state}`,
  );
  const host = scopeRead.result?.host;
  const op = policy ? "automation:policy-update" : "automation:policy-create";
  const pending = action.phase === "sending" || action.phase === "uncertain";
  function submit(event: FormEvent) {
    event.preventDefault();
    const validation = policyErrors(
      draft,
      host ? [host] : [],
      profiles,
      scopeRead.result?.observedAt ?? action.page.automation?.observed_at ?? "",
    );
    if (!host || scopeRead.loading)
      validation.workspace =
        scopeRead.error ??
        "Дождитесь загрузки локального согласия для выбранной папки.";
    if (!consented)
      validation.consent = "Подтвердите разрешённый объём работы.";
    if (
      policy &&
      action.page.automation_policy?.policy.version !== policy.version
    )
      validation.version =
        "Версия разрешения изменилась. Закройте форму и откройте актуальную.";
    setErrors(validation);
    if (Object.keys(validation).length) return;
    void action.send(
      `/web/automation/policies/${policy ? "update" : "create"}`,
      actionBody(action.page, op, {
        settings: JSON.stringify(draft),
        ...(policy
          ? { policy_id: policy.id, version: String(policy.version) }
          : {}),
      }),
    );
  }
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !pending) close();
      }}
    >
      <DialogContent
        className="automation-dialog"
        aria-describedby="automation-policy-help"
      >
        <DialogHeader>
          <DialogTitle>
            {policy
              ? "Изменить разрешение команды"
              : "Разрешить автоматический цикл"}
          </DialogTitle>
        </DialogHeader>
        <p id="automation-policy-help" className="form-help">
          Сначала владелец подключает компьютер, агентов и тесты локально. Здесь
          вы выбираете только то, что он уже разрешил. Изменение разрешения
          остановит старые циклы до явного обновления доступа.
        </p>
        <form className="mutation-form" onSubmit={submit}>
          <fieldset
            disabled={action.locked || !action.page.can_manage_automation}
          >
            <label className="form-field">
              Название
              <input
                value={draft.name}
                onChange={(e) => setDraft({ ...draft, name: e.target.value })}
              />
              <FieldError name="name" errors={errors} />
            </label>
            <PolicyHost
              draft={draft}
              setDraft={setDraft}
              hosts={hosts}
              editing={Boolean(policy)}
              errors={errors}
              selectedHost={host}
            />
            {scopeRead.loading ? (
              <p className="form-help" role="status">
                Получаем актуальное согласие для папки…
              </p>
            ) : null}
            {scopeRead.error ? (
              <div className="notice" role="alert">
                <p>{scopeRead.error}</p>
                <button
                  type="button"
                  className="button button-secondary"
                  onClick={scopeRead.retry}
                >
                  Повторить загрузку согласия
                </button>
              </div>
            ) : null}
            <fieldset
              disabled={
                !host || draft.host_consent_version !== host.automation?.version
              }
            >
              <PolicyBindings
                draft={draft}
                setDraft={setDraft}
                host={host}
                profiles={profiles}
                errors={errors}
              />
              <PolicyLimits
                draft={draft}
                setDraft={setDraft}
                host={host}
                errors={errors}
              />
            </fieldset>
            <label className="form-field">
              Кому разрешён запуск (ID участников, через запятую)
              <input
                value={draft.intake_principals.join(", ")}
                onChange={(e) =>
                  setDraft({
                    ...draft,
                    intake_principals: e.target.value
                      .split(",")
                      .map((x) => x.trim())
                      .filter(Boolean),
                  })
                }
              />
            </label>
            <label className="form-field">
              Кому разрешено управление (ID участников, через запятую)
              <input
                value={draft.control_principals.join(", ")}
                onChange={(e) =>
                  setDraft({
                    ...draft,
                    control_principals: e.target.value
                      .split(",")
                      .map((x) => x.trim())
                      .filter(Boolean),
                  })
                }
              />
              <FieldError name="principals" errors={errors} />
            </label>
            <label className="automation-check">
              <input
                type="checkbox"
                checked={draft.client_intake}
                onChange={(e) =>
                  setDraft({ ...draft, client_intake: e.target.checked })
                }
              />
              <span>
                Разрешить задания из Codex / Claude указанным участникам
              </span>
            </label>
            <p className="form-help">
              У каждого участника также должно оставаться разрешение владельца
              управлять этим компьютером. Роль редактора проекта его не
              заменяет.
            </p>
            <label className="automation-check">
              <input
                type="checkbox"
                checked={consented}
                onChange={(e) => setConsented(e.target.checked)}
              />
              <span>
                Разрешаю этой команде выполнять требования в выбранной папке,
                создавать локальные коммиты и запускать выбранные проверки в
                этих пределах.
              </span>
            </label>
            <FieldError name="consent" errors={errors} />
            <FieldError name="version" errors={errors} />
            <button
              type="submit"
              className="button"
              disabled={
                !actionID(action.page, op) ||
                !host ||
                scopeRead.loading ||
                Boolean(scopeRead.error)
              }
            >
              Сохранить разрешение
            </button>
          </fieldset>
          <ActionNotice
            message={action.message}
            uncertain={action.phase === "uncertain"}
            busy={action.busy}
            onRetry={() => void action.retry()}
          />
        </form>
      </DialogContent>
    </Dialog>
  );
}
