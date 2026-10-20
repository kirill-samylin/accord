import { ProjectLink as Link } from "@/components/project-link";
import { Fact, Status } from "@/components/common";
import { date, label } from "@/lib/format";
import type { Task } from "@/lib/types";
export function TaskProperties({ task }: { task: Task }) {
  return (
    <dl className="properties">
      <Fact
        name="Статус"
        value={<Status value={task.state} blocked={task.blocked} />}
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
              {task.assignment.profile_name} · версия{" "}
              {task.assignment.profile_version}
            </Link>
          ) : task.type === "epic" ? (
            task.owner || "Без ответственного"
          ) : task.role === "human" ? (
            "Человек"
          ) : (
            "Общая очередь роли"
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
      <Fact name="Поколение" value={task.generation} />
      {task.lease ? (
        <>
          <Fact
            name="Владелец аренды"
            value={
              <Link
                prefetch={false}
                className="mono"
                href={`/app/agents/${task.lease.session_id}`}
              >
                {task.lease.session_id}
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
