"use client";
import { useState } from "react";
import { projectInitial } from "@/lib/project-initial";
import { useRouter } from "next/navigation";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, ChevronDown } from "lucide-react";
import { ProjectLink as Link } from "@/components/project-link";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { readPage, webToApp } from "@/lib/api";
import { isCatalogPage, type CatalogPage } from "@/lib/projects";
import type { PageData } from "@/lib/types";
import { ActionNotice } from "../mutations/ActionNotice";
import { useProjectScope } from "./project-context";
import { useCatalogAction } from "./use-catalog-action";
function SwitcherItems({
  page,
  selected,
  close,
}: {
  page: CatalogPage;
  selected?: string;
  close: () => void;
}) {
  const router = useRouter();
  const client = useQueryClient();
  const action = useCatalogAction(
    page,
    (result) => {
      close();
      void client.invalidateQueries();
      if (result.redirect) router.push(webToApp(result.redirect));
    },
    "/app/projects",
  );
  return (
    <>
      {page.registry.projects?.map((project) => (
        <DropdownMenuItem
          key={project.id}
          disabled={action.locked}
          onSelect={(event) => {
            event.preventDefault();
            action.send(
              "/web/catalog/switch",
              `switch:${project.id}`,
              new URLSearchParams({ project_id: project.id }),
            );
          }}
        >
          <span className="project-mark">{projectInitial(project.name)}</span>
          <span className="entity-main truncate">
            {project.name}
            {project.state === "archived" ? " · архив" : ""}
          </span>
          {project.id === selected ? <Check size={15} /> : null}
        </DropdownMenuItem>
      ))}
      {!page.registry.projects?.length ? (
        <p className="form-help">Нет доступных проектов</p>
      ) : null}
      <ActionNotice
        message={action.message}
        busy={action.busy}
        uncertain={action.phase === "uncertain"}
        onRetry={action.retry}
      />
      {action.phase === "blocked" ? (
        <button
          className="button button-ghost"
          onClick={() => void action.refresh()}
        >
          Обновить список
        </button>
      ) : null}
    </>
  );
}
export function ProjectSwitcher({ page }: { page: PageData }) {
  const [open, setOpen] = useState(false);
  const scope = useProjectScope();
  const catalog = useQuery({
    queryKey: ["project-catalog", scope.key],
    queryFn: ({ signal }) =>
      readPage<PageData | CatalogPage>("/app/projects", signal, scope.key),
    enabled: open,
    retry: false,
    gcTime: 0,
  });
  const name = page.project?.name ?? page.session.project;
  return (
    <DropdownMenu
      open={open}
      onOpenChange={(value) => {
        if (!scope.locked) setOpen(value);
      }}
    >
      <DropdownMenuTrigger asChild>
        <button
          className="project-switch"
          title={name}
          disabled={scope.locked}
          aria-label="Переключить проект"
        >
          <span className="project-mark">{projectInitial(name)}</span>
          <strong className="truncate">{name || "Accord"}</strong>
          <ChevronDown size={14} />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="project-switch-menu">
        {catalog.isPending ? (
          <p className="form-help" role="status">
            Загрузка проектов…
          </p>
        ) : catalog.isError ? (
          <div role="alert">
            <p>Не удалось загрузить проекты</p>
            <button
              className="button button-secondary"
              onClick={() => catalog.refetch()}
            >
              Повторить
            </button>
          </div>
        ) : isCatalogPage(catalog.data) ? (
          <SwitcherItems
            page={catalog.data}
            selected={page.project?.id}
            close={() => setOpen(false)}
          />
        ) : (
          <DropdownMenuItem disabled>
            <Check size={14} />
            {name}
          </DropdownMenuItem>
        )}
        <DropdownMenuItem asChild>
          <Link href="/app/projects">Все проекты</Link>
        </DropdownMenuItem>
        {page.project ? (
          <DropdownMenuItem asChild>
            <Link href={`/app/projects/${page.project.id}`}>
              Репозитории и доступ
            </Link>
          </DropdownMenuItem>
        ) : null}
        <DropdownMenuItem asChild>
          <Link href="/app/settings">Настройки проекта</Link>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
