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
  token_created: "Доступ выдан",
  token_revoked: "Доступ отозван",
  tokens_revoked_all: "Все доступы отозваны",
  session_opened: "Сессия открыта",
  task_created: "Задача создана",
  task_updated: "Задача обновлена",
  task_assigned: "Назначение задачи изменено",
  check_created: "Проверка назначена",
  claimed: "Задача взята",
  released: "Задача возвращена",
  commented: "Комментарий добавлен",
  linked: "Связь добавлена",
  unlinked: "Связь удалена",
  task_ready: "Задача возвращена в очередь",
  task_in_progress: "Задача в работе",
  task_done: "Задача завершена",
  task_cancelled: "Задача отменена",
  human_answered: "Получен ответ человека",
  epic_open: "Эпик открыт",
  epic_done: "Эпик завершён",
  epic_cancelled: "Эпик отменён",
  epic_reopened: "Эпик открыт заново",
  lease_expired: "Владение истекло",
  lease_recovered: "Владение восстановлено",
  attempt_recovered: "Попытка восстановлена",
  database_recovered: "Хранилище восстановлено",
  note_created: "Заметка создана",
  note_deleted: "Заметка удалена",
  blocked_changed: "Блокировка изменена",
};
export const outcomeLabels: Record<string, string> = {
  approve: "Одобрено",
  changes_requested: "Нужны исправления",
  passed: "Пройдено",
  failed: "Не пройдено",
  not_run: "Не запускалось",
  not_applicable: "Не применимо",
};
export function label(value?: string) {
  return value
    ? (stateLabels[value] ??
        roleLabels[value] ??
        eventLabels[value] ??
        outcomeLabels[value] ??
        value)
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
