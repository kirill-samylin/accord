import { withReadDeadline } from "./read-deadline.ts";
import type { MutationResult, PageData } from "./types.ts";
const screen =
  /^(?:\/(?:overview|tasks(?:\/[A-Za-z0-9_-]+)?|questions|agents(?:\/[A-Za-z0-9_-]+)?|projects|designs|history|activity|stats|settings|login))?\/?$/;
export function webToApp(path: string): string {
  if (!path.startsWith("/web") || path.includes("\\") || /[\r\n]/.test(path))
    throw new Error("Недопустимый адрес перехода");
  const url = new URL(path, "https://accord.invalid");
  const tail = url.pathname.slice(4);
  if (url.origin !== "https://accord.invalid" || !screen.test(tail))
    throw new Error("Недопустимый адрес перехода");
  return `/app${tail === "/" ? "" : tail}${url.search}${url.hash}`;
}
export function appToWeb(path: string): string {
  if (!path.startsWith("/app") || path.includes("\\"))
    throw new Error("Недопустимый адрес страницы");
  const url = new URL(path, "https://accord.invalid");
  const tail = url.pathname.slice(4);
  if (url.origin !== "https://accord.invalid" || !screen.test(tail))
    throw new Error("Недопустимый адрес страницы");
  const canonical = tail.replace(/\/$/, "");
  return `/web${canonical || "/"}${url.search}`;
}
export class APIError extends Error {
  status: number;
  result: MutationResult;
  constructor(status: number, result: MutationResult) {
    super(result.error?.message ?? "Сервер вернул неизвестный ответ");
    this.name = "APIError";
    this.status = status;
    this.result = result;
  }
}
export async function readPage(
  path: string,
  signal?: AbortSignal,
): Promise<PageData> {
  return withReadDeadline(signal, async (signal) => {
    const response = await fetch(appToWeb(path), {
      headers: { Accept: "application/json" },
      cache: "no-store",
      credentials: "same-origin",
      redirect: "error",
      signal,
    });
    if (
      response.status === 401 &&
      path !== "/app/login" &&
      typeof window !== "undefined"
    )
      window.dispatchEvent(new Event("accord:unauthorized"));
    let result: MutationResult;
    try {
      result = await response.json();
    } catch {
      throw new APIError(response.status, {
        ok: false,
        error: {
          code: "INVALID_RESPONSE",
          message: "Не удалось прочитать ответ сервера",
        },
      });
    }
    if (!response.ok || !result.ok || !result.page)
      throw new APIError(response.status, result);
    return result.page;
  });
}
export async function submitAction(
  path: string,
  body: FormData | URLSearchParams,
  signal?: AbortSignal,
): Promise<MutationResult> {
  if (
    !/^\/web\/(?:login|logout|heartbeat|session\/continue|tasks(?:\/[A-Za-z0-9_-]+\/(?:claim|answer|release|update))?|designs)$/.test(
      path,
    )
  )
    throw new Error("Недопустимое действие");
  const response = await fetch(path, {
    method: "POST",
    body,
    signal,
    headers: { Accept: "application/json" },
    cache: "no-store",
    credentials: "same-origin",
    redirect: "error",
  });
  let result: MutationResult;
  try {
    result = await response.json();
  } catch {
    throw new APIError(response.status, {
      ok: false,
      retry_same_action: true,
      error: {
        code: "UNKNOWN_RESULT",
        message: "Результат операции неизвестен. Повторите тот же запрос.",
      },
    });
  }
  if (result.redirect) webToApp(result.redirect);
  if (response.status === 401 && path !== "/web/login")
    window.dispatchEvent(new Event("accord:unauthorized"));
  return result;
}
export const pageQueryKey = (path: string) => ["page", path] as const;
