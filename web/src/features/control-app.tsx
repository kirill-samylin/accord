"use client";
import { Suspense, useEffect, useState } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import {
  QueryClient,
  QueryClientProvider,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { APIError, pageQueryKey, readPage, webToApp } from "@/lib/api";
import type { MutationResult, PageData } from "@/lib/types";
import { Shell } from "./shell";
import { LoginForm } from "./mutations/LoginForm";
import { CreateTaskDialog } from "./mutations/CreateTaskDialog";
import { SessionControls } from "./mutations/SessionControls";
import { OverviewScreen } from "./screens/overview";
import { TasksScreen } from "./screens/tasks";
import { TaskDetail } from "./screens/task-detail";
import { AgentDetail, AgentsScreen } from "./screens/agents";
import { ActivityScreen, HistoryScreen } from "./screens/organization";
import { ProjectsScreen, SettingsScreen } from "./screens/project-settings";
import { StatsScreen } from "./screens/stats";
import { DesignsScreen } from "./screens/designs";
export function ControlApp() {
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
  return (
    <QueryClientProvider client={client}>
      <Suspense fallback={<Loading />}>
        <ControlContent />
      </Suspense>
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
  const client = useQueryClient();
  const location = `${path}${search.size ? `?${search}` : ""}`;
  const [create, setCreate] = useState(false);
  const result = useQuery({
    queryKey: pageQueryKey(location),
    queryFn: ({ signal }) => readPage(location, signal),
    // Keep local tabs and drafts during filter reads within the same screen.
    placeholderData: (previous, previousQuery) =>
      String(previousQuery?.queryKey[1] ?? "").split("?")[0] === path
        ? previous
        : undefined,
  });
  const createOptions = useQuery({
    queryKey: ["create-options"],
    queryFn: ({ signal }) => readPage("/app/tasks?include_closed=true", signal),
    enabled: create && Boolean(result.data) && !result.data?.can_continue,
  });
  function unauthorized() {
    setCreate(false);
    client.clear();
    router.replace("/app/login");
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
  }, [result.error, path]); // Revocation clears every person's cached body.
  async function success(action: MutationResult) {
    if (!action.ok) return;
    setCreate(false);
    const target = action.redirect ? webToApp(action.redirect) : location;
    // Login/logout can change identity. Clearing also removes old capabilities and private drafts.
    if (path === "/app/login" || target === "/app/login") {
      client.clear();
      router.replace(target);
      return;
    }
    await client.invalidateQueries();
    if (target !== location) router.push(target);
  }
  if (result.isPending) return <Loading />;
  if (!result.data)
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
        <a href="/web/">Базовый интерфейс</a>
      </main>
    );
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
  function screen() {
    if (page.can_continue)
      return (
        <div className="page">
          <h1>Продолжить работу</h1>
          <div className="notice">
            Срок рабочей сессии истёк. Продолжите её, чтобы снова видеть проект
            и выполнять действия.
          </div>
          <SessionControls page={page} onSuccess={success} />
        </div>
      );
    const refresh = () => {
      void result.refetch();
    };
    switch (page.view || page.name) {
      case "task":
        return (
          <TaskDetail key={page.task?.key} page={page} onSuccess={success} />
        );
      case "tasks":
      case "questions":
        return (
          <TasksScreen
            key={page.view}
            page={page}
            onCreate={() => setCreate(true)}
            onSuccess={success}
          />
        );
      case "agents":
        return <AgentsScreen page={page} refresh={refresh} />;
      case "agent":
        return <AgentDetail key={page.agent?.session_id} page={page} />;
      case "projects":
        return <ProjectsScreen page={page} />;
      case "activity":
        return <ActivityScreen page={page} refresh={refresh} />;
      case "stats":
        return <StatsScreen page={page} />;
      case "designs":
        return <DesignsScreen page={page} onSuccess={success} />;
      case "history":
        return <HistoryScreen page={page} />;
      case "settings":
        return <SettingsScreen page={page} onSuccess={success} />;
      default:
        return (
          <OverviewScreen
            page={page}
            onSessionExpired={() => {
              void result.refetch();
            }}
          />
        );
    }
  }
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
      {screen()}
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
