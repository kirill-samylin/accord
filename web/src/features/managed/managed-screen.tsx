"use client";
import { useState } from "react";
import { Plus, RefreshCw } from "lucide-react";
import { EmptyState, PageHeading } from "@/components/common";
import { ProjectLink as Link } from "@/components/project-link";
import { date } from "@/lib/format";
import { providerName, runLabels } from "@/lib/managed-agents";
import type { MutationResult, PageData } from "@/lib/types";
import { actionID } from "../mutations/contracts";
import { useProjectScope } from "../projects/project-context";
import { AgentNavigation } from "../profiles/agent-navigation";
import { HostCard, EnrollmentHelp } from "./host-card";
import { StartDialog } from "./start-dialog";
export function ManagedScreen({
  page,
  refresh,
  onSuccess,
}: {
  page: PageData;
  refresh: () => void;
  onSuccess: (result: MutationResult) => void;
}) {
  const [start, setStart] = useState(false);
  const scope = useProjectScope();
  const hosts = page.managed?.hosts ?? [];
  const runs = page.managed?.runs ?? [];
  const canStart =
    page.project?.can_write !== false &&
    hosts.some((host) => host.can_control && host.state === "active") &&
    Boolean(actionID(page, "managed:start"));
  return (
    <div className="page managed-page">
      <AgentNavigation current="managed" />
      <PageHeading
        title="Управление агентами"
        description="Передайте новое требование команде или управляйте отдельными агентами по существующим задачам."
        actions={
          <>
            <button
              className="button button-secondary button-sm"
              onClick={refresh}
            >
              <RefreshCw size={14} />
              Обновить
            </button>
            <Link href="/app/automation" className="button button-sm">
              <Plus size={14} />
              Выполнить требование
            </Link>
            <button
              className="button button-secondary button-sm"
              disabled={!canStart || scope.locked}
              onClick={() => setStart(true)}
            >
              Ручной запуск агента
            </button>
          </>
        }
      />
      {!canStart ? (
        <p className="form-help">
          {page.project?.can_write === false
            ? "У вас доступ только для просмотра."
            : "Для запуска нужен подключённый компьютер с разрешением владельца на управление."}
        </p>
      ) : null}
      {page.managed?.observed_at ? (
        <p className="form-help">
          Последние данные сервера: {date(page.managed.observed_at)}. Во время
          отсутствия обновлений состояние могло измениться.
        </p>
      ) : null}
      <section className="screen-stack" aria-labelledby="managed-hosts-title">
        <h2 id="managed-hosts-title">Компьютеры</h2>
        <p className="form-help">
          Связь с worker показана по последним данным сервера. Состояние
          процесса показано отдельно в карточке работы.
        </p>
        {hosts.length ? (
          <div className="managed-host-grid">
            {hosts.map((host) => (
              <HostCard
                key={host.id}
                host={host}
                observedAt={page.managed?.observed_at}
              />
            ))}
          </div>
        ) : (
          <EmptyState
            title="Компьютеры пока не подключены"
            description="Подключите свой компьютер и компьютер коллеги по инструкции ниже."
          />
        )}
        <EnrollmentHelp actorPrincipalID={page.managed?.actor_principal_id} />
      </section>
      <section className="screen-stack" aria-labelledby="managed-runs-title">
        <h2 id="managed-runs-title">Работа агентов</h2>
        {runs.length ? (
          <div className="list-surface">
            {runs.map((run) => (
              <Link
                key={run.id}
                href={`/app/managed/runs/${run.id}`}
                prefetch={false}
                className="managed-run-row"
              >
                <div className="entity-main">
                  <strong>
                    {run.profile_snapshot?.name || providerName(run.provider)}
                  </strong>
                  <span className="muted">
                    {hosts.find((host) => host.id === run.host_id)?.name ||
                      "Компьютер недоступен"}{" "}
                    · {run.workspace}
                  </span>
                </div>
                <span>{runLabels[run.state] ?? run.state}</span>
                <span className="muted">{run.task_key || "Без задачи"}</span>
                <time className="muted" dateTime={run.updated_at}>
                  {date(run.updated_at)}
                </time>
              </Link>
            ))}
          </div>
        ) : (
          <EmptyState
            title="Работа ещё не запущена"
            description="Выберите профиль, компьютер и разрешённую рабочую папку."
          />
        )}
      </section>
      {start ? (
        <StartDialog
          page={page}
          open={start}
          onOpenChange={setStart}
          onSuccess={onSuccess}
        />
      ) : null}
    </div>
  );
}
