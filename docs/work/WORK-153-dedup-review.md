# WORK-153 — независимое ревью дедупликации и «Команды»

22.10.2026. Reviewer, та же доступная модель, отдельный от автора реализации.
Область записи: только этот WORK. Исходный кандидат `4ec5ccaeec929136d4aa900869d54b1f8a6c614a`
на `codex/dedup`, baseline `cebcbe5`. Managed launch не реализовывался и WORK
управляемого запуска не создавался: TASK012 занята отдельным поручением dedup/team.

Статус исходного кандидата: **REQUEST CHANGES**. Ниже перечислены findings,
переданные координатору и отдельному автору исправлений до source APPROVE.
Production не изменялся; `deploy-task012.command` не запускался.

## Источники и метод

Прочитаны AGENTS, TASK009/010/011/012, требования, архитектура, API,
multi-project, profiles, coordination, control-web-architecture. Проверены
фактические changed source/test chains core/store/projects/transport/bridge/web/
Next, а не только описание TASK012. Для React/Next применён
`/Users/shagi/.codex/skills/vercel-react-best-practices/SKILL.md`: границы
запросов/отмены, ключи project scope, входящие payload и mutation ownership.

## Findings исходного 4ec5cca

1. **P1 — нет capability floor новых durable semantics.**
   `cmd/agent-board/main.go:45`, `infra/accord/projects/guard.py:42–54` по-прежнему
   разрешают прежний schema4/coordination1 executable. Новые `duplicate_override`
   и creator fields сохраняются в JSON, а `duplicate_of` меняет смысл состояния.
   Старый executable не знает этих полей; `loadEntities` игнорирует unknown JSON,
   `saveBoard/saveEntities` переписывают все tasks даже при обычном transact/read.
   Поэтому допустимый старый binary теряет новые поля и semantics. Нужен явный
   постоянный `deduplication:1` floor до активации, backup/restore/update guards
   и проверка отрицания старого executable. DDL/schema bump сам по себе не нужен.

2. **P2 — «Это разные» для эпиков всегда отклоняется.**
   `internal/store/dependencies.go:48`,
   `web/src/features/duplicates/duplicate-pairs.tsx:99–110`:
   pairs включают epic, UI посылает `related`, а Store.Link разрешает связанные
   задачи только при Type=task. Минимальный fix: explicit epic→epic related с
   пустым condition/generation0 под прежним administrative gate, без расширения
   blocks/contract_of или межтиповых связей. Нужен regression фактического dismissal.

3. **P2 — восстановление эпика сохраняет duplicate_of.**
   `internal/store/transitions.go:21–22,79–86,154–166`:
   новая очистка связи находится только в task→ready ветке. Epic cancelled→open
   делегируется отдельно, получает open со старой duplicate link/баннером и
   перестаёт участвовать в поиске пары. Повторный mark может добавить вторую
   outgoing duplicate link. Нужна общая detach semantics при восстановлении
   задачи и эпика, с сохранением прочих links/history и regression.

4. **P2 — «Ждут исполнителя» включает заведомо невыдаваемые назначения.**
   `internal/store/team.go:107–112` проверяет ready/unleased/dependencies/open epic,
   но не `assignmentAvailable`. После edit/archive профиля Task011 запрещает
   любой новый claim. Team показывает её как готовую и увеличивает total.
   Использовать тот же predicate, что scheduler, не отдельную ослабленную копию.

5. **P2 — claim warnings игнорируют off и могут пропустить живого исполнителя.**
   `internal/store/duplicates.go:238–248`: mode отброшен, потому `next` выдаёт
   duplicate_check при off. Кроме того, similarTo сначала выбирает top5 всех
   candidates, и только затем фильтруется LiveLease. Пять более ранних/сильных
   ready/recent done скрывают шестую уже выполняемую похожую работу. Фильтр live
   должен идти до limit; area_overlap остаётся отдельным предупреждением.

6. **Generated cache в Git.**
   `web/.next/cache/_events_20421_9cc41a7240a4bfe6.json` — 216-byte JSON array,
   исследован только по shape/названиям ключей без вывода значений. Это generated
   telemetry artifact, а не source. Нужны удаление tracked cache и ignore `.next/`;
   наличие секрета или реальных пользовательских данных по этой проверке не заявляется.

## Поставка и достаточность проверок

