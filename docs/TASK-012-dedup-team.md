# TASK-012 — без дублей и «кто что делает»

22.10.2026. Ветка `codex/dedup` от `cebcbe5` (закрытие TASK011).
Поручение пользователя: «Мы хотим приложение, которое сможет оркестрировать и
помогать агентам, чтобы агенты не делали одни и те же задачи… чтобы либо мой,
либо второго человека агенты создавали задачи и не было дублей», затем
«приступай. А так же дизайн и функционал поправь… чтобы мы могли видеть кто что
делает».

Статус: **COMPLETE — WORK153 FINAL APPROVE после поставки и проверки evidence**.
Поручение22.10.2026: «залил новые изменения. залей, обнови стенд и дай ссылку».
Исходная версия `4ec5cca`; рабочий release
`64ed870b749289b545d29cd2ec7813c6266de54d`. Поставка и фактические результаты —
[WORK154](work/WORK-154-dedup-delivery.md).
Решения по умолчанию приняты исполнителем, т.к. пользователь не выбрал явно:
режим `warn`, порог `0.6`, похожесть без внешних сервисов (триграммы + словарь).

## Проблема

Ядро уже исключало двойной *захват* (atomic `next`, аренды, fencing), но не
двойное *создание*: агенты двух людей могли завести «Авторизация через GitHub»
и «Вход через GitHub OAuth», и обе задачи ушли бы в работу. Правило
`duplicate_check` и связь `duplicate_of` из исходного ТЗ (§9.7, §5.4) были
отложены до V1 и в коде не исполнялись.

## Что сделано

### Сервер

- `internal/similar` — детерминированная похожесть 0..1: триграммы заголовка,
  пересечение основ слов, небольшой явный RU/EN словарь (вход/логин/auth,
  экран/страница, экспорт/выгрузка, тёмная/dark…), описание как доп. сигнал,
  совпадение `external_id` = 1, разные номера (`T-12`/`T-13`) снижают оценку.
  Перефразы другими словами без общих терминов не ловятся — это ограничение.
- `create`: в той же транзакции ищет похожие **открытые или завершённые за 14
  дней** задачи того же вида (работа / вопрос человеку / эпик) во всём проекте,
  не только в эпике. Проверки review/QA не сравниваются.
  - `warn` (по умолчанию): задача создаётся, ответ содержит
    `warnings:[{rule:"duplicate_check",message,similar:[…]}]`, где у каждой
    похожей задачи есть ключ, оценка, кто создал и кто сейчас делает.
  - `strict`: `DUPLICATE_SUSPECTED` (409), пока не передано `force:true` +
    `force_reason` (≤1 КиБ). Причина и список похожих сохраняются в задаче
    (`duplicate_override`) и событии.
  - `off`: проверка выключена.
  - Повтор того же `request_id` возвращает сохранённый ответ с теми же warnings.
- `next`: в `Claim.warnings` — похожая работа, которую уже держит живая аренда
  (`duplicate_check`), и пересечение областей с живыми арендами (`area_overlap`;
  области по-прежнему метки, не блокировка файлов).
- Отметка дубля: `link{from:T-dup,to:T-main,type:"duplicate_of"}` (MCP, без
  13-го инструмента) или `POST /api/v1/tasks/{key}/duplicate`
  `{request_id,canonical_key,version?,reason?}`. Дубль отменяется со связью и
  событием `task_duplicate`. Чужая живая аренда никогда не снимается
  (`LEASE_HELD`), отменённая основная и циклы запрещены. Права — как у отмены:
  coordinator/human/автор. Восстановление `transition → ready` снимает связь.
- Чтения: `POST /api/v1/tasks/similar` (`{title,description?,type?,role?}`),
  `GET /api/v1/team`; CLI `call similar|duplicate|team`; MCP ресурсы
  `accord://similar{?session_id,title,description,type,role}` и
  `accord://team{?session_id}` (пробелы кодировать `%20`, не `+`).
- `describe` возвращает `rules.duplicate_check` = текущий режим.
- Задачи в ответах получили `created_by` (сохраняется для новых, для старых
  выводится из сессий), `holder_owner`, `holder_machine`, `duplicate_of`.
- `serve --duplicate-mode off|warn|strict --duplicate-threshold 0.3..1`
  (или `ACCORD_DUPLICATE_MODE`). Хранилище: только аддитивные JSON-поля задач и
  событий, без DDL и новой версии схемы.

### Веб

- Новый экран **«Команда»** (`/app/team`, второй пункт меню и мобильной панели):
  люди с цветом и статусом «в сети», их агенты (роль/профиль, машина, текущая
  задача, сколько уже работает, области), свободные агенты, счётчики «в работе /
  создал / завершил за сутки / открыто», список свободных задач. Обновляется
  каждые 10 с, пока вкладка видима; чтение не продлевает сессии и аренды.
