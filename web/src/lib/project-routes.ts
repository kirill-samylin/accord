import { projectID } from "./projects.ts";

const screens =
  /^(?:\/(?:overview|tasks(?:\/[A-Za-z0-9_-]+)?|questions|agents(?:\/[A-Za-z0-9_-]+)?|profiles(?:\/ap_[a-f0-9]{32})?|projects|designs|history|activity|stats|settings|team|automation(?:\/host-scope|\/workflows\/[A-Za-z0-9_-]+|\/policies\/[A-Za-z0-9_-]+)?|managed(?:\/runs\/[A-Za-z0-9_-]+)?|login))?\/?$/;
const base = "https://accord.invalid";
// The administration screen lives beside the projects catalog: /app/admin is
// served by /web/catalog/admin and shares the "catalog" scope.
const adminApp = /^\/app\/admin\/?$/;
const adminWeb = /^\/web\/catalog\/admin\/?$/;
const catalogApp = /^\/app\/(?:projects|admin)(?:\/|$)/;
function internal(path: string): URL {
  if (
    !path.startsWith("/") ||
    /[\\\r\n]/.test(path) ||
    /(?:^|\/)\.{1,2}(?:\/|$)/.test(path.split(/[?#]/)[0])
  )
    throw new Error("Недопустимый адрес перехода");
  const url = new URL(path, base);
  if (url.origin !== base || /%2f|%5c|%2e/i.test(url.pathname))
    throw new Error("Недопустимый адрес перехода");
  return url;
}
export function projectRoute(path: string) {
  const url = internal(path);
  const match = /^\/(app|web)\/p\/([^/]+)(.*)$/.exec(url.pathname);
  if (!match) return undefined;
  if (
    !projectID.test(match[2]) ||
    !screens.test(match[3]) ||
    /^\/login\/?$/.test(match[3])
  )
    throw new Error("Недопустимый адрес проекта");
  return { id: match[2], tail: match[3], url };
}
export function routeScope(path: string): string {
  const url = internal(path);
  const route = /^\/(?:app|web)\/p\/([^/]+)(?:\/|$)/.exec(url.pathname);
  if (route) {
    if (!projectID.test(route[1])) throw new Error("Недопустимый проект");
    return route[1];
  }
  const detail = /^\/app\/projects\/(p_[a-f0-9]{32})\/?$/.exec(url.pathname);
  if (detail) return `catalog:${detail[1]}`;
  if (
    /^\/app\/projects(?:\/|$)/.test(url.pathname) ||
    adminApp.test(url.pathname)
  )
    return "catalog";
  return "default";
}
export function expectedProjectID(path: string) {
  return (
    projectRoute(path)?.id ??
    /^\/app\/projects\/(p_[a-f0-9]{32})\/?$/.exec(internal(path).pathname)?.[1]
  );
}
export function screenPath(path: string): string {
  const route = projectRoute(path);
  return route ? `/app${route.tail || ""}` : internal(path).pathname;
}
// View state kept only in the app URL (tabs, toggles) so reload and Back keep
// it. The Go page handlers reject unknown query parameters, so these never
// reach /web/.
export const clientOnlyParams = ["tab", "archived"] as const;
export function appToWebRoute(path: string): string {
  const url = internal(path);
  for (const name of clientOnlyParams) url.searchParams.delete(name);
  const route = projectRoute(path);
  if (route) {
    if (!url.pathname.startsWith("/app/"))
      throw new Error("Недопустимая страница");
    return `/web/p/${route.id}${route.tail.replace(/\/$/, "") || "/"}${url.search}`;
  }
  const detail = /^\/app\/projects\/(p_[a-f0-9]{32})\/?$/.exec(url.pathname);
  if (detail) return `/web/catalog/projects/${detail[1]}${url.search}`;
  if (adminApp.test(url.pathname)) return `/web/catalog/admin${url.search}`;
  if (!url.pathname.startsWith("/app") || !screens.test(url.pathname.slice(4)))
    throw new Error("Недопустимый адрес страницы");
  if (url.pathname.replace(/\/$/, "") === "/app/login")
    url.searchParams.delete("next");
  const tail = url.pathname.slice(4).replace(/\/$/, "");
  return `/web${tail || "/"}${url.search}`;
}
export function webToAppRoute(path: string): string {
  const url = internal(path);
  const route = projectRoute(path);
  if (route) {
    if (!url.pathname.startsWith("/web/"))
      throw new Error("Недопустимый переход");
    return `/app/p/${route.id}${route.tail === "/" ? "" : route.tail}${url.search}${url.hash}`;
  }
  const detail = /^\/web\/catalog\/projects\/(p_[a-f0-9]{32})\/?$/.exec(
    url.pathname,
  );
  if (detail) return `/app/projects/${detail[1]}${url.search}${url.hash}`;
  if (adminWeb.test(url.pathname)) return `/app/admin${url.search}${url.hash}`;
  if (!url.pathname.startsWith("/web") || !screens.test(url.pathname.slice(4)))
    throw new Error("Недопустимый адрес перехода");
  const tail = url.pathname.slice(4);
  return `/app${tail === "/" ? "" : tail}${url.search}${url.hash}`;
}
export function projectAppPath(id: string | undefined, path: string): string {
  const url = internal(path);
  const scoped = projectRoute(path);
  if (scoped && id && scoped.id !== id)
    throw new Error("Ссылка относится к другому проекту");
  if (scoped || catalogApp.test(url.pathname)) {
    appToWebRoute(path);
    return path;
  }
  appToWebRoute(path);
  if (!id || url.pathname === "/app/login") return path;
  if (!projectID.test(id)) throw new Error("Недопустимый проект");
  return `/app/p/${id}${url.pathname.slice(4)}${url.search}${url.hash}`;
}
export function projectWebPath(id: string | undefined, path: string): string {
  if (/^\/web\/catalog\//.test(path)) return path;
  const url = internal(path);
  if (url.pathname.startsWith("/web/p/")) {
    const actual = routeScope(path);
    if (id && actual !== id)
      throw new Error("Действие относится к другому проекту");
    return path;
  }
  if (!url.pathname.startsWith("/web/"))
    throw new Error("Недопустимое действие");
  if (!id || url.pathname === "/web/login") return path;
  if (!projectID.test(id)) throw new Error("Недопустимый проект");
  return `/web/p/${id}${url.pathname.slice(4)}${url.search}`;
}
export function safeLoginReturn(value: string | null): string | undefined {
  if (!value) return;
  try {
    appToWebRoute(value);
    if (projectRoute(value) || catalogApp.test(internal(value).pathname))
      return value;
  } catch {
    /* Invalid return URLs never become redirects. */
  }
}
