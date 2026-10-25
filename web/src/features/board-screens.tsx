"use client";
import type { MutationResult, PageData } from "@/lib/types";
import { OverviewScreen } from "./screens/overview";
import { TasksScreen } from "./screens/tasks";
import { TaskDetail } from "./screens/task-detail";
import dynamic from "next/dynamic";
import { SessionControls } from "./mutations/SessionControls";
// Overview, tasks and task details are the hot path and stay in the main
// chunk; every other screen loads on first visit.
function ScreenLoading() {
  return (
    <div className="page" role="status" aria-label="Загрузка раздела">
      <div className="skeleton" />
      <div className="skeleton" />
      <div className="skeleton" />
    </div>
  );
}
const AgentsScreen = dynamic(
  () => import("./screens/agents").then((m) => m.AgentsScreen),
  { loading: ScreenLoading },
);
const AgentDetail = dynamic(
  () => import("./screens/agents").then((m) => m.AgentDetail),
  { loading: ScreenLoading },
);
const ActivityScreen = dynamic(
  () => import("./screens/organization").then((m) => m.ActivityScreen),
  { loading: ScreenLoading },
);
const HistoryScreen = dynamic(
  () => import("./screens/organization").then((m) => m.HistoryScreen),
  { loading: ScreenLoading },
);
const ProjectsScreen = dynamic(
  () => import("./screens/project-settings").then((m) => m.ProjectsScreen),
  { loading: ScreenLoading },
);
const SettingsScreen = dynamic(
  () => import("./screens/project-settings").then((m) => m.SettingsScreen),
  { loading: ScreenLoading },
);
const StatsScreen = dynamic(
  () => import("./screens/stats").then((m) => m.StatsScreen),
  { loading: ScreenLoading },
);
const TeamScreen = dynamic(
  () => import("./screens/team").then((m) => m.TeamScreen),
  { loading: ScreenLoading },
);
const DesignsScreen = dynamic(
  () => import("./screens/designs").then((m) => m.DesignsScreen),
  { loading: ScreenLoading },
);
const ProfilesScreen = dynamic(
  () => import("./profiles/profiles-screen").then((m) => m.ProfilesScreen),
  { loading: ScreenLoading },
);
const ProfileDetail = dynamic(
  () => import("./profiles/profile-detail").then((m) => m.ProfileDetail),
  { loading: ScreenLoading },
);
const ManagedScreen = dynamic(
  () => import("./managed/managed-screen").then((m) => m.ManagedScreen),
  { loading: ScreenLoading },
);
const AutomationScreen = dynamic(
  () =>
    import("./automation/automation-screen").then((m) => m.AutomationScreen),
  { loading: ScreenLoading },
);
const ManagedDetail = dynamic(
  () => import("./managed/managed-detail").then((m) => m.ManagedDetail),
  { loading: ScreenLoading },
);
export function BoardScreen({
  page,
  refresh,
  onCreate,
  onSuccess,
}: {
  page: PageData;
  refresh: () => void;
  onCreate: () => void;
  onSuccess: (result: MutationResult) => void;
}) {
  if (page.can_continue)
    return (
      <div className="page">
        <h1>Продолжить работу</h1>
        <SessionControls page={page} onSuccess={onSuccess} />
      </div>
    );
  switch (page.view || page.name) {
    case "automation":
    case "workflow":
    case "automation-policy":
      return (
        <AutomationScreen
          key={
            page.workflow?.workflow.id ??
            page.automation_policy?.policy.id ??
            "automation"
          }
          page={page}
          refresh={refresh}
          onSuccess={onSuccess}
        />
      );
    case "managed":
      return (
        <ManagedScreen page={page} refresh={refresh} onSuccess={onSuccess} />
      );
    case "managed-run":
      return (
        <ManagedDetail
          key={page.managed_run?.run.id}
          page={page}
          onSuccess={onSuccess}
        />
      );
    case "profiles":
      return (
        <ProfilesScreen page={page} refresh={refresh} onSuccess={onSuccess} />
      );
    case "profile":
      return (
        <ProfileDetail
          key={page.profile?.id}
          page={page}
          onSuccess={onSuccess}
        />
      );
    case "task":
      return (
        <TaskDetail key={page.task?.key} page={page} onSuccess={onSuccess} />
      );
    case "tasks":
    case "questions":
      return (
        <TasksScreen
          key={page.view}
          page={page}
          onCreate={onCreate}
          onSuccess={onSuccess}
        />
      );
    case "team":
      return <TeamScreen page={page} refresh={refresh} onSuccess={onSuccess} />;
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
      return <DesignsScreen page={page} onSuccess={onSuccess} />;
    case "history":
      return <HistoryScreen page={page} />;
    case "settings":
      return <SettingsScreen page={page} onSuccess={onSuccess} />;
    default:
      return (
        <OverviewScreen
          page={page}
          onSessionExpired={refresh}
          onCreate={onCreate}
        />
      );
  }
}
