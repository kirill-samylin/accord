"use client";
import { useState } from "react";
import { useRouter } from "next/navigation";
import { useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Link2, Moon, Sun, UserPlus, Users } from "lucide-react";
import { ProjectLink as Link } from "@/components/project-link";
import { Select } from "@/components/ui/select";
import type { AdminPage } from "@/lib/projects";
import type { CatalogMutationResult } from "@/lib/types";
import { webToApp } from "@/lib/api";
import { plural } from "@/lib/format";
import { useAppearance } from "../hooks/use-appearance";
import { useProjectScope } from "../projects/project-context";
import { RegistryForm } from "../projects/registry-form";
import { projectRole } from "../projects/project-members";
import { AuditList } from "./audit-list";
import {
  identitySource,
  personViews,
  unlinkedIdentities,
  validatePerson,
} from "./admin-model";
import { PersonBadges, PersonDialog, RoleSelect } from "./person-card";
export default function AdminScreen({ page }: { page: AdminPage }) {
  const appearance = useAppearance();
  const scope = useProjectScope();
  const router = useRouter();
  const client = useQueryClient();
  const [selected, setSelected] = useState<string | null>(null);
  const [adding, setAdding] = useState(false);
  const people = personViews(page);
  const unlinked = unlinkedIdentities(page);
  const person = page.people.find((item) => item.id === selected);
  function success(result: CatalogMutationResult) {
    if (result.redirect) {
      const target = webToApp(result.redirect);
      if (target === "/app/login") {
        client.clear();
        router.replace(target);
        return;
      }
      router.replace(target);
    }
    void client.invalidateQueries();
  }
  function created(result: CatalogMutationResult) {
    setAdding(false);
    if (result.person) setSelected(result.person.id);
    success(result);
  }
  return (
    <div className="catalog-workspace admin-workspace">
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
      <main className="page catalog-page admin-page" id="main-content">
        <Link href="/app/projects" className="inline-link">
          <ArrowLeft size={16} />
          Все проекты
        </Link>
        <div className="page-heading">
          <div className="page-heading-copy">
            <h1 tabIndex={-1}>Администрирование</h1>
            <p className="muted">
              Люди, их токены и глобальные роли. Права внутри проектов
              по-прежнему задаются участием в проекте.
            </p>
          </div>
          <button
            className="button"
            disabled={
              scope.locked || !page.registry.actions["admin:person:create"]
            }
            onClick={() => setAdding((value) => !value)}
            aria-expanded={adding}
          >
            <UserPlus size={16} />
            Добавить человека
          </button>
        </div>
        {scope.locked ? (
          <p className="notice" role="status">
            Сначала дождитесь результата отправки или повторите её.
          </p>
        ) : null}
        {adding ? (
          <section className="admin-section" aria-labelledby="admin-add-title">
            <h2 id="admin-add-title">Новый человек</h2>
            <RegistryForm
              page={page}
              actionKey="admin:person:create"
              endpoint="/web/catalog/admin/people"
              title="Добавить"
              validate={validatePerson}
              onSuccess={created}
            >
              <div className="form-grid">
                <label className="form-field">
                  <span>
                    Имя<span className="required-mark"> *</span>
                  </span>
                  <input name="name" maxLength={64} autoComplete="off" />
                </label>
                <label className="form-field">
                  Глобальная роль
                  <RoleSelect />
                </label>
              </div>
              <label className="form-field">
                Заметка
                <textarea name="note" maxLength={512} rows={2} />
              </label>
              <p className="form-help">
                Токены выдаются из карточки человека после создания.
              </p>
            </RegistryForm>
          </section>
        ) : null}
        <section className="admin-section" aria-labelledby="admin-people-title">
          <div className="admin-section-head">
            <h2 id="admin-people-title">Люди</h2>
            <span className="muted">
              {people.length}{" "}
              {plural(people.length, "человек", "человека", "человек")}
            </span>
          </div>
          {people.length ? (
            <div className="admin-people">
              {people.map(({ person, identities, projects }) => (
                <button
                  type="button"
                  className="admin-person-row entity-row"
                  key={person.id}
                  onClick={() => setSelected(person.id)}
                  disabled={scope.locked}
                >
                  <span className="entity-main">
                    <strong className="truncate">{person.name}</strong>
                    <span className="muted truncate">
                      {person.note || person.id}
                    </span>
                  </span>
                  <PersonBadges person={person} />
                  <span className="admin-person-count">
                    {identities.length}{" "}
                    {plural(identities.length, "доступ", "доступа", "доступов")}
                  </span>
                  <span className="admin-person-projects muted">
                    {projects.length
                      ? projects
                          .map((p) => `${p.name} · ${projectRole(p.role)}`)
                          .join(", ")
                      : "Нет проектов"}
                  </span>
                </button>
              ))}
            </div>
          ) : (
            <div className="empty-state">
              <Users size={30} />
              <h3>Пока никого нет</h3>
              <p className="muted">Добавьте первого человека.</p>
            </div>
          )}
        </section>
        {unlinked.length ? (
          <section
            className="admin-section"
            aria-labelledby="admin-unlinked-title"
          >
            <div className="admin-section-head">
              <h2 id="admin-unlinked-title">Доступы без человека</h2>
              <span className="muted">
                Исторические токены без человека. Привяжите их вручную.
              </span>
            </div>
            <div className="admin-identity-list">
              {unlinked.map((identity) => (
                <div className="admin-identity" key={identity.id}>
                  <div className="entity-main">
                    <strong className="truncate">
                      {identity.owner} · {identity.machine}
                    </strong>
                    <span className="muted">
                      {identitySource(identity.source)}
                      {identity.revoked ? " · отозван" : ""}
                    </span>
                    <code className="mono break">{identity.id}</code>
                  </div>
                  <RegistryForm
                    page={page}
                    actionKey={`admin:identity:${identity.id}:link`}
                    endpoint={`/web/catalog/admin/identities/${identity.id}`}
                    title="Привязать"
                    fields={{ action: "link" }}
                    disabled={!page.people.length}
                    onSuccess={success}
                  >
                    <label className="form-field">
                      <span className="admin-link-label">
                        <Link2 size={13} aria-hidden /> Привязать к человеку
                      </span>
                      <Select name="person_id" placeholder="Выберите человека">
                        {page.people.map((item) => (
                          <option value={item.id} key={item.id}>
                            {item.name}
                          </option>
                        ))}
                      </Select>
                    </label>
                  </RegistryForm>
                </div>
              ))}
            </div>
          </section>
        ) : null}
        <AuditList page={page} />
      </main>
      <PersonDialog
        page={page}
        person={person}
        onOpenChange={(open) => {
          if (!open) setSelected(null);
        }}
        onSuccess={success}
      />
    </div>
  );
}
