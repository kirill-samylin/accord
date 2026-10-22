import {
  profileOption,
  type ManagedHost,
  type ManagedRun,
  type StartDraft,
} from "./managed-agents.ts";
import type { ProfileSummary } from "./agent-profiles.ts";
import type { PageData } from "./types.ts";
export const project = `p_${"a".repeat(32)}`;
export const other = `p_${"b".repeat(32)}`;
export const now = Date.parse("2026-10-22T09:00:00Z");
export const observedAt = new Date(now).toISOString();
export function host(): ManagedHost {
  return {
    id: "mh_test",
    owner_principal_id: "owner",
    name: "Computer",
    version: 1,
    state: "active",
    controllers: [],
    workspaces: ["gui"],
    providers: ["codex"],
    last_seen_at: new Date(now - 1000).toISOString(),
    lease_expires_at: new Date(now + 20000).toISOString(),
    worker_epoch: 1,
    can_control: true,
  };
}
export function run(): ManagedRun {
  return {
    id: "mr_test",
    host_id: "mh_test",
    version: 2,
    provider: "codex",
    workspace: "gui",
    prompt: "Work",
    created_by: "owner",
    state: "running",
    generation: 1,
    provider_session_id: "native-exact-session",
    created_at: new Date(now).toISOString(),
    updated_at: new Date(now).toISOString(),
    output: [],
    output_truncated: false,
  };
}
export function profile(): ProfileSummary {
  return {
    id: `ap_${"c".repeat(32)}`,
    version: 1,
    name: "Developer",
    description: "",
    role: "developer",
    provider: "openai",
    model: "",
    state: "active",
    created_at: "",
    updated_at: "",
    created_by: "",
    updated_by: "",
  };
}
export function page(): PageData {
  return {
    name: "managed-run",
    view: "managed-run",
    title: "Managed",
    csrf: "csrf",
    actions: {
      "managed:command": "issued-command",
      "managed:start": "issued-start",
    },
    principal: { owner: "Owner", machine: "Mac" },
    session: {
      session_id: "human",
      role: "human",
      role_generation: 1,
      expires_at: "",
      project: "GUI",
      session_ttl_seconds: 60,
      task_ttl_seconds: 60,
      heartbeat_every_seconds: 10,
      queue: { ready: 0, waiting: 0, in_progress: 0 },
    },
    can_continue: false,
    can_edit: true,
    draft: {},
    managed: { observed_at: observedAt, hosts: [host()], runs: [run()] },
    managed_run: { run: run(), commands: [] },
    project: {
      id: project,
      name: "GUI",
      state: "active",
      role: "editor",
      version: 1,
      repositories: [],
      can_write: true,
      can_manage: false,
      can_archive: false,
    },
  };
}
export function draft(): StartDraft {
  return {
    host_id: "mh_test",
    workspace: "gui",
    provider: "codex",
    profile: profileOption(profile()),
    prompt: "Работа",
    task_key: " T-12 ",
  };
}
