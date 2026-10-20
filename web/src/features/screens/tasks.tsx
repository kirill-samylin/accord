"use client";
import { useState } from "react";
import { Columns3, List, Plus } from "lucide-react";
import { EmptyState, PageHeading, Status, TaskRow } from "@/components/common";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import type { MutationResult, PageData, Task } from "@/lib/types";
import { label } from "@/lib/format";
import {
  ClearFilters,
  FilterMenu,
  Pagination,
  SearchField,
  SelectFilter,
  useFilters,
} from "../query-controls";
import { TaskPeek } from "./task-peek";
import { AssignmentLabel } from "../coordination/assignment-label";
import { groupTasks } from "@/lib/task-groups";
export function TasksScreen({
  page,
  onCreate,
  onSuccess,
}: {
  page: PageData;
  onCreate: () => void;
  onSuccess: (result: MutationResult) => void;
}) {
  const filters = useFilters();
  const canCreate = page.project?.can_write !== false && !page.can_continue;
  const [peek, setPeek] = useState<Task | null>(null);
  const questions = page.view === "questions";
  const [view, setView] = useState("all");
  const source =
    questions && view === "mine"
      ? (page.my_questions ?? [])
      : (page.tasks ?? []);
  const tasks =
    questions && view === "ready"
      ? source.filter(
          (task) => task.state === "ready" && !task.blocked && !task.lease,
        )
      : source;
  const board = filters.params.get("layout") === "board";
  const states = page.states ?? [
    "created",
    "ready",
    "in_progress",
    "done",
    "cancelled",
  ];
  const filterCount = [
    "state",
    "role",
    "epic_key",
    "owner",
    "include_closed",
  ].filter((key) => filters.params.get(key)).length;
  const filtered = filterCount > 0 || Boolean(page.query);
  function list(items: Task[], epic = false) {
    return items.length ? (
      <div className="list-surface">
        {groupTasks(items, epic ? ["open", "done", "cancelled"] : states).map(
          ({ state, tasks: group }) => {
            return (
              <section key={state}>
                <div className="list-group-heading">
                  <Status value={state} />
                  <span className="muted">{group.length}</span>
                </div>
                {group.map((task) => (
                  <TaskRow key={task.key} task={task} onPeek={setPeek} />
                ))}
              </section>
            );
          },
        )}
      </div>
    ) : (
      <EmptyState
        title={
          filtered
            ? "Задачи не найдены"
            : questions
              ? "Вопросов пока нет"
              : "Создайте первую задачу"
        }
        description={
          filtered
            ? "Измените условия поиска или сбросьте фильтры."
            : questions
              ? "Здесь появятся вопросы, требующие участия человека."
              : "Разделите работу на задачи и назначьте роль исполнителя."
        }
        action={
          filtered ? (
            <ClearFilters />
          ) : (
            <button
              className="button button-secondary"
              onClick={onCreate}
              disabled={!canCreate}
            >
              <Plus size={14} />
              Новая задача
            </button>
          )
        }
      />
    );
  }
  return (
    <div className="page">
      <PageHeading
        title={questions ? "Вопросы" : "Задачи"}
        description={
          questions
            ? "Решения, которые нужны команде для продолжения работы"
            : "Вся работа команды в одном месте"
        }
        actions={
          <button
            className="button"
            onClick={onCreate}
            disabled={!canCreate}
            aria-label="Новая задача"
          >
            <Plus size={14} />
            <span className="desktop-label">Новая задача</span>
          </button>
        }
      />
      <Tabs value={view} onValueChange={setView}>
        <TabsList>
          <TabsTrigger value="all">
            {questions ? "Все вопросы" : "Все задачи"}
          </TabsTrigger>
          {questions ? (
            <>
              <TabsTrigger value="mine">
                Мои вопросы
                {page.my_questions?.length
                  ? ` · ${page.my_questions.length}`
                  : ""}
              </TabsTrigger>
              <TabsTrigger value="ready">Ждут ответа</TabsTrigger>
            </>
          ) : (
            <TabsTrigger value="epics">
              Эпики{page.epics?.length ? ` · ${page.epics.length}` : ""}
            </TabsTrigger>
          )}
        </TabsList>
        <TabsContent value={view}>
          <div className="screen-stack">
            <div className="toolbar">
              <div className="toolbar-start">
                {!questions ? (
                  <SearchField
                    key={page.query ?? ""}
                    value={page.query}
                    onSearch={(q) => filters.set({ q })}
                  />
                ) : null}
                <FilterMenu count={filterCount}>
                  <SelectFilter
                    title="Статус"
                    value={page.state}
                    values={states}
                    onChange={(state) => filters.set({ state })}
                  />
                  {!questions ? (
                    <SelectFilter
                      title="Роль"
                      value={page.role}
                      values={page.roles ?? []}
                      onChange={(role) => filters.set({ role })}
                    />
                  ) : null}
                  <SelectFilter
                    title="Эпик"
                    value={page.epic_key}
                    values={(page.epics ?? []).map((epic) => ({
                      value: epic.key,
                      title: `${epic.key} · ${epic.title}`,
                    }))}
                    onChange={(epic_key) => filters.set({ epic_key })}
                  />
                  <form
                    className="form-field"
                    onSubmit={(event) => {
                      event.preventDefault();
                      filters.set({
                        owner: String(
                          new FormData(event.currentTarget).get("owner") ?? "",
                        ),
                      });
                    }}
                  >
                    <label htmlFor="owner-filter">Исполнитель</label>
                    <input
                      id="owner-filter"
                      name="owner"
                      defaultValue={page.draft.owner}
                      placeholder="Владелец задачи"
                    />
                    <button className="button button-secondary button-sm">
                      Применить
                    </button>
                  </form>
                  <label className="check-field">
                    <input
                      type="checkbox"
                      checked={filters.params.get("include_closed") === "true"}
                      onChange={(event) =>
                        filters.set({
                          include_closed: event.target.checked
                            ? "true"
                            : undefined,
                        })
                      }
                    />
                    Показывать закрытые
                  </label>
                  {filtered ? <ClearFilters /> : null}
                </FilterMenu>
                {filtered ? <ClearFilters /> : null}
              </div>
              <div className="toolbar-end">
                <span className="toolbar-count">
                  {view === "epics" ? (page.epics?.length ?? 0) : tasks.length}{" "}
                  в выборке
                </span>
                {view !== "epics" ? (
                  <div className="segmented">
                    <button
                      aria-label="Список"
                      aria-pressed={!board}
                      onClick={() => filters.set({ layout: "list" }, false)}
                    >
                      <List size={16} />
                    </button>
                    <button
                      aria-label="Доска"
                      aria-pressed={board}
                      onClick={() => filters.set({ layout: "board" }, false)}
                    >
                      <Columns3 size={16} />
                    </button>
                  </div>
                ) : null}
              </div>
            </div>
            {view === "epics" ? (
              list(page.epics ?? [], true)
            ) : board && tasks.length ? (
              <div className="kanban" aria-label="Задачи по статусам">
                {states.map((state) => (
                  <section className="kanban-column" key={state}>
                    <div className="kanban-heading">
                      <Status value={state} />
                      <span className="muted">
                        {tasks.filter((task) => task.state === state).length}
                      </span>
                    </div>
                    <div className="kanban-items">
                      {tasks
                        .filter((task) => task.state === state)
                        .map((task) => (
                          <button
                            className="kanban-card"
                            key={task.key}
                            onClick={() => setPeek(task)}
                          >
                            <div className="kanban-card-key">
                              <span className="mono muted">{task.key}</span>
                              <span className="muted">P{task.priority}</span>
                            </div>
                            <strong>{task.title}</strong>
                            <div className="kanban-card-meta">
                              <span>{label(task.role)}</span>
                              <span className="truncate">
                                <AssignmentLabel task={task} />
                              </span>
                            </div>
                            {task.blocked ? <Status value="blocked" /> : null}
                          </button>
                        ))}
                    </div>
                  </section>
                ))}
              </div>
            ) : (
              list(tasks)
            )}
            <Pagination
              cursor={view === "epics" ? undefined : page.next_cursor}
              epicCursor={view === "epics" ? page.epics_next_cursor : undefined}
            />
          </div>
        </TabsContent>
      </Tabs>
      <TaskPeek
        task={peek}
        onClose={() => setPeek(null)}
        onSuccess={onSuccess}
      />
    </div>
  );
}
