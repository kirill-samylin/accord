import { ProjectLink as Link } from "@/components/project-link";
import { Fact, Status } from "@/components/common";
import { assignmentText } from "@/lib/coordination";
import { actorLabel } from "@/lib/event-display";
import { date, label } from "@/lib/format";
import type { Task } from "@/lib/types";
import { PersonChip } from "@/components/people";
export function TaskProperties({ task }: { task: Task }) {
  return (
    <dl className="properties">
      <Fact
        name="Статус"
        value={<Status value={task.state} blocked={task.blocked} />}
      />
      {task.holder_owner && task.lease ? (
        <Fact
          name="Сейчас делает"
          value={
            <PersonChip
              owner={task.holder_owner}
              prefix={
                task.holder_machine ? `${task.holder_machine} ·` : undefined
              }
            />
          }
        />
      ) : null}
      <Fact
        name="Создал"
        value={task.created_by ? <PersonChip owner={task.created_by} /> : "—"}
      />
      <Fact name="Роль" value={label(task.role)} />
      <Fact
        name="Назначение"
        value={
          task.assignment ? (
            <Link
              prefetch={false}
              className="inline-link"
              href={`/app/profiles/${task.assignment.profile_id}?version=${task.assignment.profile_version}`}
            >
              {assignmentText(task)}
            </Link>
          ) : (
            assignmentText(task)
          )
        }
      />
      <Fact name="Приоритет" value={`P${task.priority}`} />
      <Fact name="Тип" value={task.type === "epic" ? "Эпик" : "Задача"} />
      <Fact
        name="Эпик"
        value={
          task.epic_key ? (
            <Link
              prefetch={false}
              className="inline-link"
              href={`/app/tasks/${task.epic_key}`}
            >
              {task.epic_key}
            </Link>
          ) : (
            "—"
          )
        }
      />
      <Fact name="Категория" value={task.category} />
      <Fact name="Сложность" value={task.complexity} />
      <Fact name="Области" value={task.areas?.join(", ")} />
      <Fact name="Создана" value={date(task.created_at)} />
      <Fact name="Взята в работу, раз" value={task.generation} />
      {task.lease ? (
        <>
          <Fact
            name="Владелец аренды"
            value={
              <Link
                prefetch={false}
                title={task.lease.session_id}
                href={`/app/agents/${task.lease.session_id}`}
              >
                {actorLabel(
                  task.holder_owner,
                  task.holder_machine,
                  task.lease.session_id,
                )}
              </Link>
            }
          />
          <Fact name="Аренда до" value={date(task.lease.expires_at)} />
        </>
      ) : null}
      <Fact name="Версия" value={task.version} />
    </dl>
  );
}
