"use client";
import type { CatalogPage } from "@/lib/projects";
import type { CatalogMutationResult } from "@/lib/types";
import { RegistryForm } from "./registry-form";
export function ProjectRepositories({
  page,
  onSuccess,
}: {
  page: CatalogPage;
  onSuccess: (result: CatalogMutationResult) => void;
}) {
  const project = page.project;
  if (!project) return null;
  const endpoint = `/web/catalog/projects/${project.id}/repositories`;
  return (
    <section className="settings-section">
      <h2>Репозитории</h2>
      <p className="form-help">
        Ссылки связаны с проектом. Доступ к коду настраивается на машине агента;
        сохранение ссылки не проверяет удалённый доступ.
      </p>
      {!project.repositories.length ? (
        <p className="muted">Репозитории пока не подключены</p>
      ) : null}
      {project.repositories.map((repository) => (
        <div className="repository-card" key={repository.id}>
          {project.can_manage ? (
            <>
              <RegistryForm
                page={page}
                actionKey={`repository:${repository.id}:update`}
                endpoint={`${endpoint}/${repository.id}`}
                title="Сохранить репозиторий"
                fields={{
                  action: "update",
                  version: String(repository.version),
                }}
                onSuccess={onSuccess}
              >
                <label className="form-field">
                  URL репозитория
                  <input
                    name="url"
                    required
                    maxLength={2048}
                    defaultValue={repository.url}
                    spellCheck={false}
                  />
                </label>
                <label className="form-field">
                  Ветка · необязательно
                  <input
                    name="branch"
                    maxLength={200}
                    defaultValue={repository.branch}
                    spellCheck={false}
                  />
                </label>
              </RegistryForm>
              <RegistryForm
                page={page}
                actionKey={`repository:${repository.id}:remove`}
                endpoint={`${endpoint}/${repository.id}`}
                title="Отключить репозиторий"
                fields={{
                  action: "remove",
                  version: String(repository.version),
                }}
                confirm="Отключить ссылку на репозиторий? Код и история задач сохранятся."
                onSuccess={onSuccess}
              />
            </>
          ) : (
            <>
              <p className="break mono">{repository.url}</p>
              <p className="muted">
                Ветка: {repository.branch || "Не указана"}
              </p>
            </>
          )}
        </div>
      ))}
      {project.can_manage ? (
        <details className="registry-add">
          <summary>Подключить репозиторий</summary>
          <RegistryForm
            page={page}
            actionKey="repository:add"
            endpoint={endpoint}
            title="Подключить репозиторий"
            onSuccess={onSuccess}
          >
            <label className="form-field">
              URL репозитория
              <input
                name="url"
                required
                maxLength={2048}
                placeholder="https://github.com/owner/repository.git"
                spellCheck={false}
              />
            </label>
            <label className="form-field">
              Ветка · необязательно
              <input name="branch" maxLength={200} spellCheck={false} />
            </label>
          </RegistryForm>
        </details>
      ) : null}
    </section>
  );
}
