"use client";
import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { PageData } from "@/lib/types";
import {
  automationRoles,
  stageLabels,
  type AutomationRole,
} from "@/lib/automation";
import {
  profileDraftErrors,
  profileFields,
  type ProfileDraft,
} from "@/lib/agent-profiles";
import { actionBody } from "../mutations/contracts";
import { useProjectScope } from "../projects/project-context";
import { useAction } from "../mutations/useAction";
import { ActionNotice } from "../mutations/ActionNotice";
const instructions: Record<AutomationRole, string> = {
  analyst:
    "Уточни требование, ограничения и критерии приёмки. Составь проверяемый план непересекающихся задач с зависимостями и обязательными локальными тестами. Используй инструменты Accord для плана и вопросов; не выдумывай выполненную работу.",
  developer:
    "Выполни назначенную задачу, сохрани ограничения проекта и чужую работу. Проверь изменения, создай локальный commit и передай точный SHA и evidence в Accord. Не делай push, merge или deploy без отдельного поручения.",
  reviewer:
    "Независимо проверь назначенный точный SHA, корректность, безопасность и тесты. Не меняй исходники. Дай approve, changes_requested либо blocked с конкретными evidence и местами исправления. Не считай выход CLI подтверждением результата.",
  tester:
    "Проверь только назначенный SHA после положительного ревью. Запусти разрешённые реальные проверки через Accord, прочитай их результаты и передай evidence. Невыполненные тесты обозначай NOT-RUN; ошибки возвращай разработчику.",
};
function TeamRole({
  page,
  role,
  onCreated,
  onLock,
}: {
  page: PageData;
  role: AutomationRole;
  onCreated: () => void;
  onLock: (role: string, value: boolean) => void;
}) {
  const scope = useProjectScope();
  const [draft, setDraft] = useState<ProfileDraft>({
    name: stageLabels[role],
    role,
    description: "Команда автоматического цикла",
    instructions: instructions[role],
    provider: "unconfigured",
    model: "",
    version: "",
  });
  const [created, setCreated] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const action = useAction(
    page,
    () => {
      setCreated(true);
      onCreated();
    },
    "action",
    (locked) => onLock(role, locked),
  );
  const templatePage = {
    ...action.page,
    actions: {
      ...action.page.actions,
      "profile:create":
        action.page.view === "profiles"
          ? action.page.actions["profile:create"]
          : page.actions[`automation:team:${role}`],
    },
  };
  function submit() {
    const validation = profileDraftErrors(draft);
    setErrors(validation);
    if (Object.keys(validation).length) return;
    void action.send(
      "/web/profiles",
      actionBody(templatePage, "profile:create", profileFields(draft, false)),
    );
  }
  return (
    <section className="automation-card screen-stack">
      <h3>
        {stageLabels[role]} {created ? "· профиль создан" : ""}
      </h3>
      <fieldset disabled={created || action.locked || scope.locked}>
        <label className="form-field">
          Название
          <input
            value={draft.name}
            onChange={(e) => setDraft({ ...draft, name: e.target.value })}
          />
        </label>
        <label className="form-field">
          Агент по умолчанию
          <select
            value={draft.provider}
            onChange={(e) => setDraft({ ...draft, provider: e.target.value })}
          >
            <option value="unconfigured">Выбрать при настройке запуска</option>
            <option value="openai">Codex</option>
            <option value="anthropic">Claude Code</option>
          </select>
        </label>
        <label className="form-field">
          Инструкции
          <textarea
            rows={5}
            value={draft.instructions}
            onChange={(e) =>
              setDraft({ ...draft, instructions: e.target.value })
            }
          />
        </label>
        {Object.values(errors).map((error) => (
          <p className="field-error" key={error}>
            {error}
          </p>
        ))}
        {!created ? (
          <button
            type="button"
            className="button button-secondary"
            onClick={submit}
          >
            Создать профиль
          </button>
        ) : null}
      </fieldset>
      <ActionNotice
        message={action.message}
        uncertain={action.phase === "uncertain"}
        busy={action.busy}
        onRetry={() => void action.retry()}
      />
    </section>
  );
}
export function TeamSetup({
  page,
  close,
  refresh,
}: {
  page: PageData;
  close: () => void;
  refresh: () => void;
}) {
  const [locks, setLocks] = useState<Record<string, boolean>>({});
  const [created, setCreated] = useState(0);
  const busy = Object.values(locks).some(Boolean);
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !busy) close();
      }}
    >
      <DialogContent
        className="automation-dialog"
        aria-describedby="team-setup-help"
      >
        <DialogHeader>
          <DialogTitle>Подготовить команду</DialogTitle>
        </DialogHeader>
        <p id="team-setup-help" className="form-help">
          Четыре редактируемых профиля. Каждый создаётся отдельно;
          подтверждённых: {created} из 4. Подключение компьютера и согласие
          владельца на запуск настраиваются отдельно.
        </p>
        <div className="automation-team">
          {automationRoles.map((role) => (
            <TeamRole
              key={role}
              page={page}
              role={role}
              onCreated={() => setCreated((n) => n + 1)}
              onLock={(key, value) =>
                setLocks((old) => ({ ...old, [key]: value }))
              }
            />
          ))}
        </div>
        <p className="form-help">
          При неизвестном результате повторите ту же отправку в карточке. Уже
          созданные профили сохраняются, даже если подготовлена только часть
          команды.
        </p>
        <button
          type="button"
          className="button"
          disabled={busy}
          onClick={() => {
            refresh();
            close();
          }}
        >
          Готово
        </button>
      </DialogContent>
    </Dialog>
  );
}
