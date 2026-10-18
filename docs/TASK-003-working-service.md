# TASK-003 — рабочий общий Accord

- Статус: в работе; этап: контракты оставшегося объёма и эксплуатационная поставка.
- Источник: пользователь18.10.2026 поручил доделать оставшееся до рабочего проекта.
- База:966dda6, рабочая ветка codex/pilot, репозиторий Accord.
- Координатор: основной агент. Общую карточку редактирует только координатор.
- Новый запрос разрешает реализацию, безопасное развёртывание общего Accord,
  проверки на доступном Mac/VPS и публикацию проверенных изменений.
  Существующие Tandem/его данные/HTTPS443 сохраняются; доска не используется.
- Уточнение пользователя: GitHub OAuth App нет; сначала рабочий доступ
  по отдельным токенам. GitHub OAuth/browser identity не подменяется bearer.
- Второй физический ПК пользователь сможет проверить позже, ОС пока не указана.
  Это внешняя зависимость; подготовить воспроизводимую инструкцию и не заявлять PASS.

## Цель и критерии

Общий сервис на существующем VPS через защищённую частную сеть; отдельные
токены машин; воспроизводимые установка/обновление/откат, автозапуск и backup/
restore; клиенты поддерживают lease во время длительной работы; реальный Codex
с Mac использует удалённую очередь; необходимые функции и ограничения
сверены с исходным ТЗ и требованиями. Остаток полной V1 отслеживается явно,
отсутствующая внешняя credential/машина не превращается в фиктивную проверку.
Полный объём нельзя молча сократить до deploy неизменённого0A.

## Назначения

