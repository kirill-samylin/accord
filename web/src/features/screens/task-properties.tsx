import Link from "next/link";
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
      <Fact name="Исполнитель" value={task.owner || "Не назначен"} />
      <Fact name="Приоритет" value={`P${task.priority}`} />
      <Fact name="Тип" value={task.type === "epic" ? "Эпик" : "Задача"} />
      <Fact
        name="Эпик"
        value={
          task.epic_key ? (
            <Link className="inline-link" href={`/app/tasks/${task.epic_key}`}>
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
