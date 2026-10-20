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

## Source/operator review — APPROVE

Проверены все frozen WORK130 файлы (авторский manifest SHA256
`bca540a82cbdfd84a6667eac9cb964e1e62f24d30e5d475d8620460f2d305ac7`),
Next standalone config, root builder tracked-assets delta и private transport
SHA256 `0eb5144b3c506b9826d204930cf69c82f25a53ac74a8ce52948b72019415ae7c`.
Независимый manifest11code/testfiles — `.local/task008-review/code-manifest.json`,
SHA256 `e231b7eca4edbae5789a7d6884a00d3fd47f2be5aae0411dff587c1169b7005e`.
Файлы перечитаны, хэши после freeze совпали. Открытых обязательных findings нет.

Закрыто: Darwin native заменены exact-lock Linux SRI packages; static/public
скопированы в custom distDir; first-current systemd verification проверяет
immutable path; canonical absolute прежний pointer поддержан; existing Next
account строго отделён UID/groups; SIGTERM/INT/HUP вызывают recovery с защитой
повторных сигналов; transport проверяет root-owned nonsymlink ancestors до
распаковки и сам manifest; builder поставляет только tracked deployment assets,
не ignored Python caches. Прежние private routes, upload и native nginx bytes
сохраняются узким renderer. При rollback каждый шаг независим, БД не заменяется.

Reviewer independently запустил новые unit **7/7 PASS**. Root evidence полного
infra suite **83/83 PASS** прочитано; это mocked operator/unit, не Linux/live proof.
Официальный [Next standalone contract](https://nextjs.org/docs/app/api-reference/config/next-config-js/output)
проверен: explicit static/public copy и HOSTNAME/PORT соответствуют документации.

**APPROVE source** разрешает отдельному WORK132 написать process/E2E тесты и
root собрать/перенести immutable inactive release. Финальное переключение
ожидает review новых тестов, exact artifact provenance и actual Linux staging;
они пока NOT-RUN. Этот verdict не является live production PASS.
