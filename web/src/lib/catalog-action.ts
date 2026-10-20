import type { CatalogPage } from "./projects.ts";
export function catalogBody(
  page: CatalogPage,
  action: string,
  fields: URLSearchParams,
) {
  const body = new URLSearchParams(fields);
  body.set("csrf", page.registry.csrf);
  body.set("action_id", page.registry.actions[action] ?? "");
  return body;
}
