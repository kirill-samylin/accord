"use client";
import { useState } from "react";
import { PageHeading, EmptyState } from "@/components/common";
import { ProjectLink as Link } from "@/components/project-link";
import { RefreshCw, Settings2, Users } from "lucide-react";
import { workflowLabels } from "@/lib/automation";
import { workflowReason } from "@/lib/automation-reasons";
import { date } from "@/lib/format";
import type { MutationProps } from "../mutations/contracts";
import { RequirementForm } from "./requirement-form";
import { PolicyDialog } from "./policy-dialog";
import { TeamSetup } from "./team-setup";
import { WorkflowControls } from "./workflow-controls";
import { WorkflowQuestions } from "./workflow-questions";
import { WorkflowProgress, WorkflowReports } from "./workflow-progress";
import { PolicyDetail } from "./policy-detail";
export function AutomationScreen({
  page,
  refresh,
  onSuccess,
}: MutationProps & { refresh: () => void }) {
  const [team, setTeam] = useState(false),
    [policy, setPolicy] = useState(false);
  const detail = page.workflow;
  return (
    <div className="page automation-page">
      <PageHeading
        title={detail ? "Выполнение требования" : "Выполнить требование"}
        description="Аналитик → разработчик → независимый ревьюер → тестировщик → результат."
        actions={
          <>
            <button
              className="button button-secondary button-sm"
              onClick={refresh}
            >
              <RefreshCw size={14} />
              Обновить
            </button>
            {!detail ? (
              <>
                <button
                  className="button button-secondary button-sm"
                  disabled={!page.can_manage_profiles}
                  onClick={() => setTeam(true)}
                >
                  <Users size={14} />
                  Подготовить команду
                </button>
                <button
                  className="button button-secondary button-sm"
                  disabled={!page.can_manage_automation}
                  onClick={() => setPolicy(true)}
                >
                  <Settings2 size={14} />
                  Настроить запуск
                </button>
              </>
            ) : (
              <Link
                href="/app/automation"
                className="button button-secondary button-sm"
              >
                Все требования
              </Link>
            )}
          </>
        }
      />
      {page.draft?.cursor ? (
        <Link
          href={
            detail
              ? `/app/automation/workflows/${detail.workflow.id}`
              : "/app/automation"
          }
          prefetch={false}
        >
          К началу списка
        </Link>
      ) : null}
      {page.automation?.observed_at ? (
        <p className="form-help">
          Последние данные: {date(page.automation.observed_at)}. Ожидание
          команды и подтверждённый запуск показаны отдельно.
        </p>
      ) : null}
      {detail ? (
        <>
          <WorkflowProgress
            detail={detail}
            managed={page.managed}
            delivery={page.managed_run}
          />
          <WorkflowQuestions page={page} onSuccess={onSuccess} />
          <WorkflowControls page={page} onSuccess={onSuccess} />
          <WorkflowReports detail={detail} />
          {detail.next_cursor ? (
            <Link
              className="button button-secondary"
              href={`/app/automation/workflows/${detail.workflow.id}?cursor=${encodeURIComponent(detail.next_cursor)}`}
              prefetch={false}
            >
              Следующая страница истории
            </Link>
          ) : null}
        </>
      ) : page.automation_policy ? (
        <PolicyDetail page={page} onSuccess={onSuccess} />
      ) : (
        <>
          <RequirementForm page={page} onSuccess={onSuccess} />
          {!page.can_manage_automation ? (
            <div className="notice">
              <p>
                Для автоматического выполнения нужен подключённый компьютер с
                локальным согласием на автоматизацию и реальными тестами.
              </p>
              <Link href="/app/managed" prefetch={false}>
                Подключение компьютеров и управление агентами →
              </Link>
            </div>
          ) : null}
          <section className="screen-stack">
            <h2>Разрешения команды</h2>
            {page.automation?.policies.length ? (
              <div className="list-surface">
                {page.automation.policies.map((entry) => (
                  <Link
                    key={entry.id}
                    href={`/app/automation/policies/${entry.id}`}
                    prefetch={false}
                    className="automation-list-row"
                  >
                    <strong>{entry.name}</strong>
                    <span>
                      {entry.workspace} · версия {entry.version} ·{" "}
                      {entry.state === "active" ? "действует" : entry.state}
                    </span>
                    <time dateTime={entry.expires_at}>
                      до {date(entry.expires_at)}
                    </time>
                  </Link>
                ))}
              </div>
            ) : (
              <p className="form-help">
                Подготовьте профили команды и настройте разрешение запуска.
                Обычные ручные запуски остаются в разделе «Управление».
              </p>
            )}
          </section>
          <section className="screen-stack">
            <h2>Требования и работа</h2>
            {page.automation?.workflows.length ? (
              <div className="list-surface">
                {page.automation.workflows.map((flow) => (
                  <Link
                    key={flow.id}
                    href={`/app/automation/workflows/${flow.id}`}
                    prefetch={false}
                    className="automation-list-row"
                  >
                    <strong className="truncate">
                      {flow.intake_text}
                      {flow.intake_truncated ? "…" : ""}
                    </strong>
                    <span>
                      {workflowLabels[flow.state] ?? flow.state}
                      {flow.reason_code
                        ? ` · ${workflowReason(flow.reason_code)}`
                        : ""}
                    </span>
                    <time dateTime={flow.updated_at}>
                      {date(flow.updated_at)}
                    </time>
                  </Link>
                ))}
              </div>
            ) : (
              <EmptyState
                title="На этой странице требований нет"
                description="Просмотрите другие страницы истории, если они доступны, или опишите новое требование выше."
              />
            )}
            {page.automation?.next_cursor ? (
              <Link
                className="button button-secondary"
                href={`/app/automation?cursor=${encodeURIComponent(page.automation.next_cursor)}`}
                prefetch={false}
              >
                Следующая страница
              </Link>
            ) : null}
          </section>
        </>
      )}
      {team ? (
        <TeamSetup page={page} close={() => setTeam(false)} refresh={refresh} />
      ) : null}
      {policy ? (
        <PolicyDialog
          page={page}
          close={() => setPolicy(false)}
          onSuccess={onSuccess}
        />
      ) : null}
    </div>
  );
}
