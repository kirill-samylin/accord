import { assignmentText } from "@/lib/coordination";
import type { Task } from "@/lib/types";
// AssignmentLabel uses the same wording as the task detail; a live lease is
// named by its holder, with the opaque session hash only in the tooltip.
export function AssignmentLabel({ task }: { task: Task }) {
  const assignment = task.assignment;
  return (
    <span
      className="task-assignment"
      title={
        assignment
          ? `${assignment.profile_name} · версия ${assignment.profile_version}`
          : task.type === "epic"
            ? "Ответственный за эпик"
            : task.role === "human"
              ? "Задача ждёт ответа человека"
              : "Задача доступна общей очереди роли"
      }
    >
      <span className="truncate">{assignmentText(task)}</span>
      {task.lease ? (
        <span
          className="muted truncate"
          title={`Сессия: ${task.lease.session_id}`}
        >
          {task.holder_owner
            ? `Делает ${task.holder_owner}`
            : "Взята сессией агента"}
        </span>
      ) : null}
    </span>
  );
}
