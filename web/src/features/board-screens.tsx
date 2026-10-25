"use client";
import type { MutationResult, PageData } from "@/lib/types";
import { OverviewScreen } from "./screens/overview";
import { TasksScreen } from "./screens/tasks";
import { TaskDetail } from "./screens/task-detail";
import { AgentDetail, AgentsScreen } from "./screens/agents";
import { ActivityScreen, HistoryScreen } from "./screens/organization";
import { ProjectsScreen, SettingsScreen } from "./screens/project-settings";
import { StatsScreen } from "./screens/stats";
import { TeamScreen } from "./screens/team";
import { DesignsScreen } from "./screens/designs";
import { SessionControls } from "./mutations/SessionControls";
import { ProfilesScreen } from "./profiles/profiles-screen";
import { ProfileDetail } from "./profiles/profile-detail";
import { ManagedScreen } from "./managed/managed-screen";
import { AutomationScreen } from "./automation/automation-screen";
import { ManagedDetail } from "./managed/managed-detail";
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
        <div className="notice">
          Срок рабочей сессии истёк. Продолжите её, чтобы снова видеть проект и
          выполнять действия.
        </div>
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
