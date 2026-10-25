"use client";
import { useState } from "react";
import { KeyRound } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Select } from "@/components/ui/select";
import { Disclosure } from "@/components/disclosure";
import type { AdminIdentity, AdminPage, Person } from "@/lib/projects";
import type { CatalogMutationResult, IssuedCredential } from "@/lib/types";
import { date } from "@/lib/format";
import { RegistryForm } from "../projects/registry-form";
import { projectRole } from "../projects/project-members";
import { useProjectScope } from "../projects/project-context";
import {
  activeAdmins,
  currentPerson,
  identityExpired,
  identityRevoked,
  identitySource,
  personActive,
  personProjects,
  personRole,
  validatePerson,
} from "./admin-model";
import { IssuedSecretDialog, TokenIssueForm } from "./token-dialog";
export function RoleSelect({ value = "member" }: { value?: string }) {
  return (
    <Select name="role" defaultValue={value}>
      <option value="member">Участник · права по проектам</option>
      <option value="admin">Администратор · люди и токены</option>
    </Select>
  );
}
export function PersonBadges({ person }: { person: Person }) {
  return (
    <span className="admin-badges">
      <span className={`role-badge person-role-${person.role}`}>
        {personRole(person.role)}
      </span>
      <span
        className={`status ${personActive(person) ? "status-active" : "status-cancelled"}`}
      >
        {personActive(person) ? "Активен" : "Выключен"}
      </span>
    </span>
  );
}
export function IdentityRow({
  page,
  identity,
  onSuccess,
}: {
  page: AdminPage;
  identity: AdminIdentity;
  onSuccess: (result: CatalogMutationResult) => void;
}) {
  const revoked = identityRevoked(identity);
  const expired = !revoked && identityExpired(identity);
  const self = identity.id === page.registry.identity.id;
  return (
    <div className="admin-identity">
      <div className="entity-main">
        <strong className="truncate">{identity.machine}</strong>
        <span className="muted">
          {identity.owner} · {identitySource(identity.source)} · до{" "}
          {date(identity.expires_at)}
          {self ? " · это вы" : ""}
        </span>
        <code className="mono break">{identity.id}</code>
      </div>
      <span
        className={`status ${revoked ? "status-credential_invalid" : expired ? "status-cancelled" : "status-done"}`}
      >
        {revoked ? "Отозван" : expired ? "Истёк" : "Действует"}
      </span>
      {!revoked ? (
        <RegistryForm
          page={page}
          actionKey={`admin:identity:${identity.id}:revoke`}
          endpoint={`/web/catalog/admin/identities/${identity.id}`}
          title="Отозвать"
          fields={{ action: "revoke" }}
          confirm={`Отозвать доступ «${identity.owner} · ${identity.machine}»? Токен перестанет действовать сразу, восстановить его нельзя.`}
          onSuccess={onSuccess}
        />
      ) : null}
    </div>
  );
}
export function PersonDialog({
  page,
  person,
  onOpenChange,
  onSuccess,
}: {
  page: AdminPage;
  person: Person | undefined;
  onOpenChange: (open: boolean) => void;
  onSuccess: (result: CatalogMutationResult) => void;
}) {
  const scope = useProjectScope();
  // The one-time secret lives here only; closing the dialog discards it.
  const [issued, setIssued] = useState<IssuedCredential | null>(null);
  const identities = person
    ? page.identities.filter((identity) => identity.person_id === person.id)
    : [];
  const projects = person ? personProjects(page, person) : [];
  const self = person && currentPerson(page)?.id === person.id;
  const lastAdmin =
    person &&
    person.role === "admin" &&
    personActive(person) &&
    activeAdmins(page) === 1;
  return (
    <Dialog
      open={Boolean(person)}
      onOpenChange={(open) => {
        if (scope.locked || issued) return;
        if (!open) setIssued(null);
        onOpenChange(open);
      }}
    >
      <DialogContent
        className="person-dialog"
        aria-describedby="person-dialog-description"
        showCloseButton={!scope.locked}
      >
        {person ? (
          <>
            <DialogHeader>
              <DialogTitle>{person.name}</DialogTitle>
              <PersonBadges person={person} />
            </DialogHeader>
            <p id="person-dialog-description" className="muted">
              {person.note || "Без заметки"} · версия {person.version} · изменён{" "}
              {date(person.updated_at)}
            </p>
            <section className="person-section">
              <h3>Данные и роль</h3>
              <RegistryForm
                key={`${person.id}:${person.version}`}
                page={page}
                actionKey={`admin:person:${person.id}:update`}
                endpoint={`/web/catalog/admin/people/${person.id}`}
                title="Сохранить"
                fields={{ action: "update", version: String(person.version) }}
                validate={validatePerson}
                onSuccess={onSuccess}
              >
                <label className="form-field">
                  <span>
                    Имя<span className="required-mark"> *</span>
                  </span>
                  <input
                    name="name"
                    maxLength={64}
                    autoComplete="off"
                    defaultValue={person.name}
                  />
                </label>
                <label className="form-field">
                  Заметка
                  <textarea
                    name="note"
                    maxLength={512}
                    rows={2}
                    defaultValue={person.note}
                  />
                </label>
                <label className="form-field">
                  Глобальная роль
                  <RoleSelect value={person.role} />
                </label>
                {lastAdmin ? (
                  <p className="form-help">
                    Последнего активного администратора нельзя понизить.
                  </p>
                ) : null}
              </RegistryForm>
            </section>
            <section className="person-section">
              <h3>Активность</h3>
              <p className="form-help">
                {personActive(person)
                  ? "При деактивации все доступы человека перестают работать; токены не отзываются, повторная активация вернёт доступ."
                  : "Человек выключен: его вход в браузере и запросы его агентов отвергаются."}
              </p>
              <RegistryForm
                page={page}
                actionKey={`admin:person:${person.id}:update`}
                endpoint={`/web/catalog/admin/people/${person.id}`}
                title={personActive(person) ? "Деактивировать" : "Активировать"}
                fields={{
                  action: personActive(person) ? "deactivate" : "activate",
                  version: String(person.version),
                }}
                confirm={
                  personActive(person)
                    ? `Деактивировать «${person.name}»? Доступ по всем его токенам приостановится.`
                    : `Активировать «${person.name}»? Его токены снова начнут действовать.`
                }
                disabled={Boolean(self || lastAdmin)}
                onSuccess={onSuccess}
              />
              {self ? (
                <p className="form-help">Нельзя деактивировать себя.</p>
              ) : lastAdmin ? (
                <p className="form-help">
                  Нельзя деактивировать последнего активного администратора.
                </p>
              ) : null}
            </section>
            <section className="person-section">
              <h3>Проекты</h3>
              {projects.length ? (
                <ul className="admin-project-list">
                  {projects.map((project) => (
                    <li key={project.id}>
                      <strong>{project.name}</strong>
                      <span className="muted">
                        {" "}
                        · {projectRole(project.role)}
                        {project.state === "archived" ? " · архив" : ""}
                      </span>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="muted">Не участвует ни в одном проекте.</p>
              )}
            </section>
            <section className="person-section">
              <h3>Доступы и токены</h3>
              {identities.length ? (
                <div className="admin-identity-list">
                  {identities.map((identity) => (
                    <IdentityRow
                      key={identity.id}
                      page={page}
                      identity={identity}
                      onSuccess={onSuccess}
                    />
                  ))}
                </div>
              ) : (
                <p className="muted">Пока нет ни одного доступа.</p>
              )}
              <Disclosure
                className="registry-add"
                summary={
                  <>
                    <KeyRound size={14} aria-hidden /> Выдать токен
                  </>
                }
              >
                <TokenIssueForm
                  page={page}
                  person={person}
                  onIssued={setIssued}
                  onSuccess={onSuccess}
                />
              </Disclosure>
            </section>
          </>
        ) : null}
        <IssuedSecretDialog issued={issued} onClose={() => setIssued(null)} />
      </DialogContent>
    </Dialog>
  );
}
