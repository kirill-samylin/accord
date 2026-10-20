"use client";
import { ProjectLink as Link } from "@/components/project-link";
import { useQuery } from "@tanstack/react-query";
import { useProjectScope } from "../projects/project-context";
import { ArrowUpRight, Bot, RefreshCw } from "lucide-react";
import {
  AgentRow,
  Avatar,
  EmptyState,
  EventRows,
  Metric,
  PageHeading,
  SectionHeading,
  Status,
  TaskRow,
} from "@/components/common";
import { readPage } from "@/lib/api";
import type { PageData } from "@/lib/types";
import { date, label, number } from "@/lib/format";
import { useDashboard } from "../hooks/use-dashboard";
export function OverviewScreen({
  page,
  onSessionExpired,
}: {
  page: PageData;
  onSessionExpired: () => void;
}) {
  const scope = useProjectScope();
  const live = useDashboard(page, onSessionExpired);
  const recent = useQuery({
    queryKey: ["recent-tasks", scope.key],
    queryFn: ({ signal }) =>
      readPage(scope.app("/app/tasks?include_closed=true"), signal),
    retry: false,
    gcTime: 0,
  });
  const overview = live.data?.overview ?? page.overview;
  const agents = live.data?.agents.sessions ?? page.agents ?? [];
  const activity = live.data?.activity.events ?? page.activity ?? [];
  if (!overview) return <EmptyState title="Состояние проекта недоступно" />;
  const counts = [
    { key: "created", count: overview.tasks_created },
    { key: "ready", count: overview.tasks_ready },
    { key: "in_progress", count: overview.tasks_in_progress },
    { key: "done", count: overview.tasks_done },
    { key: "cancelled", count: overview.tasks_cancelled },
  ];
  return (
    <div className="page overview-page">
      <PageHeading
        title="Обзор"
        description="Что происходит в проекте прямо сейчас"
        actions={
          <>
            <span className="observation muted">
              {date(overview.observed_at)}
            </span>
            <button
              className="icon-button"
              aria-label="Обновить состояние"
              disabled={live.isFetching || live.paused}
              onClick={() => live.refetch()}
            >
              <RefreshCw size={15} className={live.isFetching ? "spin" : ""} />
            </button>
          </>
        }
      />
      {live.isError ? (
        <div className="notice error-notice notice-row" role="alert">
          <span>
            Не удалось обновить. Показаны данные на {date(overview.observed_at)}
            .
          </span>
          <button
            className="button button-secondary button-sm"
            onClick={() => live.refetch()}
          >
            Повторить
          </button>
        </div>
      ) : null}
      <section>
        <SectionHeading title="Агенты" href="/app/agents" />
        {agents.length ? (
          <div className="agent-strip">
            {agents.slice(0, 8).map((agent) => (
              <Link
                prefetch={false}
                key={agent.session_id}
                className="agent-card"
                href={`/app/agents/${agent.session_id}`}
              >
                <div className="agent-card-head">
                  <Avatar name={agent.owner} role={agent.role} agent />
                  <div className="agent-card-copy">
                    <strong className="truncate">
                      {agent.owner || label(agent.role)}
                    </strong>
                    <span className="muted">{label(agent.role)}</span>
                  </div>
                  <span
                    className={`session-dot session-${agent.status}`}
                    title={label(agent.status)}
                  />
                </div>
                <div className="agent-card-task truncate">
                  {agent.task ? (
                    <>
                      <span className="mono muted">{agent.task.key}</span>{" "}
                      {agent.task.title}
                    </>
                  ) : (
                    <span className="muted">
                      Сессия с доступом · без задачи
                    </span>
                  )}
                </div>
                <div className="agent-card-meta muted">
                  <Bot size={11} />
                  <span className="truncate">{agent.machine}</span>
                  <ArrowUpRight size={12} />
                </div>
              </Link>
            ))}
          </div>
        ) : (
          <div className="compact-empty muted">
            <Bot size={18} />
            Нет активных сессий агентов
          </div>
        )}
      </section>
      <section className="metrics-grid">
        <Metric
          title="Всего задач"
          value={number(overview.tasks_total)}
          caption={`${overview.open_epics} открытых эпиков`}
          href="/app/tasks?include_closed=true"
        />
        <Metric
          title="В работе"
          value={number(overview.tasks_in_progress)}
          caption={`${overview.leased_tasks} действующих аренд`}
          href="/app/tasks?state=in_progress"
        />
        <Metric
          title="Ожидают выполнения"
          value={number(overview.tasks_ready)}
          caption={`${overview.tasks_blocked} заблокировано`}
          href="/app/tasks?state=ready"
        />
        <Metric
          title="Завершено"
          value={number(overview.tasks_done)}
          caption={`${overview.active_agent_sessions} сессий агентов с доступом`}
          href="/app/tasks?state=done"
        />
      </section>
      <div className="dashboard-charts">
        <section className="chart-panel">
          <SectionHeading title="Состояние задач" />
          <div
            className="state-chart"
            role="img"
            aria-label={counts
              .map((item) => `${label(item.key)}: ${item.count}`)
              .join(", ")}
          >
            {counts.map((item) => (
              <div key={item.key} className="state-bar-row">
                <Status value={item.key} />
                <div className="bar-track">
                  <span
                    className={`bar-fill fill-${item.key}`}
                    style={{
                      width: `${overview.tasks_total ? (item.count / overview.tasks_total) * 100 : 0}%`,
                    }}
                  />
                </div>
                <span className="tabular-nums">{item.count}</span>
              </div>
            ))}
          </div>
        </section>
        <section className="chart-panel">
          <SectionHeading title="Очередь проекта" />
          <dl className="queue-summary">
            <div>
              <dt>Готовы к работе</dt>
              <dd>{overview.tasks_ready}</dd>
            </div>
            <div>
              <dt>Ждут зависимостей</dt>
              <dd>{overview.tasks_blocked}</dd>
            </div>
            <div>
              <dt>Люди с доступом</dt>
              <dd>{overview.active_human_sessions}</dd>
            </div>
          </dl>
          <Link prefetch={false} className="inline-link" href="/app/questions">
            Открыть вопросы
            <ArrowUpRight size={14} />
          </Link>
        </section>
      </div>
      <div className="dashboard-bottom">
        <section className="min-w-0">
          <SectionHeading title="Задачи проекта" href="/app/tasks" />
          <div className="list-surface">
            {recent.isPending ? (
              <div className="skeleton-lines" role="status">
                Загрузка задач…
              </div>
            ) : recent.isError ? (
              <div className="compact-empty">
                <span>Задачи не загрузились</span>
                <button
                  className="button button-secondary button-sm"
                  onClick={() => recent.refetch()}
                >
                  Повторить
                </button>
              </div>
            ) : recent.data?.tasks?.length ? (
              recent.data.tasks
                .slice(0, 7)
                .map((task) => <TaskRow key={task.key} task={task} />)
            ) : (
              <EmptyState
                title="Задач пока нет"
                description="Создайте первую задачу для команды."
              />
            )}
          </div>
        </section>
        <section className="min-w-0">
          <SectionHeading title="Активность" href="/app/activity" />
          <div className="list-surface">
            <EventRows events={activity.slice(0, 5)} />
          </div>
        </section>
      </div>
    </div>
  );
}
