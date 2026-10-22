# WORK154 — поставка дедупликации и «Команды»

22.10.2026. Координатор; область записи: TASK012, этот WORK и private
`.local/task012/deploy/`. Product исправляет отдельный автор156; source/tests
проверяет153; новые process/browser проверки пишет155 после SOURCE APPROVE.

## Полномочие и checkpoint

Пользователь: «залил новые изменения. залей, обнови стенд и дай ссылку».
Это разрешает push проверенной `codex/dedup` и безопасное обновление прежнего
Accord VPS. Main/merge/PR не требуются. Управляемый запуск этой поставкой не
реализуется; прежние вопросы о worker не относятся к дедупликации.

Исходный source `4ec5ccaeec929136d4aa900869d54b1f8a6c614a`, parent `cebcbe5`.
Read-only VPS preflight подтвердил Go/Next release
`e53c18c9643b9ee086141d0f227ab485f4f88cac`, Go SHA256
`a03f64d5ff008b7f63023117fb7b1c15c33e5617ce7ac08ffc6440610315269d`,
Next BuildID `HHViYVPwDuQx7uj4Uk1YC`, active services/timers, NRestarts0,
inspection/restore markers отсутствуют. Это проверено заново; прежний snapshot
не используется как доказательство текущих данных.

## Проверки и порядок

Root `make check` на source4ec5cca **PASS**: Go fmt/vet/all/race/build,
embedded web, infra155, Next72/tests/types/format и настоящий Next production
build. Прежний NOT-RUN production build закрыт локальной сборкой на Node24.19.
Лог private `.local/task012/deploy/make-check.log`.

Первичный secret-pattern scan66changed files: совпадений0. Это ограниченный
pattern scan, не доказательство отсутствия всех возможных пользовательских
данных. Случайный generated Next cache перед push удаляет автор156 и добавляет
ignore. Credentials/реальные VPS данные не включаются в отчёты или Git.

Новую persistent duplicate semantics до поставки защищает собственный capable
floor; board schema4/coordination/ACL сохраняются. Независимый SOURCE APPROVE,
авторские regression tests, отдельный QA и review новых tests обязательны.

Поставка: clean immutable Go Linux/Next bundle → exact hashes → inactive
transfer → actual hardened stage → full all-project snapshot и штатный
`infra/accord/projects/upgrade.py apply` → независимый readonly HTTPS →
finalize/full backup → actual owned runtime/preservation → final review.
Initial TASK009 apply, DB rollback, bootstrap/reset и старый Go запрещены.
Remote Python только `-B`; unknown recovery artifacts не удаляются.

`deploy-task012.command` оставлен как чужой untracked файл; не исполняется.
Его комментарий об автоматическом rollback не соответствует capable updater.

## Статус

IN PROGRESS. WORK153 **SOURCE + AUTHOR TESTS APPROVE**:69files manifest SHA256
`7b55514bb25f43a13aec24c4d71664d89f771ee272cd55613f347e0eec756290`.
Root независимо повторил формулу и получил тот же hash. Final root `make check`
**PASS** после всех fixes: Go/all/race/vet/build, embedded16, infra157,
Next72/types/format/production build. Лог `make-check-final-source.log`.
Tester155 теперь разрешены новые process/browser tests. Release/push/production
update/final review пока не завершены; результаты дополняются по факту.

## QA нашла bridge defect

Actual native stdio QA155 выявила **PRODUCT FAIL**: mirrorResources не
регистрировал новые team/similar templates, запрос получал MCP -32602.
Добавленный автором156 regression выявил вторую часть: form-style `+` для
пробелов не совпадал с upstream URI matcher. Исправлены exact allowlist и
RFC6570 `%20` forwarding с сохранением literal `%2B`; SID guards/tool12 прежние.
История product/test-only FAIL сохранена в WORK155/156, прежние отчёты не менялись.

Reviewer153 повторно дал **SOURCE + AUTHOR TESTS APPROVE**:70files manifest
`fd749802c85a790808fb0b75203856531aecee33359eab2a1b42d3ac3216b67f`, root
независимо получил тот же hash. Для финальной QA создаётся новый clean candidate;
прежний mac69256c4/889bf770 остаётся историческим, не заменяется на PASS мосту.

Отдельный tester157 updater4→4/fullrestore **PASS8/8**, reviewer независимо
повторил8/8 и одобрил test source/evidence. Сохранены7syntheticDB/ACL/epochs/
profiles/assignments/history; actual Go/SQLite/guard replacement, systemd/Next/
UID simulated и честно отделены от actual VPS stage. Exact final Linux впереди.

## Финальная локальная QA

Новый clean candidate `4742e71c96d41aed17243e04ab3e5171f9d5da1f`, macOS SHA256
`f17b4bb8dfa06dbdd9168687beb79c2fa4ce9b33579f6e58cca86932738bbfb5`.
Root повторный полный `make check` после bridgefix **PASS**: Go/all/race/vet/
build, embedded16, infra157, Next72/types/format/production build.

