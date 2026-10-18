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