`deploy-task012.command` в исходном виде не является approved delivery gate:
push идёт первым (строки14–15), отсутствуют полный make check/source APPROVE/
независимая QA/exact Linux update rehearsal до apply. Комментарий строки4 про
rollback противоречит принятому roll-forward контракту. `set -e` лишь прекращает
следующие команды: он не доказывает откат, сохранность, исправное hold/recovery
или возможность безопасного повторного apply. Удаление release dirs строкой27
не является допустимым восстановлением неизвестного частичного результата.
Сценарий не запускался и не предлагается пользователю для запуска. Использовать
reviewed capable `upgrade.py` stage/apply/resume/finalize после отдельных барьеров.

Заявленные автором unit/race/tsc/esbuild и Chromium с заглушками next/* полезны,
но не доказывают standalone Next build/runtime и package provenance. Независимая
product/browser/update QA TASK012, exact Linux и реальная production поставка
пока NOT-RUN; прошлые TASK011 PASS не переименовываются в TASK012 PASS.

## Положительные свойства

Create check выполняется внутри прежней atomic mutation; ответ warnings
сохраняется в durable replay. Mark duplicate сохраняет live lease и проверяет
administrative права, состояние и цикл; восстановление не стирает audit.
Новые project service reads/writes идут через текущие ACL/epochs и session guards.
MCP остаётся12tools, ресурсы требуют собственную board session. React выводит
пользовательский текст через экранирование; project links/queries сохраняют scope.
QueryBoundary remount по routeScope защищает новое useSimilar placeholderData от
переноса между проектами. Team footnote честно отделяет session от OS process.

## Следующая проверка

Автор исправлений отдельно закрывает findings; reviewer перечитывает diff и
regressions, повторяет подходящие unit/integration. Только после source APPROVE
отдельный tester пишет/запускает process/E2E; новые тесты также проходят review.

## Повтор source — core и guard

WORK156 исправил пять core findings без ослабления scheduler/auth. Прочитан
полный diff и `internal/store/dedup_release_test.go`: epic related ограничен
epic↔epic/empty condition/generation0 и прежними правами; restore убирает только
собственную outgoing duplicate link; Team использует `assignmentAvailable`;
live candidates фильтруются до limit5, off сохраняет только bounded area advice.
`deduplication:1` обязателен для нового permanent guard и candidate capabilities.
Restore/backup/stage используют тот же guard, upgrade ставит helper под owned
startup hold до активации. Служебный authority marker bytes/UID/mode не меняются.
Tracked cache удалён из индекса, `.next/` добавлен в ignore.

Reviewer независимо повторил store/web/transport/bridge/similar — PASS;
projects/cmd — PASS; guard/ingress unit9/9 — PASS. История двух первых запусков
projects/cmd — FAIL в fixture setup `unsafe web data parent`: штатный Mac TMPDIR
идёт через symlink `/var`; один лишь umask077 не помог. Повтор с существующим
repo-private realpath `.local/toolchains/tmp` прошёл. Source storage guard
не менялся, этот результат не является FAIL продуктовой дедупликации.

Core/guard source slice **APPROVE**. Общий SOURCE APPROVE пока ожидает узкий
frontend fix восстановления strict create после authoritative concurrent
`DUPLICATE_SUSPECTED`, когда preview не показал candidates. Неизвестный transport
outcome должен по-прежнему замораживать исходную отправку; это отдельный случай
от известного отклонения сервером. Runtime QA/поставка по-прежнему NOT-RUN.

## Общий SOURCE APPROVE

Повторный frontend review: `useAction.errorCode` задаётся только для known
rejection с освобождённым pending, очищается при dispatch/success; unknown
outcome оставляет frozen retry/locked fields. `CreateTaskDialog` принимает
authoritative DUPLICATE_SUSPECTED независимо от preview, требует explicit consent
и причину, сбрасывает согласие при правке draft. После следующей validation
ошибки уже выбранный force сохраняет доступную форму причины и checkbox;
нет скрытого зависшего consent. Закрытие диалога unmounts весь draft.

Reviewer team/actionState10/10 PASS на Node24.19.0; existing module-type warning
не является FAIL, зависимости/настройки package ради его подавления не менялись.
Автор156 web72/types/format PASS. Реальный concurrent strict browser scenario,
UTF-8 reason validation и uncertain exact retry — обязательные отдельные QA
сценарии после этого source approval, не объявляются проверенными по чтению кода.

**SOURCE + AUTHOR TESTS APPROVE** после исправлений WORK156, mandatory source
findings закрыты. Product/author-test manifest:69files, SHA256
`7b55514bb25f43a13aec24c4d71664d89f771ee272cd55613f347e0eec756290`.
Baseline `cebcbe5b1955ffc95fb74aabc90c2233d6c83ae3`; набор — union
`git diff --name-only BASELINE` и `git ls-files --others --exclude-standard`,
отфильтрованный на `.gitignore`, `cmd/`, `internal/`, `infra/`, `tests/infra/`,
`web/src/`. Manifest — sorted строки `sha256 + two spaces + relative path + LF`,
для отсутствующего пути `DELETED + two spaces + path + LF`. Docs и пользовательский
deploy script в этот product manifest не входят. Cache deletion и ignore
проверены отдельно. UI file SHA256: useAction
`0b6c5fff335c9dfd5469e8aefae2e042830e1b6ec5102aa2cd15a0ca0990ba64`,
CreateTaskDialog
`09ad8ab921851b16fbaaad948751d3fee44de78a92b5abd32faf5ac937e25187`.

Разрешён следующий барьер: отдельный tester process/browser TASK012 и review
новых тестов. Root полный final make check, immutable Linux/update/restore,
actual stage/guard replacement, production full backup/HTTPS/preservation и
итоговый evidence review остаются отдельными обязательными барьерами.
Это не FINAL APPROVE и не разрешение запускать исходный deploy-task012.command.

## Delivery source review

Source checkpoint `69256c419a7a5529e7890633304e54c819b358c0`; root независимо
совпал с69manifest и сообщил final make check PASS (Go/all/race/vet/build,
embedded16, infra157, Next72/types/format/настоящий build). Это сообщение root,
а не новый полный независимый прогон reviewer; его узкие повторы описаны выше.

Прочитаны private `.local/task012/deploy/{build_release,transfer,activate_release,
collect_runtime,transport_safe,run_https}.py` и фактические release/updater chains.
Go build требует clean source и pinnedGo1.27.2, VCSmodified=false/exact revision;
Next standalone пакет связывается с той же revision и pinned LinuxNode SHA.
Transfer проверяет exact regular files/hashes, не следует symlink/path traversal,
пишет только новый inactive release; existing/partial target не удаляется молча.
`activate_release.py` вызывает именно capable `upgrade.py`; apply сохраняет
owned startup hold, останавливает обоих writers и делает full all-project
snapshot до helper installation, переключения и storage validation. Новый
semantic guard установлен под hold до старта writer; ACL/schema4 сохранены.
Failed apply остаётся blocked-roll-forward; старый Go/DB не выбираются.
Final проверяет service UID999/0600 marker и bytes из pre-snapshot, actual Go
ELF/argv/PID, pointers, helpers, effective configuration и новый full backup.
Remote Python через `transport_safe.py` использует `-B`, local imports также
запрещают bytecode; unknown artifacts не удаляются. Bootstrap/reset не вызываются.

Delivery source slice build/transfer/stage/apply/finalize **APPROVE**.
Private helper hashes (до actual выполнения): build_release
`0e5dd3bb9615ca28ba62bc500dc15969325ccd7aec6b479adaf48822f02a87dd`, transfer
`8e5337725c449e157f1f0d39d8d375670e73b3a8695f53e8b9610188d830a6c9`,
activate_release
`516691ed297dab797edc167cdb81676b2b41767eb25350cf9855d3052326993d`,
collect_runtime
`d7b5b4f42f6e8e196d1ef013c24825db998712e0ce7eee556e9fb670a33b6a83`,
transport_safe
`b487f76bbbf5fa5413fe95abd807a53361a8f7ed801738742a5abad08ffa19f7`.

HTTPS runner пока запускает TASK011 readonly script, что полезно как regression,
но не проверяет новый `/team` и dedup projections. До TASK012 actual HTTPS нужен
отдельно reviewed новый/расширенный tester155 readonly script и точная привязка
runner к нему. Ни script source review, ни старый HTTPS PASS не являются новыми
runtime PASS. Финальный evidence review остаётся впереди.

Отдельно reviewed private `package_qa.py` и `linux_qa.py`. Первое замечание
reviewer: committed fixture bytes сверялись с HEAD без связи с release revision.
Координатор добавил HEAD==release-proof revision и per-kind source proof
(revision, manifestSHA, tarSHA, count). Runner проверяет всё до remote transfer,
создаёт новый exclusive private synthetic root, сверяет exact archived files,
старый e53 ELF SHA и новый candidate SHA. Runtime передаёт только synthetic
fixtures, Node из candidate bundle, Python-B; report требует PASS/exit0 и
свободные3100/3101 до/после. Failure сохраняет private diagnostics и не
выполняет production operations. Actual suites требуют отдельного test review.
После binding fix wrappers **SOURCE APPROVE**: package_qa SHA256
`eedcc9ee42dd18c049394d014f5e7a55791b1da1390808a8cccb371751fe5f19`,
linux_qa
`3a3f43bdc3ddee69f3970c6ce971242346392f1b28862fa446376fe1afa841c7`.

## Независимая QA обнаружила bridge defect

QA155 actual process run `.local/task012-qa/run-1792693223830/results.json`
выявил **PRODUCT FAIL**, подтверждённый reviewer по source:
`internal/bridge/resources.go:27–31` не регистрирует team/similar templates в
mirrorResources, хотя нижний readNotes host allowlist уже их принимает.
Actual stdio клиент получает -32602 Resource not found до session-bound handler.
Прямой HTTP MCP тест не обнаружил эту неполноту. Это P2 обязательный functional
finding, не test-only ошибка. Bridge SOURCE APPROVE приостановлен до отдельного
author fix/regression; прежний69manifest не финальный. Остальные review slices
сохраняются; production не переключается ради диагностики.

Отдельно reviewed новый `tests/browser/task012/production-readonly.cjs`:
**SOURCE APPROVE** до выполнения. Browser allowlist разрешает только собственные
login/logout/project open; native calls только GET и MCP read methods. Проверяет
team через web/native/resource, existing-only similar, прежние profile/coordination
screens, task digest/nonrenewal, пустой create dialog без ввода/submit, cleanup.
Page screenshots/error reports private, token value не выводится; reports
сохраняют script hash/unchanged, cleanup не скрывает исходную ошибку.
Script SHA256
`400ce6a9ce3c227e4089505684356eae429965e26c2d4c41c3be33ce5d8a41ff`,
support TASK009 SHA256
`f3c2313e4b169e266f5bb3945ebc82049a892fd3bf80cd16a4c312eda07265cf`.
HTTP readonly не заменяет actual stdio QA; новый script запускается только после
исправления и повторного source/local/Linux approval.

## Updater157 — test source и независимый повтор

Новые6файлов `tests/process/task012-update/` прочитаны полностью, включая README,
Environment наследует неизменённый reviewed TASK011 adapter. Проверены actual
old guard до update, installed new guard перед storage под hold, negative old
schema4, exact7DB rows, byte/history/marker oracle, crash73/resume/fail-closed,
manual/drift/known/unknown partial и новый offline fullrestore с auth reset.
Полный source manifest совпал с заявленным:
`04d539f492927fb5932ed872314d3031847bda1c89473afb66db997ec6b003ae`.

Report tester157 `run-1792693333967058000` SHA256
`a4a1dac182519c2ab05aa263683c21b8f1e3ae9b44793572fa0d81af5a3193b2`
проверен; reviewer независимо выполнил тот же suite: **PASS8/8**,
`.local/task012-update-qa/run-1792693420987480000/results.json`, SHA256
`874cc63df898a6e8857a272796de62d6c427111153dbe4d7a0697f1c9d2f5b48`,
source_unchanged=true, fullrestore7DB/authreset/history/source bytes PASS.
Оба относятся к source69256c4/macOS candidate889bf770; exact финальный Linux
после bridge fix проверяется отдельно. Systemd/Next/UID simulation остаётся
объявленной границей и не превращается в actual production evidence.

**UPDATER TEST SOURCE + LOCAL EVIDENCE APPROVE**. Исторические test-only FAIL
сохранены в WORK157 и private run-*; их поздний PASS не переписывает.

## Bridge fix — повтор SOURCE APPROVE

WORK156 добавил exact team/similar templates. При повторном review выявлена
вторая часть: `query.Encode()` превращает пробелы title в `+`, что не совпадает
с RFC6570 resource matcher. Автор воспроизвёл actual product FAIL своим новым
integration test и нормализовал outbound spaces в `%20`; literal plus остаётся
`%2B`, SID guards/URI projection не меняются. Положительный test title теперь
«Экспорт задач JSONL + CSV». Один последующий reviewer run был FAIL test-only:
fixture использовала PathEscape и оставляла literal `+` в query; исправлено на
QueryEscape + space normalization. Финальный reviewer bridge suite **PASS**.

Новые unit проверяют templates ровно по одному, отказ до Describe, omitted/own
SID reads, typed team/similar contents и original returned URI, foreign/repeated
SID отказ и12tools. **SOURCE + AUTHOR TESTS APPROVE восстановлен**:
product70files manifest SHA256
`fd749802c85a790808fb0b75203856531aecee33359eab2a1b42d3ac3216b67f`
по прежней формуле. Прежний69256c4/69manifest не является финальным candidate.

Tester1557newfiles также прочитаны: actual Go/SQLite/CLI/native HTTP/stdio MCP,
контролируемые synthetic identities, strict concurrency/lease/restore/off/ACL,
actual Next+Chrome и unknown-result replay; production readonly утверждён
отдельно выше. **TEST SOURCE APPROVE** на SHA256
`df43cdb88c2f7129f0e341dd64e1852a77fc297df0491cac7d9ae5ff570cfbd4`.
Root runner теперь вызывает именно TASK012 readonly; SHA256
`a15df00d3e0a8582f7177f12fecc85db8f61b748e3d372d04155db6ae38c17d6`.
Окончательная local evidence ещё ожидается. Reviewer запросил tester-only
покрытие заявленного10s team polling/visibility/offline/resume и A→B browser
isolation: прежние cases обновляли страницу навигацией, что не доказывает этот
новый hook. Последующие changes тестов требуют нового review/hash.

Reviewer дополнительно просмотрел новый `polling.cjs`: actual native claim/release
меняют открытый Team DOM через реальные10s GET без navigation; actual browser
offline останавливает запросы, online возобновляет; visibilityState/event —
явно заявленная input simulation, не OS tab proof. Session expiry сохраняется,
projectB не показывает A labels/work. Parsed stdio test теперь проверяет точные
task keys с кириллицей/пробелами/C++ вместо одного contents.length.

Первый frozen local4742 PASS22/22 report
`.local/task012-qa/run-1792693626921/results.json`, SHA256
`4814637e01a61b47cd6a9e0fe32a1f3b67d14d3fa283e5923eb7a98d0bf75996`,
source_unchanged=true,17screens/482browserresponses проверен как промежуточный
PASS. Reviewer визуально посмотрел Team1440light/390dark и create390: читаемые
карточки/дубли/формы, без observed horizontal overflow. Mobile остаётся viewport.
Затем tester добавил actual epic related→unlink→2×duplicate/restore с сохранением
отдельной related и повторным появлением пары. Этот узкий delta **TEST SOURCE
APPROVE**, финальные8files SHA256
`788b6d59bd393641496f4674eb3c6955475a98113cee710f402687e0868b03b3`;
окончательный frozen report ожидается.

Исключение300строк: этот WORK сохраняет историю независимых findings, actual
FAIL, disposition и отдельных source/test/runtime gate. Аудит append-only
продолжения оставлен одним документом, чтобы старые approvals не выглядели
итоговыми после изменения candidate; product source этим не усложняется.

## Окончательная local QA — APPROVE

Frozen final report `.local/task012-qa/run-1792693746818/results.json`, SHA256
`eac721307dc8afc0f5ca875668cbb55b47b4ab1c8d48cce8bcb8055aae238b82`:
**PASS22/22**, source_unchanged=true. Reviewer сверил все434captured source/test
file hashes с текущими bytes — exact match.8test manifest788b6d59 выше совпал;
product70manifestfd749 не менялся. Candidate
`4742e71c96d41aed17243e04ab3e5171f9d5da1f`, macOS SHA256
`f17b4bb8dfa06dbdd9168687beb79c2fa4ce9b33579f6e58cca86932738bbfb5`.
Actual local Next BuildID `7cLF6QiHKLNWtrKA9ckFg`, Node24.19.0,
Chrome155.0.8059.86.17screens; reviewer дополнительно посмотрел finalTeam1440dark
и detail390light, blocking visual findings нет.3100/3101 independently free.

482 — число всех browser responses, включая намеренно проверенные400/409/503;
оно не объявляется482успешных200. Actual final includes stdio team/similar
space/literalplus,10s polling/pause/resume, strict late concurrency/reason recovery,
unknown committed exactretry, epic2×restore, viewer/ACL/project boundaries.
OS visibility input остаётся simulation; physical secondPC/fullV1/workers NOT-RUN.
Root сообщил полный final bridge makecheck PASS в `make-check-final-bridge.log`.

**LOCAL QA FINAL + TEST SOURCE/EVIDENCE APPROVE**. Можно готовить immutable
release/exact Linux QA и actual hardened stage; это ещё не FINAL delivery approval.
