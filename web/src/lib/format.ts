export const stateLabels: Record<string, string> = {
  open: "Открыт",
  created: "Создана",
  ready: "Готова",
  in_progress: "В работе",
  done: "Завершена",
  cancelled: "Отменена",
  active: "Активна",
  expired: "Истекла",
  credential_invalid: "Доступ отозван",
  blocked: "Заблокирована",
};
export const roleLabels: Record<string, string> = {
  coordinator: "Координатор",
  analyst: "Аналитик",
  architect: "Архитектор",
  developer: "Разработчик",
  reviewer: "Ревьюер",
  tester: "Тестировщик",
  human: "Человек",
};
export const eventLabels: Record<string, string> = {
  task_created: "Создана задача",
  claimed: "Задача взята в работу",
  task_ready: "Задача готова",
  task_in_progress: "Задача в работе",
  task_done: "Задача завершена",
  task_cancelled: "Задача отменена",
  released: "Задача освобождена",
  human_answered: "Сохранён ответ",
  lease_expired: "Истекла аренда",
  session_opened: "Открыта сессия",
  task_updated: "Обновлена задача",
  commented: "Добавлен комментарий",
  linked: "Добавлена связь",
  unlinked: "Удалена связь",
  epic_open: "Эпик открыт",
  epic_done: "Эпик завершён",
  epic_reopened: "Эпик переоткрыт",
  design_uploaded: "Загружен дизайн",
};
export function label(value?: string) {
  return value
    ? (stateLabels[value] ?? roleLabels[value] ?? eventLabels[value] ?? value)
    : "—";
}
export function date(value?: string, time = true): string {
  if (!value || value.startsWith("0001-")) return "—";
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return "—";
  return new Intl.DateTimeFormat("ru-RU", {
    day: "numeric",
    month: "short",
    ...(time ? ({ hour: "2-digit", minute: "2-digit" } as const) : {}),
  }).format(parsed);
}
export function number(value: number) {
  return new Intl.NumberFormat("ru-RU", { maximumFractionDigits: 1 }).format(
    value,
  );
}
export function duration(seconds: number) {
  return seconds < 60
    ? `${number(seconds)} с`
    : seconds < 3600
      ? `${number(seconds / 60)} мин`
      : `${number(seconds / 3600)} ч`;
}
export function bytes(value: number) {
  return value < 1024
    ? `${value} Б`
    : value < 1048576
      ? `${number(value / 1024)} КБ`
      : `${number(value / 1048576)} МБ`;
}
export function initial(value: string) {
  return value.trim().slice(0, 2).toUpperCase() || "A";
}
