# WORK-157 — отдельная process QA capable update/restore TASK012

22.10.2026. Tester157, после SOURCE + AUTHOR TESTS APPROVE153. Владение:
`tests/process/task012-update/` и этот WORK. Product source, другие тесты/WORK,
каноническая TASK и production не изменялись. Baseline AGENTS/TASK012/coordination,
updater/guard/snapshot/restore и неизменённые TASK011 adapters прочитаны.

Статус: **local synthetic PASS8/8; новые тесты ожидают независимого review153**.
Это не production PASS и не проверка systemd/Next/root-owner через адаптер.

## Версия и воспроизводимость

Candidate source `69256c419a7a5529e7890633304e54c819b358c0`, отдельный clean-build
бинарник `.local/task012/candidate/agent-board`, macOS SHA256
`889bf7705c31b2ba732b0038d52dbf363de1c8734685a671d3f9e43b82d8ab76`.
Old actual schema4 f2edac9 (product identical e53c18c), SHA256
`daf342c2bb7d714af7d4aceea6910de31eb9e6ddc0751b926ac667284e0bf8ac`.
Old full infra взят из private immutable TASK011 release e53c18c. Старый guard
отдельно pinned SHA256
`c1703c392fc62891e150e23e52cdc68132f05f1c5dad570c53404f4de07b50ed`.

Новый suite использует неизменённые environment/fixture TASK011 импортом.
Инструкция и обязательные env inputs — README suite. No `.git` dependency:
тот же набор допускает exact Linux replay из packaged fixture. Linux old ELF
pinned SHA256 `a03f64d5ff008b7f63023117fb7b1c15c33e5617ce7ac08ffc6440610315269d`.
Linux кандидат всегда требует переданный exact SHA256; macOS также.

## Фактические результаты

Final local run `.local/task012-update-qa/run-1792693333967058000/results.json`:
**PASS8/8**, source_unchanged=true; report SHA256
`a4a1dac182519c2ab05aa263683c21b8f1e3ae9b44793572fa0d81af5a3193b2`.

1. Actual old Go seed: default+2projects,7DB, owner credentials/ACL/epochs,
   repository associations, profiles/revisions/bound session snapshot/live lease,
   tasks/notes, pinned profile assignment и dependency. Opaque history SQLite
   schema1/2/3 + checksum fixtures явно synthetic, не реальные прошлые миграции.
2. Actual updater success/finalize: обе snapshots format2 schema4, точное
   сохранение всех прежних SQL rows семи БД. Независимый file oracle сохраняет
   историю и bytes/UID/mode authority marker. Старый установленный guard принимает
   old executable до update; новый permanent guard заменяет его и отвергает old
   schema4 до storage command под owned startup hold. Новый actual writer ready.
3. Startup injected failure после pointer/storage: fail-closed, оба owned
   процесса остановлены, hold сохранён, exact rows unchanged, explicit resume PASS.
4. Реальный worker os._exit73 после Go pointer и до Next pointer: journal
   activation-started, snapshot/hold и разные pointers наблюдались снаружи;
   explicit resume/finalize PASS, rollback не выполнялся.
5. Чужой manual startup marker: reject без изменения marker/data/pointers.
6. Drift установленного helper: reject без перезаписи неизвестного helper.
7. Известный partial `.guard.py.next-new`: безопасное завершение atomic install.
8. Неизвестный partial: fail-closed, чужие bytes сохранены, pointer/data unchanged.

В success отдельный **actual fullrestore7DB PASS**: новый offline target,
все credentials revoked; board sessions expired, catalog/browser sessions и
эпhemeral actions cleared; membership access_epoch +1. Task history/assignments/
dependencies/profile revisions/notes/idempotency и прежние audit rows сохранены;
добавлены только recovery events с reason=restore. Бинарник не получал токены
production. Boot guard остаётся после reset; source snapshot и active fixture
bytes неизменны. Исторические migration files совпадают побайтно.

## История FAIL

Все private run-* сохранены; поздний PASS их не переписывает.

- run1792693227667822000: FAIL в новом seed — tester пропустил обязательный
  description у task. Исправлен только seed; продукт не менялся.
- run1792693241389820000: FAIL restore oracle — требовал equality events,
  хотя штатный recovery обязан дописать audit. Исправлено на exact preservation
  старых events + bounded explicit restore-only appended events.
- run1792693274028612000: промежуточный PASS8/8. Затем усилены pre-storage
  installed guard gate, независимые marker/history oracle и provenance report.
- Final run1792693333967058000: PASS8/8 усиленного suite.

## Границы и следующий gate

Actual: Go processes, SQLite, operator functions, full snapshots/restore, helper
replacement, abrupt process death/resume. Simulated: systemd lifecycle/introspection,
Next runtime/bundle, ingress, invoking-UID adapter вместо root-owner проверки.
На Linux тот же harness дополнительно проверит настоящий /proc executable/argv.
Exact Linux replay, hardened stage, real services и production HTTPS — **NOT-RUN
этим tester**, их выполняет coordinator отдельно. Второй физический ПК NOT-RUN.
Нет тестовых/source файлов >300 строк. Secrets/productiondata не читались.

Suite6file manifest SHA256
`04d539f492927fb5932ed872314d3031847bda1c89473afb66db997ec6b003ae`:
sorted `sha256 + two spaces + repo path + LF`, README и5Python files.
Передан координатору для independent review153; approval пока не заявляется.
