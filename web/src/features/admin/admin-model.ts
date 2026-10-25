import type {
  AdminIdentity,
  AdminPage,
  AuditEntry,
  Person,
} from "../../lib/projects.ts";
import { date } from "../../lib/format.ts";
// Pure helpers for the administration screen (docs/admin.md). Nothing here
// touches secrets: issued tokens live only in component state.
const roleNames: Record<string, string> = {
  admin: "Администратор",
  member: "Участник",
};
export const personRole = (role: string) => roleNames[role] ?? role;
const sourceNames: Record<string, string> = {
  legacy: "Токен (legacy)",
  catalog: "Каталог",
};
export const identitySource = (source: string) => sourceNames[source] ?? source;
// Audit event names are not enumerated by the contract; unknown events fall
// back to the raw name so nothing is hidden.
export const auditEvents: Record<string, string> = {
  person_created: "Человек добавлен",
  person_updated: "Человек изменён",
  person_deactivated: "Человек деактивирован",
  person_activated: "Человек активирован",
  credential_issued: "Токен выдан",
  credential_revoked: "Токен отозван",
  identity_linked: "Identity привязана",
  admin_bootstrapped: "Назначен администратор (CLI)",
};
export const auditEvent = (event: string) => auditEvents[event] ?? event;
// `at` may arrive as RFC 3339 text or as unix seconds / milliseconds.
export function auditTime(at: string | number | undefined): string {
  if (at === undefined || at === "") return "—";
  if (typeof at === "number") {
    if (!Number.isFinite(at) || at <= 0) return "—";
    return date(new Date(at < 1e12 ? at * 1000 : at).toISOString());
  }
  if (/^\d+$/.test(at)) return auditTime(Number(at));
  return date(at);
}
export const personActive = (person: Person) => Boolean(person.active);
export const identityRevoked = (identity: AdminIdentity) =>
  Boolean(identity.revoked);
export function identityExpired(identity: AdminIdentity, now = Date.now()) {
  const until = new Date(identity.expires_at).getTime();
  return Number.isFinite(until) && until > 0 && until < now;
}
export interface PersonProject {
  id: string;
  name: string;
  state: string;
  role: string;
}
export interface PersonView {
  person: Person;
  identities: AdminIdentity[];
  projects: PersonProject[];
}
const roleRank: Record<string, number> = { owner: 3, editor: 2, viewer: 1 };
// Projects reach a person through the memberships of their identities; the
// strongest role wins when several identities share a project.
export function personProjects(page: AdminPage, person: Person) {
  const own = new Set(
    page.identities
      .filter((identity) => identity.person_id === person.id)
      .map((identity) => identity.id),
  );
  const found = new Map<string, PersonProject>();
  for (const membership of page.memberships) {
    if (!own.has(membership.identity_id)) continue;
    const project = page.projects.find(
      (item) => item.id === membership.project_id,
    );
    if (!project) continue;
    const previous = found.get(project.id);
    if (
      !previous ||
      (roleRank[membership.role] ?? 0) > (roleRank[previous.role] ?? 0)
    )
      found.set(project.id, { ...project, role: membership.role });
  }
  return [...found.values()].sort((a, b) => a.name.localeCompare(b.name));
}
export function personViews(page: AdminPage): PersonView[] {
  return page.people
    .map((person) => ({
      person,
      identities: page.identities.filter(
        (identity) => identity.person_id === person.id,
      ),
      projects: personProjects(page, person),
    }))
    .sort(
      (a, b) =>
        Number(personActive(b.person)) - Number(personActive(a.person)) ||
        a.person.name.localeCompare(b.person.name),
    );
}
export const unlinkedIdentities = (page: AdminPage) =>
  page.identities.filter(
    (identity) =>
      !identity.person_id ||
      !page.people.some((person) => person.id === identity.person_id),
  );
export const activeAdmins = (page: AdminPage) =>
  page.people.filter(
    (person) => personActive(person) && person.role === "admin",
  ).length;
// The signed-in admin's own person, found through the current identity.
export function currentPerson(page: AdminPage) {
  const self = page.identities.find(
    (identity) => identity.id === page.registry.identity.id,
  );
  return self ? page.people.find((p) => p.id === self.person_id) : undefined;
}
export function actorName(page: AdminPage, actorID: string) {
  const identity = page.identities.find((item) => item.id === actorID);
  if (!identity) return actorID || "—";
  const person = page.people.find((item) => item.id === identity.person_id);
  return person ? `${person.name} · ${identity.machine}` : identity.owner;
}
export function personName(page: AdminPage, personID: string) {
  return page.people.find((item) => item.id === personID)?.name ?? personID;
}
export function identityName(page: AdminPage, identityID: string) {
  const identity = page.identities.find((item) => item.id === identityID);
  return identity ? `${identity.owner} · ${identity.machine}` : identityID;
}
// `<owner>-<machine>.token`: path separators and control characters cannot
// be part of a download name, everything else is kept as is.
export function tokenFileName(owner: string, machine: string) {
  const clean = (value: string) =>
    value.replace(/[\\/\u0000-\u001f\u007f]+/g, "_").trim() || "token";
  return `${clean(owner)}-${clean(machine)}.token`;
}
export function validatePerson(values: URLSearchParams) {
  const errors: Record<string, string> = {};
  const name = (values.get("name") ?? "").trim();
  if (!name) errors.name = "Введите имя";
  else if (name.length > 64) errors.name = "Имя длиннее 64 символов";
  if ((values.get("note") ?? "").length > 512)
    errors.note = "Заметка длиннее 512 символов";
  if (!["member", "admin"].includes(values.get("role") ?? ""))
    errors.role = "Выберите роль";
  return errors;
}
export function validateCredential(values: URLSearchParams) {
  const errors: Record<string, string> = {};
  const machine = (values.get("machine") ?? "").trim();
  if (!machine) errors.machine = "Укажите название компьютера";
  else if (machine.length > 64) errors.machine = "Название длиннее 64 символов";
  const days = values.get("days") ?? "";
  if (!/^\d{1,3}$/.test(days) || Number(days) < 1 || Number(days) > 365)
    errors.days = "Срок — целое число дней от 1 до 365";
  return errors;
}
export function auditSubject(page: AdminPage, entry: AuditEntry) {
  const parts: string[] = [];
  if (entry.person_id) parts.push(personName(page, entry.person_id));
  if (entry.identity_id) parts.push(identityName(page, entry.identity_id));
  return parts.join(" · ");
}