- Панель **«Возможные дубли»**: пары похожих задач с оценкой и авторами,
  кнопки «Оставить T-N» (закрывает свободную из пары) и «Это разные» (связь
  `related`, пара больше не показывается).
- Обзор: блок «Кто что делает» по людям и предупреждение о дублях.
- Список задач: «делает denis» / «от kirill», метка «дубль T-N».
- Карточка задачи: баннер дубля/создания вопреки похожим, блок «Похожие
  задачи» с кнопкой «Закрыть как дубль T-N», свойства «Сейчас делает»/«Создал».
- Диалог «Новая задача»: похожие задачи появляются при вводе названия; в
  `strict` создание требует «Это другая задача» и причину.
- Активность: событие «Отмечена дублем», владелец события цветной меткой.

## Как агентам пользоваться

1. Перед `create` прочитать `accord://similar?session_id=…&title=…`.
2. Если в ответе `create` есть `warnings` — открыть похожие задачи; если это та
   же работа, отметить свою `link type=duplicate_of` и не начинать её.
3. Если в `Claim.warnings` есть похожая работа у другого человека — сверить
   объём до правок.

## Исходные проверки автора4ec5cca до поставки

- Go 1.27.2 (собран из исходников вашего toolchain): `go vet ./...`,
  `go test ./...` — PASS, `go test -race` для store/web/similar/transport — PASS.
- Новые тесты: `internal/similar`, `internal/store/duplicates_test.go`,
  `internal/web/duplicates_test.go`, `internal/transport/duplicates_test.go`.
- Клиент: prettier PASS, Node-тесты 72/72 PASS, `tsc --noEmit` (TypeScript 6)
  без ошибок.
- Живой прогон в Chromium на реальном Go-сервере с двумя токенами (denis,
  kirill): отметка дубля, «Это разные», подсказки в диалоге, 1440/390,
  светлая/тёмная тема.

На этом историческом этапе NOT-RUN: `next build` и TypeScript 7 (в облачной среде не было доступа к npm за
нативными бинарниками — интерфейс собирался esbuild с заглушками `next/*`);
production; два физических ПК. Перед выпуском выполните на Mac `make check`.

## История подготовки поставки22.10.2026

Root настоящий `make check` исходного4ec5cca **PASS**, включая Go/all/race/vet,
infra155 и Next72/types/format/production build. Прежний NOT-RUN локальной
production-сборки закрыт. Независимое ревью153 потребовало собственный
`deduplication:1` floor при прежней schema4, исправление «Это разные» для эпиков
и исключение недоступных profile assignments из свободной очереди «Команды».
Отдельный автор156 вносит fixes и unit/integration regressions. Отдельный
tester155 начинает новые process/browser tests только после SOURCE APPROVE,
затем тесты также проходят review. Случайный generated Next cache удаляется
из tracked source перед push. Секреты и реальные данные не публикуются.

Перед активацией — immutable clean release, exact Linux проверка/hardened stage,
полный all-project snapshot через reviewed capable updater. Новый permanent
guard запрещает прежний schema4 binary без deduplication до writer start;
DDL/сброса данных/DB rollback нет. Production обновление и final HTTPS пока
NOT-RUN; физический второй ПК и управляемый запуск остаются NOT-RUN.

Независимая actualprocess/Next/Chrome QA155 выявила bridge defect: stdio
не публиковал новые resources и form-style `+` ломал title с пробелами.
Автор156 исправил mirror/percentencoding, добавил regression Unicode/пробелы/
literal `+`/ownSID/foreignSID/tools12; reviewer153 повторно SOURCE APPROVE.
Root финальный `make check` после этого PASS (Go/all/race/vet/build, embedded16,
infra157, Next72/types/format/production build). Product70manifest
`fd749802c85a790808fb0b75203856531aecee33359eab2a1b42d3ac3216b67f`.

Финальная QA155 **PASS22/22**,17screens/sourceunchanged, reviewer153 LOCAL FINAL
APPROVE; updater157 **PASS8/8** и независимый повтор reviewer8/8. Это synthetic
fixtures: actualGo/Next/Chrome и7DB, systemd/UID часть updater simulated;
production/Linux идут отдельным барьером. Доказательства и история FAIL —
WORK153–157. Native stdio MCP не является реальным Claude/Codex executor.

## Итоговая поставка22.10.2026

