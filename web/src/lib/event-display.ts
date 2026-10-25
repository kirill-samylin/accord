// Display helpers that keep opaque server identifiers (profile revision refs,
// 64-hex session hashes) out of visible text. The raw value is still offered
// to callers so it can live in a title attribute for support/debugging.

export interface NamedProfile {
  id: string;
  name: string;
}

// Profile events carry `reason = "<profile id>@<version>"` (store/profiles.go).
const profileRevision = /^(ap_[0-9a-f]{32})@(\d+)$/;

export function parseProfileRevision(reason?: string) {
  const match = reason ? profileRevision.exec(reason) : null;
  return match ? { id: match[1], version: Number(match[2]) } : null;
}

// profileReasonText names the profile when the page provides it and falls
// back to a neutral label; it returns null when reason is not a profile ref.
export function profileReasonText(
  reason: string | undefined,
  profiles: readonly NamedProfile[] = [],
) {
  const ref = parseProfileRevision(reason);
  if (!ref) return null;
  const name = profiles.find((profile) => profile.id === ref.id)?.name;
  return `${name ? `«${name}»` : "Профиль агента"} · версия ${ref.version}`;
}

// Session ids are SHA-256 hashes; they mean nothing to a person.
export function isOpaqueSession(value?: string) {
  return !!value && /^[0-9a-f]{32,}$/i.test(value);
}

// actorLabel prefers a person/machine, then a neutral agent-session label.
export function actorLabel(owner?: string, machine?: string, session?: string) {
  if (owner) return machine ? `${owner} · ${machine}` : owner;
  if (machine) return machine;
  if (!session) return "";
  return isOpaqueSession(session) ? "Сессия агента" : session;
}
