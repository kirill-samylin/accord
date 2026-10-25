import type { PageData, Task } from "./types.ts";
// Preserve every entity, including newly introduced server states and epic 'open'.
// A ready task still waiting on a dependency is listed under "blocked", right
// after the ready group, so "Готова" counts only tasks that can be taken now.
export function groupTasks(tasks: Task[], order: string[]) {
  const groups = new Map<string, Task[]>();
  for (const task of tasks) {
    const key = task.state === "ready" && task.blocked ? "blocked" : task.state;
    const group = groups.get(key) ?? [];
    group.push(task);
    groups.set(key, group);
  }
  const ready = order.indexOf("ready");
  const ordered =
    ready >= 0 && !order.includes("blocked")
      ? [...order.slice(0, ready + 1), "blocked", ...order.slice(ready + 1)]
      : order;
  const states = [
    ...ordered,
    ...[...groups.keys()].filter((state) => !ordered.includes(state)),
  ];
  return states
    .filter((state) => groups.has(state))
    .map((state) => ({ state, tasks: groups.get(state)! }));
}
// "Мои вопросы": the server lists questions the person may edit; a question
// the current session has claimed (holds the lease) is theirs too, even when
// someone else created it. Order: server list first, then claimed ones.
export function myQuestions(
  page: Pick<PageData, "my_questions" | "questions" | "tasks" | "session">,
) {
  const mine = [...(page.my_questions ?? [])];
  const seen = new Set(mine.map((task) => task.key));
  const session = page.session.session_id;
  for (const task of [
    ...(page.session.my_tasks ?? []),
    ...(page.questions ?? []),
    ...(page.tasks ?? []),
  ]) {
    if (seen.has(task.key) || task.role !== "human") continue;
    if (task.lease?.session_id !== session) continue;
    seen.add(task.key);
    mine.push(task);
  }
  return mine;
}
// Task-list tabs live in ?tab= (client-only, see project-routes.ts).
export function taskTab(value: string | null, questions: boolean) {
  const tabs = questions ? ["all", "mine", "ready"] : ["all", "epics"];
  return value && tabs.includes(value) ? value : "all";
}
