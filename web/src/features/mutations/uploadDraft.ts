import type { DesignAsset } from "../../lib/types.ts";

const scopes = new Set(["project", "epic", "task"]);

// List filters contain scope:"" for the unfiltered page. That is not an upload
// scope. Version links additionally inherit the existing material's identity;
// Go continues to enforce that all versions retain their original binding.
export function uploadDraft(
  source: Record<string, string> = {},
  assets: DesignAsset[] = [],
): Record<string, string> {
  const existing = source.design_id
    ? assets.find((asset) => asset.design_id === source.design_id)
    : undefined;
  const scope = scopes.has(source.scope)
    ? source.scope
    : existing && scopes.has(existing.scope)
      ? existing.scope
      : "project";
  const scopeKey =
    source.scope_key || (existing?.scope === scope ? existing.scope_key : "");
  return {
    title: source.title ?? existing?.title ?? "",
    scope,
    scope_key: scope === "project" ? "" : scopeKey,
    epic_key: scope === "task" ? (source.epic_key ?? "") : "",
    design_id: source.design_id ?? "",
  };
}