[Рабочий стенд — «Команда»](https://157.22.207.48/app/team).
Go/Next immutable release **64ed870b749289b545d29cd2ec7813c6266de54d**.
Go Linux SHA256 `c4b25c45de5204dee55f6bb3e85de03be7e1073267d0304aa04c5c816f918b45`;
Next BuildID `PWEoVOCD7jLxpbqh89NjG`, manifest SHA256
`63f17b3e12bfd6f2b20c28713ab0f303faf3f4e162aa17984546087b4e307af9`.
Ветка `codex/dedup` отправлена в origin; main/merge/PR не менялись.

Изменённые области: `internal/similar`, core/store dedup/team и regressions;
projects/transport/bridge/CLI/browser mappings; Next team/duplicates/people/
create-task/overview/task screens/styles; capability/infra guard/tests,
docs/README и `.gitignore`. Полный список — diff отcebcbe5 доrelease64ed870.
Generated Next telemetry cache исключён из tracked дерева; зависимости/lockfile
не обновлялись, пользовательский untracked deploy script сохранён и не запускался.

| Барьер | Фактический результат |
| --- | --- |
| Independent source/author tests | WORK153 APPROVE, product70manifest fd749802… |
| Root full check после всех fixes | Go/all/race/vet/build, embedded16, infra157, Next72/types/format/build PASS |
| Independent local actual Go/Next/Chrome | PASS22/22,17screens, source unchanged; tests separately APPROVE |
| Synthetic updater4→4/fullrestore | PASS8/8 и независимый повтор8/8; actual7DB, systemd/Next/UID simulated |
| Exact final Linux ELF product/updater | PASS10/10+8/8, source/archive/binary binding и ports cleanup |
| Actual hardened Next stage | PASS, owned stagePID/NRestarts0/cleanup/old pointers; runtime checks через reviewed operator |
| Actual capable apply | PASS, full pre-snapshot, old rows3DB preserved, guard floor replacement before writer |
| External HTTPS readonly | PASS9/9,61screens,951browser HTTP200, errors0/prefetch0, own logout/browser closed |
| Finalize/actual runtime/full backup | PASS, journal complete, actual Go argv/ELF и Next ELF/cwd/BuildID, helpers/config/capabilities |
| Final evidence review | WORK153 FINAL APPROVE, обязательных findings нет |

Production содержит default, без дополнительных проектов: сохранены3реальныхDB
(registry/default board/default web). Default+2projects/7DB проверены отдельно
на synthetic local/Linux fixtures. Production task/profile/dependency/duplicate
mutations не выполнялись: в HTTPS — только собственные login/switch/logout и
чтение существующих2tasks, team/similar/MCP и пустого create-dialog без submit.
Business flows проверены local и exact Linux. Нет выдачи тестовых owner labels
за двух физических людей/ПК или session presence за OS process liveness.

Pre-snapshot: `/var/backups/accord-web/projects-20261022T183638Z-cb8c77677ed04bd0b673ea6f60e10c1f`,
manifest SHA256 `4fa528d2bef2cd48bda38b2a533f662ba326d2843279f9db45fc134d64c10de7`.
Final backup: `/var/backups/accord-web/projects-20261022T184141Z-335d09b46e124585b51f0091d8e61128`,
manifest SHA256 `4b6bd4da82a85044ee7b94813afe197a049037f56f65f87f2fd8c279a971dbb6`.
Оба validated наVPS; реальныеDB не скачивались. Final format2/default board4/
web2+registry/history. Authority marker UID999/regular0600/pre-bytes preserved;
helpers root-owned, effective services/timers active, NRestarts0, markers0,
stage7894free. Initial TASK009 apply/DB rollback/bootstrap/reset не использовались.

Private evidence `.local/task012/deploy/`: release/transfer/source proofs,
linux-product-proof-01.json, linux-update-proof-01.json, stage-runtime-proof.json,
process-pending-https-smoke-proof.json, final-runtime-proof.json и
process-complete-proof.json. HTTPS report:
`https-20261022T213702/production-readonly-results.json`, SHA256
`0ad8789bdd96cd916e1c880c652d84dbebb439e096b9b7aa5e3b9508fc350f79`;
approved script SHA256 `400ce6a9ce3c227e4089505684356eae429965e26c2d4c41c3be33ce5d8a41ff`, unchanged.

Ограничения: mode по умолчанию warn/threshold0.6 — эвристика с false positives/
negatives, не гарантия отсутствия всех дублей. Области — метки, не filesystem
locks. Физический второй ПК, реальный Claude/Codex новый сквозной цикл, managed
AI workers, полный V1 и полный a11y audit NOT-RUN. Mobile — Chrome viewport;
visibility input hook эмулирован DOM getter/event, OS lifecycle NOT-RUN.
Backup пока только на прежнемVPS. История PRODUCT FAIL/test-only FAIL сохранена
в WORK153/155/156/157, последующие PASS её не переписывают.
