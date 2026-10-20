"use client";
import Link from "next/link";
import {
  ArrowUpRight,
  FolderKanban,
  RefreshCw,
  ShieldCheck,
} from "lucide-react";
import {
  Avatar,
  EmptyState,
  EventRows,
  Fact,
  Metric,
  PageHeading,
  Status,
} from "@/components/common";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import type { MutationResult, PageData } from "@/lib/types";
import { date, duration, label, number } from "@/lib/format";
import {
  FilterMenu,
  Pagination,
  SelectFilter,
  useFilters,
} from "../query-controls";
import { SessionControls } from "../mutations/SessionControls";
export function ProjectsScreen({ page }: { page: PageData }) {
  const overview = page.overview;
  return (
    <div className="page">
      <PageHeading
        title="Проект"
        description="Работа, участники и правила текущего проекта"
      />
      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">Обзор проекта</TabsTrigger>
          <TabsTrigger value="rules">Правила</TabsTrigger>
        </TabsList>
        <TabsContent value="overview">
          <div className="screen-stack">
            <div className="project-entity">
              <span className="project-mark">A</span>
              <div className="entity-main">
                <h2>{page.session.project}</h2>
                <span className="muted">Текущий проект Accord</span>
              </div>
              <Status value="active" />
            </div>
            <div className="metrics-grid">
              <Metric
                title="Задачи"
                value={overview?.tasks_total ?? "—"}
                href="/app/tasks?include_closed=true"
              />
              <Metric
                title="В работе"
                value={overview?.tasks_in_progress ?? "—"}
                href="/app/tasks?state=in_progress"
              />
              <Metric
                title="Завершено"
                value={overview?.tasks_done ?? "—"}
                href="/app/tasks?state=done"
              />
              <Metric
                title="Открытые эпики"
                value={overview?.open_epics ?? "—"}
                href="/app/tasks"
              />
            </div>
            {overview ? (
              <div className="project-progress">
                <div className="section-heading">
                  <h3>Выполнение задач</h3>
                  <span className="muted">
                    {overview.tasks_done} из {overview.tasks_total}
                  </span>
                </div>
                <progress
                  max={overview.tasks_total || 1}
                  value={overview.tasks_done}
                />
                <p className="form-help">
                  Доля завершённых задач в текущем проекте
                </p>
              </div>
            ) : null}
            <div className="project-links">
              {[
                {
                  href: "/app/tasks",
                  title: "Задачи и эпики",
                  text: "План работы и состояние выполнения",
                  Icon: FolderKanban,
                },
                {
                  href: "/app/agents",
                  title: "Участники",
                  text: "Агенты, люди и текущие аренды",
                  Icon: Avatar,
                },
                {
                  href: "/app/settings",
                  title: "Настройки",
                  text: "Правила и параметры сессии",
                  Icon: ShieldCheck,
                },
              ].map((item) => (
                <Link
                  prefetch={false}
                  className="entity-row"
                  href={item.href}
                  key={item.href}
                >
                  <FolderKanban size={20} />
                  <div className="entity-main">
                    <strong>{item.title}</strong>
                    <span className="muted">{item.text}</span>
                  </div>
                  <ArrowUpRight size={16} />
                </Link>
              ))}
            </div>
          </div>
        </TabsContent>
        <TabsContent value="rules">
          <Rules page={page} />
        </TabsContent>
      </Tabs>
    </div>
  );
}
function Rules({ page }: { page: PageData }) {
  const entries = Object.entries(page.session.rules ?? {});
  return (
    <div className="settings-section">
      <h2>Правила координации</h2>
      {entries.length ? (
        <dl>
          {entries.map(([key, value]) => (
            <Fact key={key} name={key} value={value} />
          ))}
        </dl>
      ) : (
        <EmptyState title="Правила не включены в этот снимок" />
      )}
      <p className="form-help">
        Параметры применяются сервером. Изменение конфигурации через веб пока
        недоступно.
      </p>
    </div>
  );
}
export function SettingsScreen({
  page,
  onSuccess,
}: {
  page: PageData;
  onSuccess: (result: MutationResult) => void;
}) {
  return (
    <div className="page">
      <PageHeading
        title="Настройки"
        description="Параметры проекта и вашей рабочей сессии"
      />
      <Tabs defaultValue="general">
        <TabsList>
          <TabsTrigger value="general">Основные</TabsTrigger>
          <TabsTrigger value="rules">Правила</TabsTrigger>
          <TabsTrigger value="session">Сессия</TabsTrigger>
        </TabsList>
        <TabsContent value="general">
          <div className="settings-section">
            <h2>Проект</h2>
            <dl>
              <Fact name="Название" value={page.session.project} />
              <Fact name="Владелец доступа" value={page.principal.owner} />
              <Fact name="Машина" value={page.principal.machine} />
              <Fact name="Роль" value={label(page.session.role)} />
            </dl>
            <h2>Время жизни</h2>
            <dl>
              <Fact
                name="Сессия"
                value={duration(page.session.session_ttl_seconds)}
              />
              <Fact
                name="Аренда задачи"
                value={duration(page.session.task_ttl_seconds)}
              />
              <Fact
                name="Рекомендуемый heartbeat"
                value={duration(page.session.heartbeat_every_seconds)}
              />
            </dl>
            <p className="form-help">
              Чтение страницы не продлевает сессию или аренду. Для вопроса
              используйте явное продление в карточке.
            </p>
          </div>
        </TabsContent>
        <TabsContent value="rules">
          <Rules page={page} />
        </TabsContent>
        <TabsContent value="session">
          <div className="settings-section">
            <h2>Текущая сессия</h2>
            <dl>
              <Fact
                name="ID"
                value={
                  <span className="mono break">{page.session.session_id}</span>
                }
              />
              <Fact
                name="Поколение роли"
                value={page.session.role_generation}
              />
              <Fact name="Действует до" value={date(page.session.expires_at)} />
              <Fact name="Роль" value={label(page.session.role)} />
            </dl>
            <SessionControls page={page} onSuccess={onSuccess} />
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
}
