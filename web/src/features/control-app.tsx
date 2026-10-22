"use client";
import { Suspense, useEffect, useState } from "react";
import dynamic from "next/dynamic";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import {
  QueryClient,
  QueryClientProvider,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { APIError, pageQueryKey, readPage, webToApp } from "@/lib/api";
import { managedPollInterval } from "@/lib/managed-agents";
import {
  projectRoute,
  routeScope,
  safeLoginReturn,
} from "@/lib/project-routes";
import { isCatalogPage, type CatalogPage } from "@/lib/projects";
import type { MutationResult, PageData } from "@/lib/types";
import { Shell } from "./shell";
import { BoardScreen } from "./board-screens";
import { useRouteHeadingFocus } from "./hooks/use-route-heading-focus";
import { LoginForm } from "./mutations/LoginForm";
import { CreateTaskDialog } from "./mutations/CreateTaskDialog";
import { ProjectProvider, useProjectScope } from "./projects/project-context";
import { ProjectOpening } from "./projects/project-opening";
const CatalogWorkspace = dynamic(() => import("./projects/catalog-workspace"), {
  loading: () => <Loading />,
});
export function ControlApp() {
  return (
    <Suspense fallback={<Loading />}>
      <ScopedApp />
    </Suspense>
  );
}
function ScopedApp() {
  const path = usePathname();
  return <QueryBoundary key={routeScope(path)} />;
}
function QueryBoundary() {
  const [client] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            retry: false,
            staleTime: 0,
            gcTime: 0,
            refetchOnWindowFocus: false,
            refetchOnReconnect: false,
          },
        },
      }),
  );
  useEffect(
    () => () => {
      void client.cancelQueries();
      client.clear();
    },
    [client],
  );
  return (
    <QueryClientProvider client={client}>
      <ProjectProvider>
        <ControlContent />
      </ProjectProvider>
    </QueryClientProvider>
  );
}
function Loading() {
  return (
    <div className="app-loading" role="status">
      <div className="skeleton" />
      <div className="skeleton" />
      <span>Загрузка проекта…</span>
      <noscript>
        <a href="/web/">Открыть базовый интерфейс</a>
      </noscript>
    </div>
  );
}
function ControlContent() {
  const path = usePathname();
  const search = useSearchParams();
  const router = useRouter();
  const scope = useProjectScope();
  const client = useQueryClient();
  const location = `${path}${search.size ? `?${search}` : ""}`;
  const [create, setCreate] = useState(false);
  const result = useQuery({
    queryKey: pageQueryKey(location),
    queryFn: ({ signal }) => readPage<PageData | CatalogPage>(location, signal),
    refetchInterval: (query) =>
      managedPollInterval(
        path,
        scope.locked,
        query.state.data && !isCatalogPage(query.state.data)
          ? query.state.data.can_continue
          : true,
        query.state.error instanceof APIError
          ? query.state.error.status
          : undefined,
        query.state.errorUpdateCount,
      ),
    refetchIntervalInBackground: false,
    placeholderData: (previous, previousQuery) =>
      String(previousQuery?.queryKey[1] ?? "").split("?")[0] === path
        ? previous
        : undefined,
  });
  useRouteHeadingFocus(
    path,
    result.isSuccess &&
      !result.isPlaceholderData &&
      result.data?.name !== "login" &&
      path !== "/app/login",
  );
  const createOptions = useQuery({
    queryKey: ["create-options", scope.key],
    queryFn: ({ signal }) =>
      readPage(scope.app("/app/tasks?include_closed=true"), signal),
    enabled:
      create &&
      Boolean(result.data) &&
      !isCatalogPage(result.data) &&
      !(result.data as PageData)?.can_continue &&
      (result.data as PageData)?.project?.can_write !== false,
  });
  function unauthorized(event?: Event) {
    if (event && "projectScope" in event && event.projectScope !== scope.key)
      return;
    setCreate(false);
    client.clear();
    const target = safeLoginReturn(location);
    router.replace(
      `/app/login${target ? `?next=${encodeURIComponent(target)}` : ""}`,
    );
  }
  useEffect(() => {
    window.addEventListener("accord:unauthorized", unauthorized);
    return () =>
      window.removeEventListener("accord:unauthorized", unauthorized);
  });
  useEffect(() => {
    if (
      result.error instanceof APIError &&
      result.error.status === 401 &&
      path !== "/app/login"
    )
      unauthorized();
  }, [result.error, path]);
  async function success(action: MutationResult) {
    if (!action.ok) return;
    setCreate(false);
    let target = action.redirect ? webToApp(action.redirect) : location;
    if (path === "/app/login")
      target = safeLoginReturn(search.get("next")) ?? target;
    if (path === "/app/login" || target === "/app/login") {
      client.clear();
      router.replace(target);
      return;
    }
    await client.invalidateQueries();
    if (target !== location) router.push(target);
  }
  if (result.isPending) return <Loading />;
  if (
    result.error instanceof APIError &&
    result.error.result.error?.code === "PROJECT_SESSION_REQUIRED"
  ) {
    const expected = projectRoute(path)?.id;
    if (expected && result.error.result.project?.id === expected)
      return (
        <ProjectOpening
          project={result.error.result.project}
          target={location}
        />
      );
  }
  if (
    !result.data ||
    (result.error instanceof APIError &&
      [403, 404].includes(result.error.status))
  )
    return (
      <main className="welcome">
        <div className="project-mark">A</div>
        <h1>
          {result.error instanceof APIError && result.error.status === 404
            ? "Страница не найдена"
            : "Не удалось открыть страницу"}
        </h1>
        <p className="muted">
          {result.error instanceof Error
            ? result.error.message
            : "Проверьте соединение и повторите попытку."}
        </p>
        <button className="button" onClick={() => result.refetch()}>
          Повторить
        </button>
        <a href="/app/projects">Все проекты</a>
      </main>
    );
  if (isCatalogPage(result.data))
    return <CatalogWorkspace page={result.data} />;
  const page = result.data;
  if (path === "/app/login" || page.name === "login")
    return (
      <main className="auth-page">
        <section className="auth-card">
          <div className="auth-brand">
            <span className="project-mark">A</span>Accord
          </div>
          <h1>Добро пожаловать</h1>
          <p className="muted">Войдите, чтобы продолжить работу с проектом.</p>
          <LoginForm page={page} onSuccess={success} />
          <p className="auth-footer">
            <a href="/web/login">Базовый интерфейс входа</a>
          </p>
        </section>
      </main>
    );
  return (
    <Shell page={page} onCreate={() => setCreate(true)} onSuccess={success}>
      {page.notice ? (
        <div className="page-notice notice" role="status">
          {page.notice}
        </div>
      ) : null}
      {result.isError ? (
        <div className="page-notice notice error-notice" role="alert">
          Не удалось обновить страницу. Последние данные и ваш черновик
          сохранены.{" "}
          <button
            className="button button-secondary button-sm"
            onClick={() => result.refetch()}
          >
            Повторить
          </button>
        </div>
      ) : null}
      {result.isFetching ? (
        <div className="refresh-indicator muted" role="status">
          Обновляем данные…
        </div>
      ) : null}
      <BoardScreen
        page={page}
        refresh={() => void result.refetch()}
        onCreate={() => setCreate(true)}
        onSuccess={success}
      />
      {create ? (
        createOptions.isPending ? (
          <div className="loading-toast" role="status">
            Готовим форму…
          </div>
        ) : createOptions.isError ? (
          <div className="loading-toast notice error-notice" role="alert">
            Не удалось загрузить форму{" "}
            <button onClick={() => createOptions.refetch()}>Повторить</button>
            <button onClick={() => setCreate(false)}>Закрыть</button>
          </div>
        ) : (
          <CreateTaskDialog
            page={createOptions.data ?? page}
            onSuccess={success}
            open={create}
            onOpenChange={setCreate}
          />
        )
      ) : null}
    </Shell>
  );
}
