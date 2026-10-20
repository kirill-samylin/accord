import type { MutationResult, PageData } from "../../lib/types.ts";

export type MutationProps = {
  page: PageData;
  onSuccess: (result: MutationResult) => void;
};
export type Payload = URLSearchParams | FormData;
export type PendingAction = { path: string; body: Payload };

export function actionID(
  page: PageData,
  operation: string,
  key?: string,
): string {
  return (
    (key ? page.actions?.[`${operation}:${key}`] : undefined) ??
    page.actions?.[operation] ??
    ""
  );
}

export function actionBody(
  page: PageData,
  operation: string,
  fields: Record<string, string> = {},
  key?: string,
): URLSearchParams {
  const body = new URLSearchParams({
    csrf: page.csrf ?? "",
    action_id: actionID(page, operation, key),
  });
  for (const [name, value] of Object.entries(fields)) {
    if (name !== "csrf" && name !== "action_id") body.append(name, value);
  }
  return body;
}

// Freeze values at the first dispatch. A retry cannot adopt edited fields, a new
// generation or a newer action ID after an unknown transport outcome.
export function snapshotPayload(body: Payload): Payload {
  if (body instanceof URLSearchParams) return new URLSearchParams(body);
  const copy = new FormData();
  body.forEach((value, key) => copy.append(key, value));
  return copy;
}

export function uploadBody(
  page: PageData,
  metadata: Record<string, string>,
  file: File,
): FormData {
  const body = new FormData();
  body.append("csrf", page.csrf ?? "");
  body.append("action_id", actionID(page, "upload"));
  for (const name of ["title", "scope", "scope_key", "epic_key", "design_id"])
    body.append(name, metadata[name] ?? "");
  body.append("file", file);
  return body;
}

export function stoppedBy(result: MutationResult): boolean {
  return [
    "LEASE_LOST",
    "NO_SESSION",
    "UNAUTHORIZED",
    "UNAUTHENTICATED",
    "AUTH_REQUIRED",
    "TOKEN_EXPIRED",
  ].includes(result.error?.code.toUpperCase() ?? "");
}