| WORK | Роль | Область записи | Результат |
| --- | --- | --- | --- |
| 018 | Архитектор/аналитик | docs/service-contract.md; docs/work/WORK-018-service-contract.md | Полная карта требований и точные контракты завершения |
| 019 | DevOps | infra/accord/**; docs/deployment.md; docs/work/WORK-019-deployment.md | Поставка/backup/restore/rollback, пока без live-мутаций |
| 020 | Security контракт | docs/security-contract.md; своя WORK | Обязательные меры до общего доступа |
| 021 | Разработчик bridge | internal/bridge и общий internal/client; CLI mcp/main/client; SessionView/Event DTO, store/sessions; своя WORK | Stdio MCP с automatic heartbeat и quarantine |
| 022 | Разработчик admin | store/auth/backup/maintenance; CLI admin/token; новые core/admin типы и tests; своя WORK | Backup, token list/audit/revoke-all, offline recovery reset |
| 023 | Разработчик транспорта | HTTP routing/helpers/security/metrics и tests; своя WORK | Origin policy, bounded limits, безопасные headers/metrics |
| 024 | Независимый security | docs/security-003.md; своя WORK | Проверка native release перед VPS |
| 025 | Отдельный исправляющий developer | infra install/rollback/recovery/restore/service; своя WORK | Возврат предыдущего executable и durable startup guards |
| 026 | Независимый reviewer | docs/review-003.md; своя WORK | Код/новые тесты и финальные infra артефакты |
| 027 | Независимый security recheck | docs/security-003-recheck.md; своя WORK | Закрытие infra findings и проверка native release |
| 028 | Тестировщик | tests/acceptance, tests/infra, Makefile wiring; своя WORK и acceptance | Process/infra E2E после approve |
| 029 | DevOps live | .local/release/accord runtime; свой report/WORK | Установка на VPS и изолированные эксплуатационные проверки |
| 030 | Независимый test reviewer | свой report/WORK | Новые E2E, wiring и client guide |
| 031 | Архитектор продолжения D | docs/config-contract.md; своя WORK | Исполнимый контракт конфигурации/ролей/миграций |
| 032 | Разработчик C store | internal/core, internal/store, schema2 и проверки совместимости backup; своя WORK | Списки, заметки, показатели и история |
| 033 | Разработчик D parser | internal/config, schema/templates, go.mod/go.sum; своя WORK | Строгая offline конфигурация и validation |
| 034 | Координатор как разработчик C transport | HTTP/MCP/CLI consumers без store/core; своя WORK | Публичный доступ к новым возможностям C |
| 035 | Разработчик proxy fix | Изолированная codex/native-proxy от99e722c | Явная Host policy для native MCP через TLS proxy |
| 036 | Независимый reviewer/security | Native host policy + offline config parser | Обе дельты APPROVE |
| 037 | Независимый reviewer/security C | Source C без parser/native fix | REQUEST CHANGES,3 дефекта |
| 038 | Новый разработчик C fixes | Store metric attribution/blocked + bridge JSON precision | Исправляет пакет037 |
| 039 | Тестировщик native proxy | Process regression в native worktree; свой отчёт | После036 approve |
| 040 | Архитектор E | docs/integration-contract.md; своя WORK | Конкретный контракт webhooks/import/export |

Независимые review/security выполняются до live deploy. Unit/integration пишет
разработчик; новые process/E2E — отдельный tester после approve и review тестов.
Модели ролей именованных Claude-агентов здесь не закреплены.

## Открытые внешние условия

- Второй ПК: проверка позже по ответу пользователя; ОС неизвестна.
- GitHub-вход: явное решение пользователя сначала отдельные токены.
- Product UI: существующие требования предусматривают согласованные макеты;
  наличие frontend не выдумывается. Архитектор выделяет functional/deployment
  критерии и точные недостающие продуктовые решения.

## Журнал

- Начато продолжение от завершённого TASK-002; рабочая копия исходно чистая.
- WORK-018 завершена: [service-contract](service-contract.md) сохраняет карту
  всего остатка V1 и выделяет последовательные реализуемые срезы.
- WORK-020 завершена: [security-contract](security-contract.md), SEC01–12;
  это требования, не approve реализации. Restore не должен оживлять доступ.
- WORK-021/022 выполняются параллельно с непересекающимися файлами:
  types.go/main.go принадлежат021, admin.go/token.go/auth.go —022.
  В types.go021 добавляет TTL hints и Event.TokenID, нужные022.
- Веб: асинхронно уточняется простой functional SSR по исходномуТЗ либо
  макеты отдельно; остальная реализация не ждёт визуального решения.
- WORK-019: артефакты infra готовы к независимому review, local helper/syntax
  проверки PASS. Backup mock в helper не считается appbackup/runtime PASS.
  Live VPS пока не изменён. Manifest11:242ac6e4…1826.
- WORK-023 закрывает SEC07–09 транспорта перед общим запуском.
- WORK-021/022: application delta заморожена; targeted unit/integration/race
  PASS. Snapshot теперь публикуется из проверенного private staging атомарным
  no-overwrite hardlink; неполный final файл никогда не виден.
- WORK-024: REQUEST CHANGES. Security не нашёл auth/fencing bypass в новой
  application части, но обнаружил два infra blocker: исключение старта/backup
  обходило возврат предыдущего рабочего executable; restore guard требовал
  directory fsync до остановки/замены DB. Оба исправляет отдельная WORK-025.
- WORK-026 проверяет application code/new tests независимо; финальная
  security приёмка потребуется после freeze infra исправлений.
- WORK-025 завершена; WORK-026 и027 дали APPROVE: безопасный schema-aware
  возврат executable, отдельный durable inspectionguard, durable restoreguard,
  дубли switch/healthy устранены. Данные при rollback не откатываются молча.
- WORK-028: полный свежий make check PASS, всего9 process top-level tests,
  19 постоянных Python infra regressions, race/vet/format/build. Implementation
 75 файлов не изменена; новая test/wiring дельта проверена WORK-030 APPROVE.
- WORK-030 также проверила client guide и narrow README/development docfix
  о Python3stdlib prerequisites. macOS/Linux/Windows способы не смешаны.
- Native release готова к actual VPS приёмке WORK-029. Полный запрос остаётся
  в работе: C/D/E из service-contract и внешние условия не закрыты одним deploy.
- WORK-031 уточняет следующий конфигурационный срез независимо от live deploy.
- WORK-031 завершена; контракты C/D сохранены в config-contract.
- Native99e722c44f83b1808120bba1fabb04cf2f68a5b5 собрана из чистого committed
  дерева. WORK-029 устанавливает только этот immutable artifact. Изменения C/D
  не включаются в сервер до отдельного review/security/приёмки.
- WORK-032/033 начаты с непересекающимися областями. WORK-034 ждёт additive
  DTO от032; coordinator остаётся единственным редактором общей TASK.
- WORK-029: HTTP/systemd/TLS/отзыв/backup и isolated Linux restore PASS,
  Tandem сохранён. **Actual remote MCP FAIL**: SDK localhost DNS rebinding
  guard отклоняет Host защищённого Tailscale proxy. WORK-035 исправляет это
  с явной exact-host policy в изолированной копии, без ослабления Origin/auth.
- Проверенная native99e722c отправлена в origin/codex/pilot; main не изменена.
  Публикация не означает MCP ready: blocker выше остаётся до реального retest.
- Blocker native MCP устранён WORK035, независимое036 APPROVE. На сервере
  dc9c74cd692e287173def9fb5743dbf18ffdef21, схема1; реальный Codex с Mac
  прошёл10MCP-вызовов, synthetic задачи отменены, token отозван/401 проверен.
  Сохранность всех Serve routes/Tandem подтверждена до/после обновления.
- Config parser033 отдельно APPROVE036. C store/transport037 REQUEST CHANGES:
  participant attribution, точный int64 через SDK bridge и закрытие blocked
  интервала при отмене. Свежий038 исправляет единый пакет; C пока не deployed.
