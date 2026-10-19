# WORK-113 — восстановление фильтра эпика в control web

- Исполнитель root, интегратор; родитель TASK-005. Backend103 завершён, запись
  core/control_validation.go и web/control_queries.go передана для этой дельты.
- Источник: QA107 actual HTTP /web/tasks?epic_key=E-2 возвращал400. В новом
  perimeter общий case epic_key/task_key ошибочно вызывал T-only validation.
- Regression написан до исправления: TestControlEpicFiltersPreserveExistingRoutes
  воспроизвёл400 на canonical E-1; proof epic-before-fix.log сохранён внеGit.
- Исправление: отдельный ValidControlEpicKey с той же canonical positive int64
  проверкой, общий private helper дляT/E. Browser query различает epic_key/task_key,
  а прежняя T-only semantics task_key/RecentEvents/native остаётся прежней.
- Unit проходит реальные web handlers и authenticated store на пяти маршрутах:
  tasks, legacytasks, questions, legacyquestions, history. Все canonical E-N дают200,
  чужой prefix/zero/leadingzero/negative/overflow/HTML отвергаются400. Task activity
  по T-N работает; E-N не принимается за task_key.
- После исправления полный affected packages прогон PASS: store2.056s, web1.762s,
  transport1.240s, webui0.356s. Batch log и ранее failing proof сохранены private.
- Никакого изменения tokens/auth/session/claims/DB/native protocol. Новый regression
  meaningful, не проверка копии реализации. Нужен независимый delta review112;
  browser case107 повторяется на новом реальном бинаре после approve.
