// Per-tab drafts for the create dialog. sessionStorage may be unavailable
// (private mode, blocked storage); every access degrades to "no draft".
const draftKey = (scope: string) => `accord:create-draft:${scope}`;
const pendingKey = (scope: string) => `accord:pending-task:${scope}`;

function read(key: string): Record<string, string> | null {
  try {
    const raw = window.sessionStorage.getItem(key);
    if (!raw) return null;
    const value = JSON.parse(raw);
    if (!value || typeof value !== "object" || Array.isArray(value))
      return null;
    const out: Record<string, string> = {};
    for (const [name, field] of Object.entries(value))
      if (typeof field === "string") out[name] = field;
    return out;
  } catch {
    return null;
  }
}
function write(key: string, value: Record<string, string> | null) {
  try {
    if (value) window.sessionStorage.setItem(key, JSON.stringify(value));
    else window.sessionStorage.removeItem(key);
  } catch {
    // Storage is a convenience only.
  }
}

export const loadCreateDraft = (scope: string) => read(draftKey(scope));
export const saveCreateDraft = (
  scope: string,
  draft: Record<string, string> | null,
) => write(draftKey(scope), draft);

// A task draft parked while the user creates the epic it needs.
export const loadPendingTask = (scope: string) => read(pendingKey(scope));
export const savePendingTask = (
  scope: string,
  draft: Record<string, string> | null,
) => write(pendingKey(scope), draft);
// Ready only on the created epic's own page, after navigation has settled;
// opening earlier lets the route change close the dialog again.
export function pendingTaskReady(scope: string, path: string) {
  const pending = read(pendingKey(scope));
  return Boolean(pending?.epic_key) && epicFromPath(path) === pending?.epic_key;
}

export function epicFromPath(path: string | null | undefined) {
  const match = /\/tasks\/(E-\d+)(?:$|[/?#])/.exec(path ?? "");
  return match?.[1] ?? "";
}
