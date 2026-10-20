const defaultRoles = [
  "coordinator",
  "analyst",
  "architect",
  "developer",
  "reviewer",
  "tester",
  "human",
];
const fields = [
  "title",
  "description",
  "epic_key",
  "owner",
  "areas",
  "target_key",
  "target_commit",
];

// Read pages reuse draft for query filters as well as failed forms. A query's
// event type or state must never become a create operation or capability.
export function createDraft(
  source: Record<string, string> = {},
  supportedRoles?: string[],
): Record<string, string> {
  const roles = supportedRoles?.length ? supportedRoles : defaultRoles;
  const role = roles.includes(source.role)
    ? source.role
    : roles.includes("developer")
      ? "developer"
      : roles[0];
  const priority = Number(source.priority);
  const draft: Record<string, string> = {
    type: source.type === "epic" ? "epic" : "task",
    role,
    priority:
      source.priority &&
      Number.isInteger(priority) &&
      priority >= 0 &&
      priority <= 100
        ? String(priority)
        : "2",
  };
  for (const name of fields)
    if (source[name] !== undefined) draft[name] = source[name];
  return draft;
}
