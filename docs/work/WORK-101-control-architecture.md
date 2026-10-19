# WORK-101 — архитектура панели управления Accord

- Родитель: REQ-005; роль architect, исполнитель `/root/control_architecture101`.
- База: a6501d8, worktree `.local/web-control-099`; статус completed (проектирование).
- Запись: только этот WORK и `/private/tmp/accord-control-099/architecture/`.
- Прочитаны actual AGENTS Accord, REQ-005, architecture/api/config/task status,
  architect/general/parallel-work Davinci read-only; web/store/core/transport C2,
  Paperclip Dashboard/Agents/Projects и resource schemas read-only.
- Claude role/model bindings недоступны; модель унаследована, не изменялась.
- Tandem OFF согласно AGENTS Accord; source/Git/deploy/секреты не изменялись.

## Результат

Контракт: `/private/tmp/accord-control-099/architecture/control-contract.md`.
Предложены Go SSR, безопасные browser read endpoints, точный Overview, existing
Sessions/Stats и bounded DESC RecentEvents без изменения native ASC cursor.
Уточнение root: frozen DTO appendix включает ControlSessions с batch effective
principal validation и whole-project ControlTasks search до пагинации; native
Sessions TTL semantics не расширяется и native tools остаются прежними.
Статусы сессий/аренд отделены от неизвестного process-running; точный last_seen
не восстанавливается. Stats не poll-ится из-за полного scan журнала.
Текущие board schema2/web schema1 и fences/CSRF/auth/Origin сохраняются.

В этой волне один реальный проект и реальные session facts; future registry/ACL
и разделение ресурсов спроектированы отдельно, epic не объявляется проектом.
Нет ложной model config, browser arbitrary URL, универсального token или
неподтверждённой интеграции D4/E. Root подтвердил эту границу в ходе задания.
Есть границы backend/UI/root integration, AC и регрессии до выпуска.

## Проверки и ограничения

- PASS: source/contracts read, фактические schema/interfaces/expiry semantics,
  official Go html/template/net/http API documentation checked 19.10.2026.
- NOT-RUN: реализация, тесты, browser/live/load/deploy/rollback — не область роли.
- Независимый approve реализации не выдан; документ — архитектурный handoff.
- Product PC-01 об execution panel ожидается у root; не блокирует control UI.
- Историческая память использована только для ориентации identity/deployment;
  текущие выводы проверены по worktree, live состояние здесь не проверялось.

## Журнал

- 2026-10-19T19:16:19Z started — фактический date, прочитаны инструкции и текущий
  резерв REQ-005; область записи подтверждена координатором.
- 2026-10-19T19:17:54Z design_ready — фактический date перед записью; architecture
  передана координатору, source/Git/deploy не менялись.
- 2026-10-19T19:20:22Z completed — фактический date; записаны DTO и effective-token
  status/search уточнения root, handoff paths переданы для старта authors.
