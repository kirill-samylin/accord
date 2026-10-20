import type { Task } from "./types.ts";
// Preserve every entity, including newly introduced server states and epic 'open'.
export function groupTasks(tasks: Task[], order: string[]) {
  const groups = new Map<string, Task[]>();
  for (const task of tasks) {
    const group = groups.get(task.state) ?? [];
    group.push(task);
    groups.set(task.state, group);
  }
  const states = [
    ...order,
    ...[...groups.keys()].filter((state) => !order.includes(state)),
  ];
  return states
    .filter((state) => groups.has(state))
    .map((state) => ({ state, tasks: groups.get(state)! }));
}
