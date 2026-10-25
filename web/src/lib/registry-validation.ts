// Client mirrors of the Go catalog validators (internal/projects: validName,
// validID/legacy ids, repositoryFields). They trim the submitted values in
// place so a whitespace-padded field is never sent, and return Russian
// messages keyed by field name. The server remains the authority.

const encoder = new TextEncoder();

function trimmed(values: URLSearchParams, name: string) {
  const value = (values.get(name) ?? "").trim();
  if (values.has(name)) values.set(name, value);
  return value;
}

export function projectNameError(name: string) {
  if (!name) return "Введите название проекта";
  // Go counts bytes: 120 bytes is about 60 Cyrillic characters.
  if (encoder.encode(name).length > 120)
    return "Название слишком длинное — сократите его";
  if (/\p{Cc}/u.test(name)) return "Название содержит недопустимые символы";
  return "";
}

const memberID = /^(i_[0-9a-f]{32}|legacy_[0-9a-f]{64})$/;
export function memberIDError(id: string) {
  if (!id) return "Введите ID учётной записи";
  return memberID.test(id)
    ? ""
    : "ID учётной записи выглядит как i_ и 32 символа 0–9/a–f (или legacy_ и 64 символа) — скопируйте его из карточки человека в «Администрировании», блок «Доступы и токены»";
}

const branchInvalid = /\.\.|[\s~^:?*[\\]|^-|[/.]$/;
export function repositoryURLError(raw: string, required = true) {
  if (!raw) return required ? "Введите адрес репозитория" : "";
  let url: URL;
  try {
    url = new URL(raw);
  } catch {
    return repositoryHint;
  }
  const path = url.pathname.replace(/^\/+/, "");
  if (!url.hostname || !path || url.search || url.hash || raw.includes("\\"))
    return repositoryHint;
  if (url.protocol === "https:" && !url.username && !url.password) return "";
  if (url.protocol === "ssh:" && url.username === "git" && !url.password)
    return "";
  return repositoryHint;
}
const repositoryHint =
  "Адрес репозитория: https://github.com/owner/repo.git или ssh://git@host/owner/repo.git (без пароля и токена)";

export function branchError(branch: string) {
  return branch && branchInvalid.test(branch)
    ? "Ветка содержит недопустимые символы"
    : "";
}

function collect(entries: [string, string][]) {
  return Object.fromEntries(entries.filter(([, message]) => message));
}

export function validateProjectFields(values: URLSearchParams) {
  return collect([
    ["name", projectNameError(trimmed(values, "name"))],
    [
      "repository_url",
      repositoryURLError(trimmed(values, "repository_url"), false),
    ],
    ["branch", branchError(trimmed(values, "branch"))],
  ]);
}
export function validateProjectName(values: URLSearchParams) {
  return collect([["name", projectNameError(trimmed(values, "name"))]]);
}
export function validateRepository(values: URLSearchParams) {
  return collect([
    ["url", repositoryURLError(trimmed(values, "url"))],
    ["branch", branchError(trimmed(values, "branch"))],
  ]);
}
export function validateMember(values: URLSearchParams) {
  return collect([
    ["identity_id", memberIDError(trimmed(values, "identity_id"))],
  ]);
}
