import Link from "next/link";
import {
  ArrowRight,
  Bot,
  Circle,
  CircleCheck,
  CircleDashed,
  CircleDot,
  CircleHelp,
  CircleX,
  Inbox,
  LockKeyhole,
} from "lucide-react";
import type { ReactNode } from "react";
import type { Agent, BoardEvent, Task } from "@/lib/types";
import { date, initial, label } from "@/lib/format";
import { cn } from "@/lib/utils";
export function Status({
  value,
  blocked = false,
}: {
  value: string;
  blocked?: boolean;
}) {
  const state = blocked ? "blocked" : value;
  const Icon =
    (
      {
        created: CircleDashed,
        ready: Circle,
        in_progress: CircleDot,
        done: CircleCheck,
        cancelled: CircleX,
        active: CircleDot,
        expired: CircleDashed,
        credential_invalid: LockKeyhole,
        blocked: CircleHelp,
      } as Record<string, typeof Circle>
    )[state] ?? Circle;
  return (
    <span className={cn("status", `status-${state}`)}>
      <Icon size={14} />
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
        <Link href={href}>
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
      <Link className="mono task-key" href={`/app/tasks/${task.key}`}>
        {task.key}
      </Link>
      <Link
        className="task-title truncate"
        title={task.title}
        href={`/app/tasks/${task.key}`}
      >
        {task.title}
      </Link>
      <span className="task-role muted truncate">{label(task.role)}</span>
      {task.owner ? (
        <span className="task-owner truncate muted" title={task.owner}>
          {task.owner}
        </span>
      ) : (
        <span className="task-owner muted">Не назначена</span>
      )}
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
    <Link className="entity-row" href={`/app/agents/${agent.session_id}`}>
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
                <Link className="mono" href={`/app/tasks/${event.task_key}`}>
                  {event.task_key}
                </Link>
              ) : null}
            </div>
            {decisions && event.data?.answer !== undefined ? (
              <p className="answer-text">{event.data.answer}</p>
            ) : null}
            {event.reason ? <p className="muted">{event.reason}</p> : null}
            <div className="event-meta muted">
              {event.data?.owner ??
                (event.session_id
                  ? `Сессия ${event.session_id.slice(0, 12)}`
                  : "Автор не записан")}
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
      <div className="metric-value">{value}</div>
      {caption ? <div className="metric-caption muted">{caption}</div> : null}
    </>
  );
  return href ? (
    <Link className="metric" href={href}>
      {body}
    </Link>
  ) : (
    <div className="metric">{body}</div>
  );
}
