import type { Task } from "@/lib/types";
export function AssignmentLabel({ task }: { task: Task }) {
  const assignment = task.assignment;
  return (
    <span
      className="task-assignment"
      title={
        assignment
          ? `${assignment.profile_name} · ${assignment.profile_id} · версия ${assignment.profile_version}`
          : task.type === "epic"
            ? "Ответственный за эпик"
            : "Задача доступна общей очереди роли"
      }
    >
      <span className="truncate">
        {assignment
          ? `${assignment.profile_name} · v${assignment.profile_version}`
          : task.type === "epic"
            ? task.owner || "Без ответственного"
            : "Очередь роли"}
      </span>
      {task.lease ? (
        <span
          className="mono muted truncate"
          title={`Аренда: ${task.lease.session_id}`}
        >
          Сессия {task.lease.session_id.slice(0, 8)}
        </span>
      ) : null}
    </span>
  );
}
