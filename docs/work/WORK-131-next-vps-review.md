# WORK-131 — независимое ревью поставки Next.js на VPS

TASK-008, 20.10.2026. Роль: независимый reviewer, не автор deployment.
Владение: эта карточка и игнорируемая `.local/task008-review/`; SSH, live services,
source deployment и Git reviewer не изменяет. Автор infra — WORK130,
координатор/root владеет inventory/apply, отдельный tester — WORK132.

## Исходный контракт

Прочитаны AGENTS, TASK008, TASK007, next-web, существующие deployment/public/web
операционные контракты, nginx renderer и Next upstream/CSP. Product baseline
`5cc0a88f74a0040540dcb111466696314810ca8f`, documentation base `7d046d5`.
Имеющийся local PASS TASK007 не заменяет Linux и HTTPS evidence.

Проверяется: immutable exact-source release, Linux runtime before flip, Node
только loopback и без доступа к Accord state/credentials, /web/native direct Go,
фиксированные Host/Origin/cookie/CSP, upload limits/deadlines, статические assets,
nginx/systemd syntax, backup только на VPS, rollback всех executable/config/unit
компонентов без восстановления БД поверх пользовательских изменений. Сохранение
backup/renew timers и прежних private маршрутов без подключения Tandem.

## Статус

Предварительное чтение — PASS. Candidate implementation, unit tests, release
manifest, Linux staging, apply/rollback evidence и live smoke — NOT-RUN: ожидается
candidate WORK130/root. До проверки candidate source deployment APPROVE не выдан.
