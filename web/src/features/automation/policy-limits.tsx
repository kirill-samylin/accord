"use client";
import { limitLabels } from "@/lib/automation-policy";
import type { AutomationLimits, PolicySettings } from "@/lib/automation";
import type { ManagedHost } from "@/lib/managed-agents";
import { FieldError } from "../mutations/ActionNotice";
export function PolicyLimits({
  draft,
  setDraft,
  host,
  errors,
}: {
  draft: PolicySettings;
  setDraft: (value: PolicySettings) => void;
  host?: ManagedHost;
  errors: Record<string, string>;
}) {
  return (
    <>
      <section className="screen-stack">
        <h3>Проверки на компьютере</h3>
        {host?.automation?.scopes[draft.workspace]?.checks.length ? (
          host.automation.scopes[draft.workspace].checks.map((preset) => (
            <label className="automation-check" key={preset.preset_id}>
              <input
                type="checkbox"
                checked={draft.checks.some(
                  (p) => p.preset_id === preset.preset_id,
                )}
                onChange={(e) =>
                  setDraft({
                    ...draft,
                    checks: e.target.checked
                      ? [...draft.checks, preset]
                      : draft.checks.filter(
                          (p) => p.preset_id !== preset.preset_id,
                        ),
                  })
                }
              />
              <span>
                {preset.preset_id} · {preset.kind} · версия{" "}
                {preset.local_version} ·{" "}
                {preset.required ? "обязательная" : "дополнительная"}
                <small className="muted">
                  {preset.isolation === "trusted_owner_process"
                    ? "Владелец разрешил исполнение кода проекта своим процессом"
                    : "Изоляция настроена владельцем"}
                </small>
              </span>
            </label>
          ))
        ) : (
          <p className="form-help">
            Владелец компьютера должен добавить и разрешить пресеты реальных
            тестов. Сайт не выдаёт это локальное разрешение.
          </p>
        )}
        <FieldError name="checks" errors={errors} />
      </section>
      <details className="automation-card">
        <summary>Пределы автоматической работы</summary>
        <div className="form-grid">
          {(Object.keys(limitLabels) as (keyof AutomationLimits)[]).map(
            (key) => (
              <label className="form-field" key={key}>
                {limitLabels[key]}
                <input
                  type="number"
                  min={0}
                  step={1}
                  value={draft.limits[key]}
                  onChange={(e) =>
                    setDraft({
                      ...draft,
                      limits: {
                        ...draft.limits,
                        [key]: Number(e.target.value),
                      },
                    })
                  }
                />
              </label>
            ),
          )}
        </div>
        <FieldError name="limits" errors={errors} />
        <p className="form-help">
          Одна разрешённая рабочая копия, последовательно. По достижении предела
          цикл запросит вмешательство. Стоимость неизвестна, если агент её не
          сообщил.
        </p>
      </details>
      <label className="form-field">
        Разрешение действует до (UTC)
        <input
          type="datetime-local"
          value={
            draft.expires_at
              ? new Date(draft.expires_at).toISOString().slice(0, 16)
              : ""
          }
          onChange={(e) =>
            setDraft({
              ...draft,
              expires_at: e.target.value ? `${e.target.value}:00.000Z` : "",
            })
          }
        />
        <FieldError name="expires_at" errors={errors} />
      </label>
      <p className="form-help">
        Доступ ограничен сроком локального согласия, максимум 7 дней. Разрешены
        локальные коммиты; отправка в Git, merge и deploy требуют отдельного
        поручения.
      </p>
    </>
  );
}
