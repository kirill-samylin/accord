import { ProjectLink as Link } from "@/components/project-link";
import {
  ArrowRight,
  Ban,
  Bot,
  Circle,
  CircleCheck,
  CircleDashed,
  CircleDot,
  Hourglass,
  Inbox,
  LockKeyhole,
  PencilLine,
  Zap,
} from "lucide-react";
import type { ReactNode } from "react";
import type { Agent, BoardEvent, Task } from "@/lib/types";
import { date, initial, label, statusHint } from "@/lib/format";
import { AssignmentLabel } from "@/features/coordination/assignment-label";
import { PersonChip } from "@/components/people";
import { cn } from "@/lib/utils";
import { CountUp } from "@/components/motion";
const stateIcons: Record<string, typeof Circle> = {
  created: PencilLine,
  ready: Hourglass,
  in_progress: Zap,
  done: CircleCheck,
  cancelled: Ban,
  canceled: Ban,
  active: CircleDot,
  expired: CircleDashed,
  credential_invalid: LockKeyhole,
  blocked: LockKeyhole,
};
export function statusIcon(state: string) {
  return stateIcons[state] ?? Circle;
}
export function Status({
  value,
  blocked = false,
}: {
  value: string;
  blocked?: boolean;
}) {
  const state = blocked ? "blocked" : value;
  const Icon = statusIcon(state);
  const hint = statusHint(state);
  return (
    <span
      className={cn("status", `status-${state}`)}
      title={hint ? `${label(state)} — ${hint}` : undefined}
    >
      <Icon size={14} aria-hidden />
      <span>{label(state)}</span>
    </span>
  );
}
export function Avatar({
  name,
  role,
  agent = false,
}: {
  name: string;
  role?: string;
  agent?: boolean;
}) {
  return (
    <span
      aria-hidden
      className={cn("avatar", agent && "avatar-agent", role && `agent-${role}`)}
    >
      {agent ? <Bot size={16} /> : initial(name)}
    </span>
  );
}
export function EmptyState({
  title = "Здесь пока ничего нет",
  description,
  action,
  icon,
}: {
  title?: string;
  description?: string;
  action?: ReactNode;
  icon?: ReactNode;
}) {
  return (
    <div className="empty-state">
      {icon ?? <Inbox size={24} />}
      <h3>{title}</h3>
      {description ? <p>{description}</p> : null}
      {action}
    </div>
  );
}
export function SectionHeading({
  title,
  href,
  action,
}: {
  title: string;
  href?: string;
  action?: ReactNode;
}) {
  return (
    <div className="section-heading">
      <h2>{title}</h2>
      {href ? (
        <Link prefetch={false} href={href}>
          Все <ArrowRight size={13} />
        </Link>
      ) : (
        action
      )}
    </div>
  );
}
export function PageHeading({
  title,
  description,
  actions,
}: {
  title: string;
  description?: string;
  actions?: ReactNode;
}) {
  return (
    <div className="page-heading">
      <div className="page-heading-copy">
        <h1>{title}</h1>
        {description ? <p>{description}</p> : null}
      </div>
      {actions ? <div className="heading-actions">{actions}</div> : null}
    </div>
  );
}
export function TaskRow({
  task,
  onPeek,
}: {
  task: Task;
  onPeek?: (task: Task) => void;
}) {
  return (
    <div className="task-row">
      <span
        className="task-row-status"
        title={label(task.blocked ? "blocked" : task.state)}
      >
        <Status value={task.state} blocked={task.blocked} />
      </span>
      <Link
        prefetch={false}
        className="mono task-key"
        href={`/app/tasks/${task.key}`}
      >
        {task.key}
      </Link>
      <Link
        prefetch={false}
        className="task-title truncate"
        title={task.title}
        href={`/app/tasks/${task.key}`}
      >
        {task.title}
      </Link>
      {task.duplicate_of ? (
        <span className="dup-tag" title={`Дубль задачи ${task.duplicate_of}`}>
          дубль {task.duplicate_of}
        </span>
      ) : null}
      <span className="task-people">
        {task.holder_owner && task.lease ? (
          <PersonChip
            owner={task.holder_owner}
            prefix="делает"
            title={
              task.holder_machine
                ? `Агент на ${task.holder_machine}`
                : undefined
            }
          />
        ) : task.created_by ? (
          <PersonChip
            owner={task.created_by}
            prefix="от"
            title="Кто создал задачу"
          />
        ) : null}
      </span>
      <span className="task-role muted truncate">{label(task.role)}</span>
      <span className="task-owner muted">
        <AssignmentLabel task={task} />
      </span>
      {onPeek ? (
        <button
          className="peek-button icon-button"
          aria-label={`Быстрый просмотр ${task.key}`}
          onClick={() => onPeek(task)}
        >
          <ArrowRight size={14} />
        </button>
      ) : null}
    </div>
  );
}
export function AgentRow({ agent }: { agent: Agent }) {
  return (
    <Link
      prefetch={false}
      className="entity-row"
      href={`/app/agents/${agent.session_id}`}
    >
      <Avatar name={agent.owner} role={agent.role} agent />
      <div className="entity-main">
        <strong className="truncate">{agent.owner || label(agent.role)}</strong>
        <span className="muted truncate">
          {label(agent.role)} · {agent.machine}
        </span>
      </div>
      <span className="entity-task mono truncate">
        {agent.task ? `${agent.task.key} · ${agent.task.title}` : "Без задачи"}
      </span>
      <Status value={agent.status} />
      <span className="entity-date muted">{date(agent.expires_at)}</span>
    </Link>
  );
}
export function EventRows({
  events,
  decisions = false,
}: {
  events: BoardEvent[];
  decisions?: boolean;
}) {
  if (!events.length)
    return (
      <EmptyState
        title={decisions ? "Решений пока нет" : "Событий пока нет"}
        description={
          decisions
            ? "Ответы на вопросы появятся здесь после сохранения."
            : "Здесь появится история работы проекта."
        }
      />
    );
  return (
    <div className="event-list">
      {events.map((event) => (
        <div className="event-row" key={event.cursor}>
          <span className="event-dot" />
          <div className="event-main">
            <div className="event-title">
              <span>
                {decisions
                  ? (event.data?.question_title ?? "Ответ на вопрос")
                  : label(event.type)}
              </span>
              {event.task_key ? (
                <Link
                  prefetch={false}
                  className="mono"
                  href={`/app/tasks/${event.task_key}`}
                >
                  {event.task_key}
                </Link>
              ) : null}
            </div>
            {decisions && event.data?.answer !== undefined ? (
              <p className="answer-text">{event.data.answer}</p>
            ) : null}
            {event.reason ? <p className="muted">{event.reason}</p> : null}
            <div className="event-meta muted">
              {event.data?.owner ? (
                <PersonChip owner={event.data.owner} />
              ) : event.session_id ? (
                `Сессия ${event.session_id.slice(0, 12)}`
              ) : (
                "Автор не записан"
              )}
              {event.data?.duplicate_of
                ? ` · дубль ${event.data.duplicate_of}`
                : ""}
              {event.data?.similar?.length
                ? ` · похожие: ${event.data.similar.join(", ")}`
                : ""}
              {event.data?.role ? ` · ${label(event.data.role)}` : ""}
              {event.data?.new_state ? ` · ${label(event.data.new_state)}` : ""}
              {decisions && event.data?.generation
                ? ` · поколение ${event.data.generation}`
                : ""}
            </div>
          </div>
          <time className="event-time muted" dateTime={event.at}>
            {date(event.at)}
          </time>
        </div>
      ))}
    </div>
  );
}
export function Fact({ name, value }: { name: string; value?: ReactNode }) {
  return (
    <div className="fact">
      <dt>{name}</dt>
      <dd>{value ?? "—"}</dd>
    </div>
  );
}
export function Metric({
  title,
  value,
  caption,
  href,
}: {
  title: string;
  value: ReactNode;
  caption?: string;
  href?: string;
}) {
  const body = (
    <>
      <div className="metric-label">{title}</div>
      <div className="metric-value">
        {typeof value === "string" && /^\d[\d\s\u00a0]*$/.test(value) ? (
          <CountUp value={Number(value.replace(/\D/g, ""))} />
        ) : typeof value === "number" ? (
          <CountUp value={value} />
        ) : (
          value
        )}
      </div>
      {caption ? <div className="metric-caption muted">{caption}</div> : null}
    </>
  );
  return href ? (
    <Link prefetch={false} className="metric" href={href}>
      {body}
    </Link>
  ) : (
    <div className="metric">{body}</div>
  );
}
