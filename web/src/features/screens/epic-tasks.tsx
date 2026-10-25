"use client";
import { useQuery } from "@tanstack/react-query";
import { readPage } from "@/lib/api";
import type { PageData } from "@/lib/types";
import { EmptyState, TaskRow } from "@/components/common";
import { useProjectScope } from "../projects/project-context";

// An epic page lists the tasks that belong to it; otherwise the user has
// no way to see an epic's contents without going back to filters.
export function EpicTasks({ epicKey }: { epicKey: string }) {
  const scope = useProjectScope();
  const tasks = useQuery({
    queryKey: ["epic-tasks", scope.key, epicKey],
    queryFn: ({ signal }) =>
      readPage<PageData>(
        scope.app(
          `/app/tasks?epic_key=${encodeURIComponent(epicKey)}&include_closed=true`,
        ),
        signal,
      ),
    retry: false,
  });
  const items = tasks.data?.tasks ?? [];
  return (
    <section className="epic-tasks" aria-labelledby="epic-tasks-title">
      <div className="section-heading">
        <h3 id="epic-tasks-title">
          Задачи эпика{items.length ? ` · ${items.length}` : ""}
        </h3>
      </div>
      <div className="list-surface">
        {tasks.isPending ? (
          <div className="skeleton-lines" role="status">
            Загрузка задач…
          </div>
        ) : tasks.isError ? (
          <div className="compact-empty">
            <span>Задачи не загрузились</span>
            <button
              className="button button-secondary button-sm"
              onClick={() => tasks.refetch()}
            >
              Повторить
            </button>
          </div>
        ) : items.length ? (
          items.map((task) => <TaskRow key={task.key} task={task} />)
        ) : (
          <EmptyState
            title="В эпике пока нет задач"
            description="Создайте задачу и выберите этот эпик."
          />
        )}
      </div>
    </section>
  );
}
