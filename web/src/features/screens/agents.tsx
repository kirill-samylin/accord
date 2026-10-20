"use client";
import Link from "next/link";
import { Bot, RefreshCw } from "lucide-react";
import {
  AgentRow,
  Avatar,
  EmptyState,
  EventRows,
  Fact,
  PageHeading,
  Status,
  TaskRow,
} from "@/components/common";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import type { PageData } from "@/lib/types";
import { date, label } from "@/lib/format";
import {
  FilterMenu,
  Pagination,
  SelectFilter,
  useFilters,
  ClearFilters,
} from "../query-controls";
export function AgentsScreen({
  page,
  refresh,
}: {
  page: PageData;
  refresh: () => void;
}) {
  const filters = useFilters();
  const agents = page.agents ?? [];
  const filterCount = ["role", "owner", "kind"].filter((key) =>
    filters.params.get(key),
  ).length;
  return (
    <div className="page">
      <PageHeading
        title="Агенты"
        description="Участники проекта и задачи, которые они удерживают"
        actions={
          <button
            className="button button-secondary button-sm"
            onClick={refresh}
          >
            <RefreshCw size={14} />
            Обновить
          </button>
        }
      />
      <Tabs
        value={
          page.agent_state ||
          (filters.params.get("include_closed") === "true" ? "all" : "active")
        }
        onValueChange={(state) =>
          filters.set({
            state: state === "all" ? undefined : state,
            include_closed:
              state === "all" || state !== "active" ? "true" : undefined,
          })
        }
      >
        <TabsList>
          <TabsTrigger value="all">Все сессии</TabsTrigger>
          <TabsTrigger value="active">Активные</TabsTrigger>
          <TabsTrigger value="expired">Истёкшие</TabsTrigger>
          <TabsTrigger value="credential_invalid">Без доступа</TabsTrigger>
        </TabsList>
      </Tabs>
      <div className="toolbar">
        <div className="toolbar-start">
          <FilterMenu count={filterCount}>
            <SelectFilter
              title="Участники"
              value={page.draft.kind ?? "agent"}
              values={[
                { value: "agent", title: "Агенты" },
                { value: "human", title: "Люди" },
                { value: "all", title: "Все" },
              ]}
              onChange={(kind) =>
                filters.set({ kind: kind || "all", role: undefined })
              }
            />
            <SelectFilter
              title="Роль"
              value={page.role}
              values={(page.roles ?? []).filter((role) =>
                page.draft.kind === "human"
                  ? role === "human"
                  : page.draft.kind === "all"
                    ? true
                    : role !== "human",
              )}
              onChange={(role) => filters.set({ role })}
            />
            <ClearFilters />
          </FilterMenu>
          <span className="toolbar-count">{agents.length} в выборке</span>
        </div>
        <span
          className="observation muted"
          title="Серверный снимок доступа и аренд"
        >
          Обновлено {date(page.agents_observed_at)}
        </span>
      </div>
      <div className="list-surface">
        {agents.length ? (
          <>
            <div className="agent-list-head">
              <span>Участник</span>
              <span>Задача</span>
              <span>Состояние</span>
              <span>Сессия до</span>
            </div>
            {agents.map((agent) => (
              <AgentRow key={agent.session_id} agent={agent} />
            ))}
          </>
        ) : (
          <EmptyState
            title="Сессии не найдены"
            description="Подключённые агенты появятся после открытия сессии. Попробуйте изменить фильтры."
            icon={<Bot size={26} />}
            action={
              filterCount || page.agent_state ? <ClearFilters /> : undefined
            }
          />
        )}
      </div>
      <Pagination cursor={page.next_cursor} />
      <p className="caption-note muted">
        Состояние сессии отражает доступ к доске. Работа процесса и модель не
        отслеживаются.
      </p>
    </div>
  );
}
export function AgentDetail({ page }: { page: PageData }) {
  const agent = page.agent;
  if (!agent)
    return (
      <div className="page">
        <EmptyState title="Сессия не найдена" />
      </div>
    );
  return (
    <div className="page">
      <div className="agent-identity">
        <Avatar name={agent.owner} role={agent.role} agent />
        <div>
          <PageHeading
            title={agent.owner || label(agent.role)}
            description={`${label(agent.role)} · ${agent.machine}`}
          />
        </div>
        <Status value={agent.status} />
      </div>
      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">Обзор</TabsTrigger>
          <TabsTrigger value="tasks">Задачи</TabsTrigger>
          <TabsTrigger value="events">События</TabsTrigger>
          <TabsTrigger value="session">Сессия</TabsTrigger>
        </TabsList>
        <TabsContent value="overview">
          <div className="detail-layout">
            <div className="screen-stack">
              <section>
                <h2 className="mb-4">Текущая работа</h2>
                {agent.task ? (
                  <div className="list-surface">
                    <TaskRow task={agent.task} />
                  </div>
                ) : (
                  <EmptyState
                    title="Нет действующей аренды задачи"
                    description="Участник не удерживает задачу в этом снимке."
                  />
                )}
              </section>
              <section>
                <h2 className="mb-4">Последние события</h2>
                <EventRows events={(page.activity ?? []).slice(0, 6)} />
              </section>
            </div>
            <aside className="properties-panel">
              <h3>Участник</h3>
              <dl>
                <Fact name="Роль" value={label(agent.role)} />
                <Fact name="Машина" value={agent.machine} />
                <Fact name="Владелец" value={agent.owner} />
                <Fact
                  name="Состояние"
                  value={<Status value={agent.status} />}
                />
                <Fact name="Сессия до" value={date(agent.expires_at)} />
                <Fact
                  name="Последнее событие"
                  value={date(agent.last_event_at)}
                />
              </dl>
            </aside>
          </div>
        </TabsContent>
        <TabsContent value="tasks">
          {agent.task ? (
            <div className="list-surface">
              <TaskRow task={agent.task} />
            </div>
          ) : (
            <EmptyState title="Нет действующей аренды" />
          )}
          <p className="caption-note muted">
            Здесь показана текущая задача сессии. История сохраняется в событиях
            и попытках задач.
          </p>
        </TabsContent>
        <TabsContent value="events">
          <EventRows events={page.activity ?? []} />
          <Pagination cursor={page.next_cursor} />
        </TabsContent>
        <TabsContent value="session">
          <div className="settings-section">
            <h2>Параметры подключения</h2>
            <dl>
              <Fact
                name="ID сессии"
                value={<span className="mono break">{agent.session_id}</span>}
              />
              <Fact
                name="Run ID"
                value={<span className="mono break">{agent.run_id}</span>}
              />
              <Fact name="Поколение роли" value={agent.role_generation} />
              <Fact name="Сессия до" value={date(agent.expires_at)} />
              <Fact name="Машина" value={agent.machine} />
            </dl>
            <p className="form-help">
              Сессия создаётся клиентом при подключении к Accord. Изменение
              модели и запуск процесса требуют профиля исполнения; этот контракт
              пока не подключён.
            </p>
            <Link className="inline-link" href="/app/settings">
              Правила проекта и сроки аренд
            </Link>
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
}
