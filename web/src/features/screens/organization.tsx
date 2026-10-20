"use client";
import { ProjectLink as Link } from "@/components/project-link";
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
import { date, duration, eventLabels, number } from "@/lib/format";
import {
  FilterMenu,
  Pagination,
  SelectFilter,
  useFilters,
} from "../query-controls";
import { SessionControls } from "../mutations/SessionControls";
export function ActivityScreen({
  page,
  refresh,
}: {
  page: PageData;
  refresh: () => void;
}) {
  const filters = useFilters();
  return (
    <div className="page">
      <PageHeading
        title="Активность"
        description="Последовательная история работы проекта"
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
      <div className="toolbar">
        <div className="toolbar-start">
          <FilterMenu
            count={
              ["task_key", "session_id", "type"].filter((key) =>
                filters.params.has(key),
              ).length
            }
          >
            <form
              className="mutation-form"
              onSubmit={(event) => {
                event.preventDefault();
                const data = new FormData(event.currentTarget);
                filters.set({
                  task_key: String(data.get("task_key") ?? ""),
                  session_id: String(data.get("session_id") ?? ""),
                  type: String(data.get("type") ?? ""),
                });
              }}
            >
              <label className="form-field">
                Ключ задачи
                <input
                  name="task_key"
                  defaultValue={page.draft.task_key}
                  placeholder="T-1"
                />
              </label>
              <label className="form-field">
                ID сессии
                <input name="session_id" defaultValue={page.draft.session_id} />
              </label>
              <label className="form-field">
                Тип события
                <select name="type" defaultValue={page.draft.type ?? ""}>
                  <option value="">Все события</option>
                  {Object.entries(eventLabels).map(([type, title]) => (
                    <option key={type} value={type}>
                      {title}
                    </option>
                  ))}
                </select>
              </label>
              <button className="button">Применить</button>
            </form>
          </FilterMenu>
          {filters.params.size ? (
            <button
              className="button button-ghost button-sm"
              onClick={filters.reset}
            >
              Сбросить
            </button>
          ) : null}
        </div>
        <span className="observation muted">
          Обновлено {date(page.observed_at)}
        </span>
      </div>
      <div className="list-surface">
        <EventRows events={page.activity ?? []} />
      </div>
      <Pagination cursor={page.next_cursor} />
    </div>
  );
}
export function HistoryScreen({ page }: { page: PageData }) {
  const filters = useFilters();
  return (
    <div className="page">
      <PageHeading
        title="Решения"
        description="Сохранённые ответы с авторством на момент принятия"
      />
      <div className="toolbar">
        <form
          className="toolbar-start"
          onSubmit={(event) => {
            event.preventDefault();
            filters.set({
              epic_key: String(
                new FormData(event.currentTarget).get("epic_key") ?? "",
              ),
            });
          }}
        >
          <input
            name="epic_key"
            aria-label="Ключ эпика"
            placeholder="Эпик, например E-1"
            defaultValue={page.epic_key}
          />
          <button className="button button-secondary">Показать</button>
        </form>
      </div>
      <div className="list-surface">
        <EventRows events={page.decisions ?? []} decisions />
      </div>
      {page.legacy_decisions?.length ? (
        <section>
          <h2>Прежние ответы</h2>
          <p className="caption-note muted">
            Историческое авторство не записано. Показан текущий ответ задачи.
          </p>
          <div className="list-surface">
            {page.legacy_decisions.map((task) => (
              <div className="legacy-decision" key={task.key}>
                <Link prefetch={false} href={`/app/tasks/${task.key}`}>
                  <span className="mono muted">{task.key}</span> {task.title}
                </Link>
                <p className="answer-text">{task.evidence.answer}</p>
              </div>
            ))}
          </div>
        </section>
      ) : null}
      <Pagination cursor={page.next_cursor} />
    </div>
  );
}
