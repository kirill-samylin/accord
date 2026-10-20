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
  identity: { id: string; owner: string; machine: string };
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
export interface CatalogPage {
  name: "projects" | "project-settings";
  registry: Registry;
  project?: Project;
  members?: Member[];
  field_errors?: Record<string, string>;
}
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