Tester155 definitive actualprocess/Next/Chrome **PASS22/22**,17screens,
source_unchanged=true, browser responses482 всего (включая намеренные negative
cases, не482 HTTP200). Next BuildID `7cLF6QiHKLNWtrKA9ckFg`, Node24.19/Chrome155.
Report `.local/task012-qa/run-1792693746818/results.json`, SHA256
`eac721307dc8afc0f5ca875668cbb55b47b4ab1c8d48cce8bcb8055aae238b82`.
Reviewer153 **LOCAL QA FINAL + TEST SOURCE/EVIDENCE APPROVE**, hashes434source/
test files совпали. Test8manifest SHA256
`788b6d59bd393641496f4674eb3c6955475a98113cee710f402687e0868b03b3`.
Root и reviewer проверили representative screenshots; blocking visual findings0.
Owned processes закрыты, порты3100/3101 свободны.

Polling подтверждён actual10s claim/release без navigation, actual offline/
resume и project switch. Visibility input эмулирован DOM getter/event: это
проверка ветки hook, не OS tab lifecycle. Физический второй ПК, real AI worker,
полный V1 и полный a11y audit не объявляются PASS. Production readonly script
одобрен отдельно, SHA256
`400ce6a9ce3c227e4089505684356eae429965e26c2d4c41c3be33ce5d8a41ff`.

## Immutable релиз

Release `64ed870b749289b545d29cd2ec7813c6266de54d`, cleanVCS, product diff от
проверенного4742 пуст. Ветка `codex/dedup` отправлена в origin; main/merge/PR
не выполнялись. Финальный pattern scan95changed paths: совпадений0, values не
выводились. Пользовательский untracked `deploy-task012.command` сохранён.

Linux Go SHA256
`c4b25c45de5204dee55f6bb3e85de03be7e1073267d0304aa04c5c816f918b45`,
Next BuildID `PWEoVOCD7jLxpbqh89NjG`, manifest SHA256
`63f17b3e12bfd6f2b20c28713ab0f303faf3f4e162aa17984546087b4e307af9`.
Reviewer независимо проверил Go1.27.2/Linuxamd64/VCSmodified=false и manifests
53Go/1247Next members (regular/no symlink), QA archives604committed files.
Private package/runner связывают revision, manifestSHA и archiveSHA до transfer;
после transfer каждый файл сверяется снова, активные pointers не меняются.
Exact Linux product/update и actual hardened stage ещё отдельные барьеры.

## Фактическое завершение

Exact final Linux **PASS10/10 product +8/8 updater**, actualELF/source604/
archive/revision binding и cleanup. Actual hardened stage **PASS**: reviewed
operator проверил LinuxNode/sharp/bundle, owned transient PID/NRestarts0;
stage JSON фиксирует PID/runtime, не отдельный raw argv dump. После stage
old pointers сохранились,7894free. WORK153 EXACT LINUX + ACTUAL STAGE APPROVE.

Actual capable apply **PASS**: full format2 snapshot до storage/writer,
newdedup guard под owned hold, oldrows3DB сохранены, затем новые Go/Next.
Readonly inspector подтвердил actualGo argv/ELF и actualNext Node ELF/cwd/
BuildID/MainPID; rawNext argv не заявляется, сохранён cmdline hash и effective
launch assertions. Marker UID999/regular0600/pre-bytes preserved, markers0.

Public HTTPS **PASS9/9**,61screens/951browserHTTP200/errors0/prefetch0,
ownlogout/browser_closed=true, source400ce6a9 unchanged. Report SHA256
`0ad8789bdd96cd916e1c880c652d84dbebb439e096b9b7aa5e3b9508fc350f79`.
Final backup format2/default4web2/registry/history validated наVPS, manifest
`4b6bd4da82a85044ee7b94813afe197a049037f56f65f87f2fd8c279a971dbb6`.
Пути обоих snapshots, changed areas, полный результат и ограничения — TASK012.

Finalize **complete**, services/timers active/NRestarts0/7894free/markers0.
Final runtime proof SHA256
`74b1628aea2aa3faec459ba1447b3654fcea7bcdb4fa6c7457a5c85bcb06e600`;
actual process complete proof SHA256
`4d1ea1da53dc31d168088364e78c07c543e5c5f52d5e686b6fcc35a649d603be`.
WORK153 **DELIVERY EVIDENCE FINAL APPROVE**, обязательных findings нет.

**COMPLETE** для поставки согласованной дедупликации/«Команды». Production
содержит default/3DB;7DB относятся только к synthetic fixtures. Source/тесты
release64ed870 после QA не менялись, closing изменения только docs/AGENTS.
Исходные `team.css`755строк и `team.tsx`305строк сохранены как цельные стилевой
слой/экран пользовательского4ec5cca: их механическое дробление не требовалось
для исправления поставки и не меняет verified behavior. Existing create dialog
>300строк отдельно объяснён автором156. Неизвестные runtime/recovery артефакты
не удалялись; пользовательский untracked script сохранён.
