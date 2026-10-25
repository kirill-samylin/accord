"use client";
import { ProjectLink as Link } from "@/components/project-link";
import { Clock3, Coffee, ListTodo, RefreshCw, Users } from "lucide-react";
import {
  EmptyState,
  PageHeading,
  SectionHeading,
  TaskRow,
} from "@/components/common";
import { LivePill, PersonAvatar, RoleBadge, useNow } from "@/components/people";
import type {
  Agent,
  MutationResult,
  PageData,
  TeamMember,
  TeamView,
} from "@/lib/types";
import { date, label } from "@/lib/format";
import { ago, personClass, since } from "@/lib/people";
import { cn } from "@/lib/utils";
import { useTeam } from "../hooks/use-team";
import { DuplicatePairs } from "../duplicates/duplicate-pairs";

function attemptStart(agent: Agent) {
  const attempts = agent.task?.attempts ?? [];
  const current = attempts.findLast((a) => a.session_id === agent.session_id);
  return current?.started_at;
}

function AgentLine({ agent, now }: { agent: Agent; now: number }) {
  const started = attemptStart(agent);
  return (
    <div
      className={cn(
        "team-agent",
        agent.task ? "team-agent-busy" : "team-agent-idle",
      )}
    >
      <div className="team-agent-head">
        <RoleBadge role={agent.role} name={agent.profile?.name} />
        <span className="muted truncate team-agent-machine">
          {agent.machine}
        </span>
        {agent.task && started ? (
          <span className="team-agent-timer" title={`Начато ${date(started)}`}>
            <Clock3 size={12} aria-hidden />
            {since(started, now)}
          </span>
        ) : null}
      </div>
      {agent.task ? (
        <Link
          prefetch={false}
          className="team-agent-task"
          href={`/app/tasks/${agent.task.key}`}
          title={agent.task.title}
        >
          <span className="mono task-key">{agent.task.key}</span>
          <span className="truncate">{agent.task.title}</span>
        </Link>
      ) : (
        <span className="team-agent-task muted">
          <Coffee size={12} aria-hidden />
          {agent.role === "coordinator"
            ? "Координирует: создаёт и распределяет задачи"
            : `Свободен — ждёт задачу роли «${label(agent.role)}»`}
        </span>
      )}
      {agent.task?.areas?.length ? (
        <div className="team-agent-areas">
          {agent.task.areas.map((area) => (
            <span key={area} className="area-tag mono">
              {area}
            </span>
          ))}
        </div>
      ) : null}
    </div>
  );
}

function PersonCard({
  member,
  now,
  me,
}: {
  member: TeamMember;
  now: number;
  me: boolean;
}) {
  const online = member.human_online || member.agents.length > 0;
  const busy = member.agents.filter((a) => a.task);
  const idle = member.agents.filter((a) => !a.task);
  return (
    <article
      className={cn(
        "person-card",
        personClass(member.owner),
        !online && "person-card-away",
      )}
    >
      <header className="person-card-head">
        <PersonAvatar owner={member.owner} online={online} size="lg" />
        <div className="person-card-name">
          <strong className="truncate">
            {member.owner}
            {me ? <span className="you-tag">вы</span> : null}
          </strong>
          <span className="muted truncate">
            {online
              ? member.machines.length
                ? `в сети · ${member.machines.join(", ")}`
                : "в сети"
              : member.last_event_at
                ? `был активен ${ago(member.last_event_at, now)}`
                : "не в сети"}
          </span>
        </div>
      </header>
      <dl className="person-stats">
        <div title="Задачи, которые сейчас держат агенты этого человека">
          <dt>В работе</dt>
          <dd>{member.in_progress}</dd>
        </div>
        <div title="Создано задач за последние сутки">
          <dt>Создал</dt>
          <dd>{member.created_recent}</dd>
        </div>
        <div title="Завершено задач за последние сутки">
          <dt>Завершил</dt>
          <dd>{member.done_recent}</dd>
        </div>
        <div title="Незавершённые задачи, созданные этим человеком">
          <dt>Открыто</dt>
          <dd>{member.open_requested}</dd>
        </div>
      </dl>
      {member.agents.length ? (
        <div className="team-agents">
          {busy.map((agent) => (
            <AgentLine key={agent.session_id} agent={agent} now={now} />
          ))}
          {idle.map((agent) => (
            <AgentLine key={agent.session_id} agent={agent} now={now} />
          ))}
        </div>
      ) : (
        <p className="team-empty muted">
          {online
            ? "Агенты не подключены — работает в панели"
            : "Нет подключённых агентов"}
        </p>
      )}
    </article>
  );
}

