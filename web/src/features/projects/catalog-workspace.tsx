"use client";
import { useState } from "react";
import { useRouter } from "next/navigation";
import { useQueryClient } from "@tanstack/react-query";
import { Moon, Sun, Plus, FolderKanban, ArrowLeft } from "lucide-react";
import { ProjectLink as Link } from "@/components/project-link";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { CatalogPage } from "@/lib/projects";
import type { CatalogMutationResult } from "@/lib/types";
import { webToApp } from "@/lib/api";
import { useAppearance } from "../hooks/use-appearance";
import { useProjectScope } from "./project-context";
import { RegistryForm } from "./registry-form";
import { OpenProject } from "./open-project";
import { ProjectDetails } from "./project-details";
import { projectRole } from "./project-members";
export default function CatalogWorkspace({ page }: { page: CatalogPage }) {
  const [create, setCreate] = useState(false);
  const [archived, setArchived] = useState(false);
  const appearance = useAppearance();
  const scope = useProjectScope();
  const router = useRouter();
  const client = useQueryClient();
  const projects = page.registry.projects ?? [];
  const visible = projects.filter(
    (project) => archived || project.state !== "archived",
  );
  function success(result: CatalogMutationResult) {
    if (result.redirect) {
      const target = webToApp(result.redirect);
      if (target === "/app/login") {
        client.clear();
        setCreate(false);
        router.replace(target);
        return;
      }
      setCreate(false);
      router.replace(target);
    }
    void client.invalidateQueries();
  }
  return (
    <div className="catalog-workspace">
      <header className="catalog-header">
        <Link href="/app/projects" className="auth-brand">
          <span className="project-mark">A</span>Accord
        </Link>
        <div className="catalog-header-actions">
          <span className="muted truncate">{page.registry.identity.owner}</span>
          <button
            className="icon-button"
            aria-label={appearance.dark ? "Светлая тема" : "Тёмная тема"}
            onClick={appearance.toggle}
          >
            {appearance.dark ? <Sun size={17} /> : <Moon size={17} />}
          </button>
          <RegistryForm
            page={page}
            actionKey="logout"
            endpoint="/web/logout"
            title="Выйти"
            onSuccess={success}
          />
        </div>
      </header>
      <main className="page catalog-page" id="main-content">
        {page.project ? (
          <Link href="/app/projects" className="inline-link">
            <ArrowLeft size={16} />
            Все проекты
          </Link>
        ) : null}
        <div className="page-heading">
          <div className="page-heading-copy">
            <h1 tabIndex={-1}>
              {page.project ? "Настройки проекта" : "Проекты"}
            </h1>
            <p className="muted">
              {page.project
                ? "Репозитории, участники и доступ"
                : "Ваши репозитории и независимые рабочие пространства"}
            </p>
          </div>
          {!page.project ? (
            <button
              className="button"
              disabled={scope.locked || !page.registry.actions.create}
              onClick={() => setCreate(true)}
            >
              <Plus size={16} />
              Создать проект
            </button>
          ) : null}
        </div>
        {scope.locked ? (
          <p className="notice" role="status">
            Сначала дождитесь результата отправки или повторите её. Переход в
            другой проект пока недоступен.
          </p>
        ) : null}
        {page.project ? (
          <ProjectDetails page={page} onSuccess={success} />
        ) : (
          <>
            <div className="catalog-identity">
              <span className="muted">Ваш ID для приглашения в проект</span>
              <code className="mono break">{page.registry.identity.id}</code>
              <span className="muted">{page.registry.identity.machine}</span>
            </div>
            <label className="catalog-filter">
              <input
                type="checkbox"
                checked={archived}
                onChange={(event) => setArchived(event.target.checked)}
              />
              Показывать архивные
            </label>
            {visible.length ? (
              <div className="projects-grid">
                {visible.map((project) => (
                  <article className="project-card" key={project.id}>
                    <div className="project-detail-heading">
                      <span className="project-mark">
                        <FolderKanban size={19} />
                      </span>
                      <div className="entity-main">
                        <h2 className="truncate">{project.name}</h2>
                        <p className="muted">
                          {projectRole(project.role)} ·{" "}
                          {project.state === "archived" ? "Архив" : "Активный"}
                        </p>
                      </div>
                    </div>
                    <p className="form-help">
                      Репозитории: {project.repositories.length}
                    </p>
                    {project.repositories[0] ? (
                      <p className="mono break">
                        {project.repositories[0].url}
                      </p>
                    ) : (
                      <p className="muted">Репозиторий пока не подключён</p>
                    )}
                    <div className="project-card-actions">
                      <OpenProject page={page} project={project} />
                      <Link
                        className="button button-ghost"
                        href={`/app/projects/${project.id}`}
                      >
                        Параметры
                      </Link>
                    </div>
                  </article>
                ))}
              </div>
            ) : (
              <div className="empty-state">
                <FolderKanban size={30} />
                <h2>
                  {projects.length
                    ? "Нет активных проектов"
                    : "Пока нет доступных проектов"}
                </h2>
                <p className="muted">
                  Создайте свой проект или передайте владельцу ваш ID для
                  приглашения.
                </p>
              </div>
            )}
          </>
        )}
      </main>
      <Dialog
        open={create}
        onOpenChange={(value) => {
          if (!scope.locked) setCreate(value);
        }}
      >
        <DialogContent
          aria-describedby="project-create-description"
          showCloseButton={!scope.locked}
        >
          <DialogHeader>
            <DialogTitle>Создать проект</DialogTitle>
          </DialogHeader>
          <p id="project-create-description" className="muted">
            Укажите название и репозиторий. Текущий проект и его данные
            сохранятся.
          </p>
          <RegistryForm
            page={page}
            actionKey="create"
            endpoint="/web/catalog/projects"
            title="Создать проект"
            onSuccess={success}
          >
            <label className="form-field">
              Название проекта
              <input name="name" required maxLength={200} autoComplete="off" />
            </label>
            <label className="form-field">
              Репозиторий · необязательно
              <input
                name="repository_url"
                maxLength={2048}
                placeholder="https://github.com/owner/repository.git"
                spellCheck={false}
              />
            </label>
            <label className="form-field">
              Ветка · необязательно
              <input name="branch" maxLength={200} spellCheck={false} />
            </label>
            <p className="form-help">
              Код остаётся в вашем Git-сервисе. Пароли и токены не добавляйте в
              URL.
            </p>
          </RegistryForm>
        </DialogContent>
      </Dialog>
    </div>
  );
}
