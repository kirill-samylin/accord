export interface Repository {
  id: string;
  url: string;
  branch: string;
  version: number;
}
export interface Project {
  id: string;
  name: string;
  state: "active" | "archived";
  role: "owner" | "editor" | "viewer";
  can_write: boolean;
  can_manage: boolean;
  can_archive: boolean;
  version: number;
  repositories: Repository[];
}
export interface Registry {
  identity: { id: string; owner: string; machine: string; admin?: boolean };
  csrf: string;
  actions: Record<string, string>;
  projects?: Project[];
}
export interface Member {
  identity_id: string;
  owner: string;
  machine: string;
  role: "owner" | "editor" | "viewer";
  version: number;
}
export type PersonRole = "member" | "admin";
export interface Person {
  id: string;
  name: string;
  note: string;
  role: PersonRole;
  active: boolean;
  version: number;
  created_at: string;
  updated_at: string;
}
export interface AdminIdentity {
  id: string;
  person_id: string;
  owner: string;
  machine: string;
  source: "legacy" | "catalog";
  expires_at: string;
  revoked: boolean;
}
export interface AdminProject {
  id: string;
  name: string;
  state: "active" | "archived";
}
export interface AdminMembership {
  project_id: string;
  identity_id: string;
  role: "owner" | "editor" | "viewer";
}
export interface AuditEntry {
  id: number;
  event: string;
  actor_id: string;
  person_id: string;
  identity_id: string;
  details: string;
  at: string | number;
}
export interface CatalogPage {
  name: "projects" | "project-settings" | "admin";
  registry: Registry;
  project?: Project;
  members?: Member[];
  people?: Person[];
  identities?: AdminIdentity[];
  projects?: AdminProject[];
  memberships?: AdminMembership[];
  audit?: AuditEntry[];
  field_errors?: Record<string, string>;
}
export interface AdminPage extends CatalogPage {
  name: "admin";
  people: Person[];
  identities: AdminIdentity[];
  projects: AdminProject[];
  memberships: AdminMembership[];
  audit: AuditEntry[];
}
export const personID = /^u_[a-f0-9]{32}$/;
export const projectID = /^p_[a-f0-9]{32}$/;
export function isCatalogPage(value: unknown): value is CatalogPage {
  if (!value || typeof value !== "object" || !("registry" in value))
    return false;
  const registry = (value as CatalogPage).registry;
  return Boolean(
    registry &&
    typeof registry.identity?.id === "string" &&
    typeof registry.csrf === "string" &&
    registry.actions &&
    typeof registry.actions === "object",
  );
}
export function isAdminPage(value: unknown): value is AdminPage {
  if (!isCatalogPage(value) || value.name !== "admin") return false;
  return (
    Array.isArray(value.people) &&
    Array.isArray(value.identities) &&
    Array.isArray(value.projects) &&
    Array.isArray(value.memberships) &&
    Array.isArray(value.audit)
  );
}
