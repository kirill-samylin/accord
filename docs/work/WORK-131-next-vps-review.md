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

## Immutable artifacts / Linux — PASS; active switch APPROVE

Release **`80486fc472324230a9cda9dbe9906c41998fb19f`**. Independently verified
Next1250/1250 manifest file hashes, manifest SHA256
`d1c42019ec0fc86e24b8e7c7d0b292cb9f4de8f0668e1c9d2758bde430ca94f4`;
Go34/34 release hashes, executable SHA256
`b6ae1f26955bf74d43056642ea33a95bd0f65a55673bb8ac45fa54273a641f3c`.
BuildID `8_G-OjVHWfBC6zEaE2O5N`. Darwin native/ignored pyc отсутствуют.
Official downloaded Node archive matches SHASUMS256, extracted bin SHA256
`bc17c508ffeed0ec622934f9b7fa72f8e78da65350e63c3eceb56fa688aa5e12`, LICENSE
совпадает с archive member. Remote inactive transfer proofs обоих bundles прочитаны.

Новые отдельные process/E2E WORK132 исходники прошли независимое APPROVE:
`next_vps_go132.py` SHA256 `7f25fcdd234940549efc46e4eb30e7979036189135251184edc156f3a455f17d`,
`next_vps_smoke.cjs` SHA256 `d55a809c848868dd828fb3bc0ea0db9d7921c4307a984f3dac6d4fff4189e425`.
Go helper использует только synthetic private `/run` DB/token, один отдельный
порт7895 и finally cleanup. Browser — собственные login/logout, чтение реальной
доски и stateless MCP initialize/tools-list без describe/tool calls; outputs private.

Root actual evidence `.local/task008/linux-stage-proof.json` перечитано: phase
staged, exact candidate nginx SHA `9ca7773d350eec22d49c48a6bee9122bb049c192d8d448af1d72bd9aa765aec0`,
nginx/systemd проверки и real Linux Node/version/sharp/Next nonce startup прошли.
Stage peak48,222,208bytes, NRestarts0; прежний Go pointer и readiness сохранены.
`.local/task008/linux-go-proof.json`: candidate exact binary реально выполнил
readyz, JSON login, cookie overview; process stopped/fixture removed — PASS.
Reviewer лично SSH не выполнял: проверены source, artifact hashes и root evidence.

**APPROVE active switch** этого exact release выдан root. После apply обязательны
WORK132 HTTPS/browser/native smoke и finalize/backup. Они на момент этого
pre-switch verdict NOT-RUN; production stage ещё не объявлен завершённым.

## Первая HTTPS попытка FAIL и исправление — scope APPROVE

WORK132 `production-01/next-vps-results.json` фактически FAIL до входа:
7chunk/font429, интерфейс не загрузился; мутаций0. Это сохранённый отказ,
не PASS. Точная причина расхода общего burst не наблюдалась: request logging
намеренно выключен. Root выполнил reviewed rollback. Proof
`.local/task008/rollback-01-proof.json` перечитан: прежний Goac6/ready service,
nginx hash27ee, env/units/timers прежние; Next unit/current отсутствуют. DB не
восстанавливалась.

WORK130 narrow fix: отдельная ограниченная per-IP zone50r/s/burst100 только
static/fonts/license; app/native/web/upload/ACME и connection32 сохраняются.
Source и новый regression независимо APPROVE, **8/8 unit PASS**. Наследование
проверено по [официальной документации nginx](https://nginx.org/en/docs/http/ngx_http_limit_req_module.html).
Это разрешает новую immutable кандидатуру; окончательный PASS зависит от
нового Linux stage и фактического повторного HTTPS/browser результата.
