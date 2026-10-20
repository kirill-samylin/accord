import { withReadDeadline } from "./read-deadline.ts";
import type {
  CatalogMutationResult,
  MutationResult,
  PageData,
} from "./types.ts";
import type { CatalogPage } from "./projects.ts";
import {
  appToWebRoute,
  expectedProjectID,
  projectRoute,
  routeScope,
  webToAppRoute,
} from "./project-routes.ts";
export const webToApp = webToAppRoute;
export const appToWeb = appToWebRoute;
function unauthorized(
  path: string,
  signal?: AbortSignal,
  originScope?: string,
) {
  if (typeof window === "undefined" || signal?.aborted) return;
  const event = new Event("accord:unauthorized");
  Object.assign(event, {
    projectScope:
      originScope ??
      (path.startsWith("/web/catalog/")
        ? "catalog"
        : routeScope(path.replace(/^\/web/, "/app"))),
  });
  window.dispatchEvent(event);
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
export async function readPage<T extends PageData | CatalogPage = PageData>(
  path: string,
  signal?: AbortSignal,
  originScope?: string,
): Promise<T> {
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
      unauthorized(path, signal, originScope);
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
    signal.throwIfAborted();
    const expected = expectedProjectID(path);
    if (expected && result.page.project?.id !== expected)
      throw new APIError(409, {
        ok: false,
        error: {
          code: "PROJECT_SCOPE_MISMATCH",
          message: "Ответ относится к другому проекту",
        },
      });
    const page =
      "session" in result.page && result.page.project?.can_write === false
        ? {
            ...result.page,
            can_edit: false,
            can_update: false,
            can_manage_profiles: false,
          }
        : result.page;
    return page as T;
  });
}
export async function submitAction<
  T extends MutationResult | CatalogMutationResult = MutationResult,
>(
  path: string,
  body: FormData | URLSearchParams,
  signal?: AbortSignal,
  originScope?: string,
): Promise<T> {
  if (/^\/web\/p\/[^/]+\/login$/.test(path))
    throw new Error("Вход выполняется через общий адрес");
  if (
    !/^\/web\/(?:p\/p_[a-f0-9]{32}\/)?(?:login|logout|heartbeat|session\/continue|tasks(?:\/[A-Za-z0-9_-]+\/(?:claim|answer|release|update))?|profiles(?:\/ap_[a-f0-9]{32}\/(?:update|archive))?|designs)$/.test(
      path,
    ) &&
    !/^\/web\/catalog\/(?:switch|projects(?:\/p_[a-f0-9]{32}\/(?:update|members|repositories(?:\/[A-Za-z0-9_-]+)?))?)$/.test(
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
  const expected = path.startsWith("/web/p/") ? routeScope(path) : undefined;
  if (
    result.page &&
    expected &&
    (result.page.project?.id !== undefined || result.ok) &&
    result.page.project?.id !== expected
  )
    throw new APIError(409, {
      ok: false,
      retry_same_action: true,
      error: {
        code: "PROJECT_SCOPE_MISMATCH",
        message:
          "Ответ относится к другому проекту. Проверьте исходную отправку.",
      },
    });
  if (expected && result.page && !result.page.project && !result.ok) {
    // A definitive rejection can omit board metadata. Retain its form errors
    // and capabilities, never merge an unqualified page into the board view.
    result = {
      ...result,
      field_errors: result.field_errors ?? result.page.field_errors,
      actions: result.actions ?? result.page.actions,
      page: undefined,
    };
  }
  if (result.redirect) {
    const target = webToApp(result.redirect);
    if (
      expected &&
      target !== "/app/login" &&
      projectRoute(target)?.id !== expected
    )
      throw new APIError(409, {
        ok: false,
        retry_same_action: true,
        error: {
          code: "PROJECT_SCOPE_MISMATCH",
          message:
            "Недопустимый переход в другой проект. Проверьте исходную отправку.",
        },
      });
  }
  if (response.status === 401 && path !== "/web/login")
    unauthorized(path, signal, originScope);
  return result as T;
}
export const pageQueryKey = (path: string) =>
  ["page", path, routeScope(path)] as const;
