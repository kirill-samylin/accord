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
// Russian titles for every event type the server records: board events
// (internal/store emit), automation/managed events and project/admin audit
// rows. An unknown type falls back to a neutral title, never the raw code.
export const eventLabels: Record<string, string> = {
  token_created: "Доступ выдан",
  token_revoked: "Доступ отозван",
  tokens_revoked_all: "Все доступы отозваны",
  session_opened: "Сессия открыта",
  task_created: "Задача создана",
  task_updated: "Задача обновлена",
  task_assigned: "Назначение задачи изменено",
  check_created: "Проверка назначена",
  task_duplicate: "Отмечена дублем",
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
  profile_created: "Профиль агента создан",
  profile_updated: "Профиль агента обновлён",
  profile_state_changed: "Состояние профиля изменено",
  automation_policy_created: "Политика автоматизации создана",
  automation_policy_updated: "Политика автоматизации обновлена",
  automation_history_archived: "История автоматизации перенесена в архив",
  workflow_submitted: "Требование отправлено",
  workflow_plan_submitted: "План работы предложен",
  workflow_question_answered: "Ответ на вопрос автоматизации",
  workflow_completed: "Требование выполнено",
  workflow_pause: "Автоматизация приостановлена",
  workflow_cancel: "Автоматизация отменена",
  workflow_resume: "Автоматизация продолжена",
  workflow_clarify: "Уточнение отправлено",
  workflow_replan: "Запрошен новый план",
  managed_host_enrolled: "Компьютер подключён",
  managed_host_updated: "Компьютер обновлён",
  managed_host_disabled: "Компьютер отключён",
  managed_start_queued: "Запуск агента запрошен",
  managed_stop_queued: "Остановка агента запрошена",
  managed_resume_queued: "Продолжение агента запрошено",
  managed_clarify_queued: "Уточнение агенту отправлено",
  managed_session_closed: "Сессия агента закрыта",
  managed_report_accepted: "Компьютер принял команду",
  managed_report_started: "Агент запущен",
  managed_report_completed: "Агент завершил работу",
  managed_report_stopped: "Агент остановлен",
  managed_report_failed: "Запуск агента не удался",
  managed_report_recovered_stopped: "Остановка агента восстановлена",
  managed_report_delivery_unknown: "Доставка команды не подтверждена",
  project_created: "Проект создан",
  project_updated: "Проект обновлён",
  member_grant: "Участник добавлен",
  member_set: "Роль участника изменена",
  member_revoke: "Участник удалён",
  repository_add: "Репозиторий добавлен",
  repository_update: "Репозиторий обновлён",
  repository_remove: "Репозиторий удалён",
  person_created: "Человек добавлен",
  person_updated: "Данные человека обновлены",
  person_activated: "Человек снова активен",
  person_deactivated: "Человек отключён",
  identity_linked: "Доступ привязан к человеку",
  credential_issued: "Доступ выдан",
  credential_revoked: "Доступ отозван",
  admin_bootstrapped: "Администратор назначен",
  offline_owner_recovery: "Владелец восстановлен офлайн",
  offline_restore: "Данные восстановлены из копии",
  logout: "Выход из системы",
  design_uploaded: "Материал загружен",
  design_upload_aborted: "Загрузка материала прервана",
};
// The board activity filter accepts only this recorded vocabulary
// (core.ValidControlEventType); other types are display-only.
export const eventFilterTypes = [
  "token_created",
  "token_revoked",
  "tokens_revoked_all",
  "session_opened",
  "task_created",
  "task_updated",
  "task_assigned",
  "check_created",
  "task_duplicate",
  "claimed",
  "released",
  "commented",
  "linked",
  "unlinked",
  "task_ready",
  "task_in_progress",
  "task_done",
  "task_cancelled",
  "human_answered",
  "epic_open",
  "epic_done",
  "epic_cancelled",
  "epic_reopened",
  "lease_expired",
  "lease_recovered",
  "attempt_recovered",
  "database_recovered",
  "note_created",
  "note_deleted",
  "blocked_changed",
  "profile_created",
  "profile_updated",
  "profile_state_changed",
] as const;
// eventTitle never shows a raw snake_case code to people.
export function eventTitle(type: string) {
  return eventLabels[type] ?? (type ? "Событие" : "—");
}
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
// bytes uses binary units with the Russian decimal comma; files under 1 МиБ
// are shown in КиБ so a small upload never reads as "0,0 МиБ".
export function bytes(value: number) {
  return value < 1024
    ? `${value} Б`
    : value < 1048576
      ? `${number(value / 1024)} КиБ`
      : `${number(value / 1048576)} МиБ`;
}
export function initial(value: string) {
  return value.trim().slice(0, 2).toUpperCase() || "A";
}

// One-line meaning of each task state for legends and tooltips.
const statusHints: Record<string, string> = {
  created: "черновик, ещё не в очереди",
  ready: "в очереди, ждёт исполнителя",
  in_progress: "исполнитель работает",
  done: "результат принят",
  cancelled: "работа остановлена",
  canceled: "работа остановлена",
  blocked: "ждёт другую задачу",
};
export function statusHint(state: string) {
  return statusHints[state] ?? "";
}
// Russian plural form: plural(1, "версия", "версии", "версий") → "версия".
export function plural(count: number, one: string, few: string, many: string) {
  const n = Math.abs(count) % 100;
  const last = n % 10;
  if (n > 10 && n < 20) return many;
  if (last === 1) return one;
  if (last >= 2 && last <= 4) return few;
  return many;
}
