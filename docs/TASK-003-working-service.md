# TASK-003 — рабочий общий Accord

- Статус: в работе; этап: C live/performance FAIL, D fixes, E foundations.
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

## Текущее состояние

- VPS:1e9025c/schema2, C HTTP+MCP12/note/catalog/stats smoke PASS049;
  native настоящий Codex ранее PASS, Tandem/443 сохранены.
  C checkpoint опубликован в origin/codex/pilot; nativefix также origin/codex/native-proxy.
- Main checkpoint2dfd86b: C source/Host APPROVE041, полный C check PASS042,
  новые process tests/README APPROVE045. Method delta047 APPROVE048; C QA checkpoint1e9025c committed.
- D: config-runtime-003/codex/config-runtime, HEAD932aa2a; source freeze043,
  main1e9025c объединён; независимые code050/security051 выполняются.
- E pure libraries: integration-libs-003/codex/integration-libs,59ae1a1,
  автор044, APPROVE046; SQL/runtime/schema4 и включение доставок отсутствуют.
- Непосредственно далее: WORK049 private deploy1e9025c/schema2 и isolatedVPS
  load; D независимые reviews после sourcefreeze, E runtime после D.

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
| 041 | Независимый повторный reviewer/test review | C038+Host combination, native039 tests, инструкции | APPROVE всех трёх областей |
| 042 | Отдельный тестировщик C | main tests/acceptance/infra; своя WORK | Schema2/process приёмка после041 |
| 043 | Разработчик runtime D | Изолированная codex/config-runtime от2dfd86b | Config schema3, роли/scopes и workflow rules |
| 044 | Разработчик E libraries | Изолированная codex/integration-libs | GitHub/notification/interchange foundations |
| 045 | Независимый test reviewer C | Tests042/README/load method | C tests APPROVE, harness REQUEST CHANGES |
| 046 | Независимый reviewer/security E libraries | Frozen044 | APPROVE foundation; runtime obligations сохранены |
| 047 | Новый developer/tester load | Только cgroup helper/harness tests/docs | Исправил REV045-01, targeted14 PASS |
| 048 | Независимый reviewer load delta | Frozen047 | APPROVE метода до isolatedVPS load |
| 049 | DevOps C live/load | Только private runtime/evidence и свой report/WORK | Deploy1e9025c/schema2, изолированный20/2000/20 load, сохранность Tandem |
| 050 | Независимый code reviewer D | Frozen codex/config-runtime; только свой report/WORK в main | Полный D source/контракты/миграции/новые unit tests |
| 051 | Независимый security D | Тот же frozen D; только свой report/WORK в main | REQUEST CHANGES: single_editor + общие050 findings |
| 052 | Новый developer D fixes | Изолированная codex/config-fixes от932aa2a | Единый пакет050/051, узкие source/unit fixes |
| 053 | Архитектор performance | Read-only C1e9025c + private profiling; свой contract/WORK | Причина load FAIL и исполнимый безопасный fix contract |
| 054 | Независимый code/security recheck D | Frozen052 codex/config-fixes; свой report/WORK в main | Проверка всех050/051 findings и delta-regressions |
| 055 | Developer load method | Только Python acceptance harness/helpers/unit/doc; main | Явный offered-rate режим без ослабления native admission |
| 056 | Отдельный tester D | Изолированная codex/config-qa отd5a3a12, только tests/docs | Process приёмка D после APPROVE054, TTL/bootstrap/schema3 fixtures |
| 057 | Developer performance | Изолированная codex/performance отd5a3a12 | Единственный schema4 owner: cache/COW/dirty/events/stats, полный053 contract |

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
- WORK038 исправлена и041 APPROVE: все3 независимых repro PASS, C+Host
  combination и новые native039 process tests также APPROVE. Source checkpoint
  2dfd86bf1627775a8fdd87c650d9a3ada4cae17f;042 обновляет schema/tool fixtures и
  выполняет один полный make check. До этой приёмки source checkpoint не release.
- WORK043 выполняет D только в отдельной codex/config-runtime, не меняет main
  во время042. Перед объединением требуется merge свежих C test-fixtures,
  независимые code/security и затем отдельная D process приёмка.
- WORK040 E-контракт сохранён; root уточнил local-admin import requeue для
  imported_unknown. E ещё не реализована, UI/public ingress/recipients открыты.

- WORK048 APPROVE cgroup-method047,14 tests и3 synthetic probes PASS.
  C QA checkpoint1e9025cd6e3122f0adf6165e09845202736d004a committed;
  Linux/amd64 immutable release собрана из чистого дерева pinnedGo1.27.2.
  WORK049 получает только этот артефакт; D не включена в live-сборку.

- WORK043 source freeze414a24ff…481, commitdad4c5f; main1e9025c объединён
  в codex/config-runtime без конфликтов. Назначены независимые050/051.
  Существующие process fixtures требуют schema3/persistedTTL adaptation после
  approve: это ожидаемый test-input drift, не повод ослаблять startup guards.

- WORK049 live C upgrade/schema2 и authenticated HTTPS12/note/catalog/stats
  smoke PASS; отдельный synthetic token отозван/401, Mac token сохранён.
  Target load20/2000/20 выполняется в отдельном transient unit сCPU100%/512MiB.
  Git push origin/codex/pilot1e9025c PASS; main GitHub не менялась.

- Root independent dependency scan D HEAD932aa2a: govulncheck ./... PASS,
  `No vulnerabilities found` (Go1.27.2; новая x/text0.42.0 включена).
  Это dependency scan, не замена code/security review050/051.

- WORK050/051 нашли обязательные D исправления: singleton lease resurrection
  после смены config; single_editor bypass через update; human can_cancel
  epic bypass; incomplete child epic closure; area-block queue starvation;
  idempotency omitted/empty areas. D не deployed и не опубликована.
  WORK052 исправляет пакет в отдельной копии, исходный freeze932aa2a сохранён.

- WORK049 target20/2000/20 завершён: latency p95=10025.11ms FAIL при50ms;
  sampled serverRSS35.72MiB и populatedstartup28.27ms PASS. Нужен отдельный
  performance analysis/fix/review/retest; успешный deploy не означает готовность.
  Leaf CPU100%/512MiB подтверждены, effective ancestors unavailable, sharedVPS.

- WORK052 frozen source13899d45…2732,25 filehash,8 regressions/50 targeted/
  11race/vet PASS; автор воспроизвёл6 исходныхdefects старым productionoverlay.
  WORK054 выполняет независимый code/security recheck; processDQA ещёNOT-RUN.

- WORK054 APPROVE code/security d5a3a12: все6 original probes PASS и EXPECTED
  FAIL наoldGit baseline,142 unit/integration/16race/vet PASS.
  WORK056 начинает D processQA; performance053 остаётся обязательнымblocker.

- WORK053 завершена: local profiles подтверждают full-board persistence и annual
  event scan bottlenecks. Root принял технический contract053: performance
  получает schema4 (владелец057), E перенесена наschema5 до начала её runtime.
  Config/integration contracts синхронизированы. Все прежние latency FAIL/
  NOT-RUN сохраняются до реальной повторной приёмки; цели не ослаблены.
