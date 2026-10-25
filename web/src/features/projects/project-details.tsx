"use client";
import type { CatalogPage } from "@/lib/projects";
import type { CatalogMutationResult } from "@/lib/types";
import { RegistryForm } from "./registry-form";
import { ProjectRepositories } from "./project-repositories";
import { ProjectMembers, projectRole } from "./project-members";
import { OpenProject } from "./open-project";
import { projectInitial } from "@/lib/project-initial";
export function ProjectDetails({
  page,
  onSuccess,
  justCreated,
}: {
  page: CatalogPage;
  onSuccess: (result: CatalogMutationResult) => void;
  justCreated?: boolean;
}) {
  const project = page.project;
  if (!project) return <p className="notice">Проект не найден</p>;
  return (
    <div className="project-settings-grid">
      <section className="settings-section">
        <div className="project-detail-heading">
          <span className="project-mark">{projectInitial(project.name)}</span>
          <div>
            <h2>{project.name}</h2>
            <p className="muted">
              {project.state === "archived" ? "Архив" : "Активный проект"} ·{" "}
              {projectRole(project.role)}
            </p>
          </div>
        </div>
        {justCreated ? (
          <p className="notice" role="status">
            Проект создан. Открываем его — там можно создать первую задачу.
          </p>
        ) : null}
        <p className="mono break">{project.id}</p>
        <OpenProject
          page={page}
          project={project}
          autoSubmit={justCreated && project.state !== "archived"}
        />
        {project.can_manage ? (
          <>
            <RegistryForm
              page={page}
              actionKey="update"
              endpoint={`/web/catalog/projects/${project.id}/update`}
              title="Сохранить название"
              fields={{
                version: String(project.version),
                state: project.state,
              }}
              onSuccess={onSuccess}
            >
              <label className="form-field">
                Название проекта
                <input
                  name="name"
                  required
                  maxLength={200}
                  defaultValue={project.name}
                />
              </label>
            </RegistryForm>
            {project.state === "archived" || project.can_archive === true ? (
              <>
                <RegistryForm
                  page={page}
                  actionKey="update"
                  endpoint={`/web/catalog/projects/${project.id}/update`}
                  title={
                    project.state === "archived"
                      ? "Восстановить проект"
                      : "Архивировать проект"
                  }
                  fields={{
                    name: project.name,
                    state: project.state === "archived" ? "active" : "archived",
                    version: String(project.version),
                  }}
                  confirm={
                    project.state === "active"
                      ? "Архивировать проект? Данные сохранятся, новая работа будет остановлена."
                      : undefined
                  }
                  onSuccess={onSuccess}
                />
                <p className="form-help">
                  Проект нельзя архивировать, пока агенты выполняют в нём
                  задачи. Архив сохраняет историю и доступ для чтения.
                </p>
              </>
            ) : (
              <p className="form-help">
                Исходный проект сохраняет прежние данные и не может быть
                архивирован.
              </p>
            )}
          </>
        ) : (
          <p className="form-help">
            Параметры и доступы изменяет владелец проекта.
          </p>
        )}
      </section>
      <div className="screen-stack">
        <ProjectRepositories page={page} onSuccess={onSuccess} />
        <ProjectMembers page={page} onSuccess={onSuccess} />
      </div>
    </div>
  );
}
