import type { CatalogPage, Project } from "./projects.ts";
import type {
  AgentProfile,
  ProfileRevision,
  ProfileSnapshot,
  ProfileSummary,
} from "./agent-profiles.ts";
export interface Lease {
  session_id: string;
  generation: number;
  expires_at: string;
}
export interface Attempt {
  session_id: string;
  principal_id: string;
  run_id: string;
  machine: string;
  generation: number;
  started_at: string;
  ended_at?: string;
  reason?: string;
  branch?: string;
  commit?: string;
  tokens_spent?: number;
  tokens_delta: number;
}
export interface Evidence {
  commit?: string;
  run_summary?: string;
  artifact?: string[];
  summary?: string;
  answer?: string;
  verified_commit?: string;
  findings?: string;
}
export interface Task {
  key: string;
  type: string;
  title: string;
  description: string;
  role: string;
  state: string;
  priority: number;
  version: number;
  generation: number;
  owner?: string;
  requester?: string;
  ready_at: string;
  created_at?: string;
  first_claim_at?: string;
  first_done_at?: string;
  returns_count: number;
  category?: string;
  complexity?: string;
  external_id?: string;
  epic_key?: string;
  areas?: string[];
  attempts?: Attempt[];
  lease?: Lease;
  evidence: Evidence;
  review?: {
    target_key: string;
    target_commit: string;
    target_generation: number;
    outcome: string;
    author_session: string;
  };
  author_session?: string;
  comments?: { text: string; session_id: string; at: string }[];
  links?: { from: string; to: string; type: string; condition?: string }[];
  blocked: boolean;
}
export type Question = Task;
export interface Note {
  key: string;
  scope_type: string;
  scope_id: string;
  text: string;
  source_task_key: string;
  author_session: string;
  created_at: string;
}
export interface QueueSummary {
  ready: number;
  waiting: number;
  in_progress: number;
}
export interface SessionView {
  profile?: ProfileSnapshot;
  session_id: string;
  role: string;
  role_generation: number;
  expires_at: string;
  project: string;
  session_ttl_seconds: number;
  task_ttl_seconds: number;
  heartbeat_every_seconds: number;
  notes?: Note[];
  rules?: Record<string, string>;
  my_tasks?: Task[];
  queue: QueueSummary;
}
export interface Agent {
  profile?: ProfileSnapshot;
  session_id: string;
  role: string;
  run_id: string;
  machine: string;
  owner: string;
  role_generation: number;
  expires_at: string;
  status: string;
  task?: Task;
  last_event_at?: string;
}
export interface Overview {
  observed_at: string;
  project: string;
  tasks_total: number;
  tasks_created: number;
  tasks_ready: number;
  tasks_in_progress: number;
  tasks_done: number;
  tasks_cancelled: number;
  tasks_blocked: number;
  open_epics: number;
  active_agent_sessions: number;
  active_human_sessions: number;
  leased_tasks: number;
  latest_event_cursor: number;
}
export interface BoardEvent {
  cursor: number;
  type: string;
  task_key?: string;
  session_id?: string;
  reason?: string;
  at: string;
  data?: {
    answer?: string;
    generation?: number;
    author_principal?: string;
    question_title?: string;
    machine?: string;
    role?: string;
    owner?: string;
    epic_key?: string;
    kind?: string;
    old_state?: string;
    new_state?: string;
    incomplete: boolean;
    first_done: boolean;
    rework: boolean;
    tokens_spent?: number;
    tokens_spent_delta?: number;
    tokens_spent_total?: number;
  };
}
export interface Metrics {
  first_completions: number;
  rework_completions: number;
  epic_completions: number;
  returns: number;
  tokens_spent: number;
  attempts: number;
  missing_tokens: number;
  missing_tokens_share: number;
  expired_leases: number;
  rule_warnings: number;
  wall_duration_seconds: number;
  blocked_duration_seconds: number;
  current_queue: QueueSummary;
}
export interface Stats {
  from: string;
  to: string;
  group_by: string;
  totals: Metrics;
  groups: { key: string; metrics: Metrics }[];
  incomplete_data: boolean;
  incomplete_events: number;
}
export interface DesignAsset {
  asset_id: string;
  design_id: string;
  version: number;
  scope: string;
  scope_key: string;
  title: string;
  filename: string;
  media_type: string;
  size: number;
  sha256: string;
  uploader_principal: string;
  uploader_owner: string;
  created_at: string;
  download_url: string;
  preview_url?: string;
}
export interface PageData {
  profiles?: ProfileSummary[];
  profile?: AgentProfile;
  profile_revisions?: ProfileRevision[];
  can_manage_profiles?: boolean;
  project?: Project;
  name: string;
  view: string;
  title: string;
  notice?: string;
  error?: string;
  csrf: string;
  actions: Record<string, string>;
  action_id?: string;
  principal: { owner: string; machine: string };
  session: SessionView;
  can_continue: boolean;
  can_edit: boolean;
  can_update?: boolean;
  draft: Record<string, string>;
  field_errors?: Record<string, string>;
  overview?: Overview;
  agents?: Agent[];
  agent?: Agent;
  activity?: BoardEvent[];
  stats?: Stats;
  observed_at?: string;
  agents_observed_at?: string;
  activity_observed_at?: string;
  task?: Task;
  tasks?: Task[];
  epics?: Task[];
  questions?: Task[];
  my_questions?: Task[];
  legacy_decisions?: Task[];
  designs?: DesignAsset[];
  decisions?: BoardEvent[];
  next_cursor?: string;
  epics_next_cursor?: string;
  query?: string;
  agent_state?: string;
  epic_key?: string;
  role?: string;
  state?: string;
  roles?: string[];
  states?: string[];
  categories?: string[];
  complexities?: string[];
}
export interface MutationResult {
  ok: boolean;
  redirect?: string;
  error?: { code: string; message: string };
  page?: PageData;
  draft?: Record<string, string>;
  field_errors?: Record<string, string>;
  action_id?: string;
  actions?: Record<string, string>;
  retry_same_action?: boolean;
  session?: SessionView;
  next_action_id?: string;
  project?: Project;
  target?: string;
}
export interface CatalogMutationResult extends Omit<MutationResult, "page"> {
  page?: CatalogPage;
}
export interface DashboardData {
  overview: Overview;
  agents: { sessions: Agent[]; observed_at: string };
  activity: { events: BoardEvent[]; observed_at: string };
}