function Summary({ team }: { team: TeamView }) {
  const online = team.members.filter(
    (m) => m.human_online || m.agents.length,
  ).length;
  const agents = team.members
    .flatMap((m) => m.agents)
    .filter((a) => a.task || a.role !== "coordinator");
  const busy = agents.filter((a) => a.task).length;
  const tiles = [
    {
      title: "Людей в сети",
      value: online,
      caption: `всего ${team.members.length}`,
    },
    {
      title: "Агенты в работе",
      value: busy,
      caption: `${agents.length - busy} свободны`,
    },
    {
      title: "Ждут исполнителя",
      value: team.unclaimed_total,
      caption: "готовые задачи без агента",
    },
    {
      title: "Возможные дубли",
      value: team.duplicates_total,
      caption:
        team.duplicate_mode === "strict"
          ? "строгий режим: дубль не создать без причины"
          : team.duplicate_mode === "off"
            ? "проверка выключена"
            : team.duplicates_total > 0
              ? "агенты получают предупреждение"
              : "совпадений не найдено",
      warn: team.duplicates_total > 0,
    },
  ];
  return (
    <section className="team-summary">
      {tiles.map((tile) => (
        <div
          key={tile.title}
          className={cn("team-tile", tile.warn && "team-tile-warn")}
        >
          <span className="metric-label">{tile.title}</span>
          <span className="metric-value">{tile.value}</span>
          <span className="metric-caption muted">{tile.caption}</span>
        </div>
      ))}
    </section>
  );
}

export function TeamScreen({
  page,
  refresh,
  onSuccess,
}: {
  page: PageData;
  refresh: () => void;
  onSuccess: (result: MutationResult) => void;
}) {
  const live = useTeam(page.team, refresh);
  const now = useNow(30000);
  const team = live.data ?? page.team;
  if (!team) return <EmptyState title="Состояние команды недоступно" />;
  return (
    <div className="page team-page">
      <PageHeading
        title="Команда"
        description="Кто сейчас работает, чем заняты агенты каждого человека и где задачи пересекаются"
        actions={
          <>
            <LivePill
              paused={live.paused}
              failed={live.isError}
              fetching={live.isFetching}
            />
            <button
              className="icon-button"
              aria-label="Обновить команду"
              disabled={live.isFetching || live.paused}
              onClick={() => live.refetch()}
            >
              <RefreshCw size={15} className={live.isFetching ? "spin" : ""} />
            </button>
          </>
        }
      />
      {live.isError ? (
        <div className="notice error-notice" role="alert">
          Не удалось обновить. Показано состояние на {date(team.observed_at)}.
        </div>
      ) : null}
      <Summary team={team} />
      {team.duplicates.length ? (
        <DuplicatePairs
          pairs={team.duplicates}
          total={team.duplicates_total}
          page={page}
          onSuccess={onSuccess}
        />
      ) : null}
      <section>
        <SectionHeading title="Люди и их агенты" />
        {team.members.length ? (
          <div className="people-grid">
            {team.members.map((member) => (
              <PersonCard
                key={member.owner}
                member={member}
                now={now}
                me={member.owner === page.principal.owner}
              />
            ))}
          </div>
        ) : (
          <EmptyState
            icon={<Users size={24} />}
            title="Пока никого нет"
            description="Когда агент подключится через MCP или CLI, он появится здесь вместе с текущей задачей."
          />
        )}
      </section>
      <section>
        <SectionHeading
          title="Ждут исполнителя"
          href="/app/tasks?state=ready"
        />
        <div className="list-surface">
          {team.unclaimed.length ? (
            team.unclaimed.map((task) => <TaskRow key={task.key} task={task} />)
          ) : (
            <EmptyState
              icon={<ListTodo size={24} />}
              title="Свободных задач нет"
              description="Все готовые задачи уже взяты агентами или ждут зависимостей."
            />
          )}
        </div>
      </section>
      <p className="muted team-footnote">
        Агент — это подключённая сессия, а не процесс на компьютере. Данные на{" "}
        {date(team.observed_at)}; активность считается за последние{" "}
        {team.recent_window_hours} ч.
      </p>
    </div>
  );
}
