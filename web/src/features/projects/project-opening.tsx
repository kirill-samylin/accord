"use client";
import { useQuery } from "@tanstack/react-query";
import { readPage } from "@/lib/api";
import type { Project, CatalogPage } from "@/lib/projects";
import { OpenProject } from "./open-project";
import { useProjectScope } from "./project-context";
export function ProjectOpening({
  project,
  target,
}: {
  project: Project;
  target: string;
}) {
  const scope = useProjectScope();
  const catalog = useQuery({
    queryKey: ["project-catalog", project.id],
    queryFn: ({ signal }) =>
      readPage<CatalogPage>("/app/projects", signal, scope.key),
    retry: false,
    gcTime: 0,
  });
  const allowed = catalog.data?.registry?.projects?.find(
    (item) => item.id === project.id,
  );
  return (
    <main className="welcome">
      <span className="project-mark">A</span>
      <h1>{project.name}</h1>
      <p className="muted">
        Откройте проект, чтобы начать рабочую сессию. Просмотр адреса сам её не
        продлевает.
      </p>
      {catalog.isPending ? (
        <p role="status">Проверяем доступ…</p>
      ) : allowed && catalog.data ? (
        <OpenProject page={catalog.data} project={allowed} target={target} />
      ) : (
        <div className="notice" role="alert">
          {catalog.isError
            ? "Не удалось проверить доступ"
            : "Проект недоступен для вашей учётной записи"}
          <button
            className="button button-secondary"
            onClick={() => catalog.refetch()}
          >
            Повторить
          </button>
        </div>
      )}
      <a href="/app/projects">Все проекты</a>
    </main>
  );
}
