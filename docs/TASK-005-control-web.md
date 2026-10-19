# TASK-005 — панель управления Accord по Paperclip

- Родитель REQ-005. Источник: пользователь19.10.2026 выбрал Paperclip как источник
  дизайна/возможностей и поручил «Приступай»; прежний runtime Accord сохраняется.
- Статус: в работе, реализация после independent analyst/architect/reference audit.
- База a6501d8, codex/web-control. Общий REQ/состояние редактирует root099.
- Контракт: control-web-contract.md §8 и приоритетные уточнения §9; reference
  WORK102, upstream5739a53. Давинчиread-only, TandemOFF.
- Результат этой поставки: полный control shell/overview, реальные agent sessions
  с task/credential validity, задачиlist/board+wholeprojectsearch, recentactivity,
  stats/settings/currentproject и сохранённые вопросы/дизайн/решения.
- Сервисы managed execution/multiproject registry/provider config/budgets/skills
  не подменяются формами; future масштабирование описано ADR. ПолныйREQupload
  всех18категорий остаётся дорожнойкартой. PC-01 пользователю отправлен.

## Приёмка

WORK100 AC-01…10 + архитектурныеAC, code/security/actualUI независимыеreviews,
process/browser QA послеcodeapprove, полный makecheck одинраз, actualHTTPS
upgrade/smoke+backup и публикацияпровереннойветки. Авторовпроверяютдругиеагенты.
Сохранитьauth/Origin/CSRF/leases/MCP12/cookies/nativeclients/базы/файлы/историю.
НичегоизневыпущенныхD4/Eнепереносить. НикакихсекретоввGit.

## Назначения

- WORK100 analyst — готовыйконтракт/inventory/roadmap.
- WORK101 architect — готовыйконтрактDTO/будущаяизоляция.
- WORK102 early design review — source/captures/tokens,state gaps, неfinalapprove.
- WORK103 backend author — core/control, store/control(+cataloghelper), internal/web/**.
- WORK104 UI author — internal/webui/**.
- WORK105 security contract — контракт/новыereaders/poll, собственнаяWORK/evidence.
- Далее code/security/UI reviews → testerprocess/browser → release.
