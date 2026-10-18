# WORK-073 — исправление clock-boundary ожиданий backup harness

- Родитель TASK-003; исполнитель `/root/backupfix073`, роль developer/test-method fix.
- База `7200cc9e1fdfc544dd274ccb8f3991c849112885`, отдельный checkout `.local/backup-method-fixes-003`, ветка `codex/backup-method-fixes`.
- Вход: R070-B01/P2 из review070; APPROVE065 wrapper остаётся действующим, config067 вне scope.
- Профиль developer и процесс/parallel-work Давинчи прочитаны. Named bindings недоступны, используется унаследованная модель. Tandem OFF по прямому указанию пользователя/AGENTS.
- Запись только три `tests/acceptance/backup_process*.py`, эта WORK, новый `docs/backup-method-fix-073.md`; доказательства — ignored `.local/work073/`.
- Source/Go/infra, frozen068, D4, общая TASK, соседний Davinci и performanceQA072 readonly. Git mutations/push/SSH/live/агенты не выполняются.
- Цель: точные timestamp-aware expected identities и содержимое по известному происхождению, независимый oracle UTC hour/day/latest2/budget, deterministic rollover regression без real-clock sleep или подмены wrapper.

## Журнал

- 2026-10-18T20:30:44+03:00 started: назначение получено от root, входы/границы подтверждены, source hashes сохранены; статус в работе.

## Реализация и проверки

- Точный независимый oracle в support: декларативные24 UTC hour/7 dates,
  latest2, logical budget, timestamp/mtime/name; известные hash/content origins.
- Main проверяет exact identities и per-origin schema/rows, учитывает clock
  interval у plan и timestamp реальной новой копии для обеих apply фаз.
- Restore seed закрытой второй версии сохранён отдельно до rotation; удаление
  безусловного second_copy.exists обосновано timestamp/mtime/UUID порядком.
- Детальный контракт, команды и факты — `docs/backup-method-fix-073.md`.
- Старый7200 EXPECTED method FAIL воспроизведён адресно:3PASS/1FAIL,
  `retained_pair_count`, first-hour copy preserved=true; temp удалён.
- Python final14PASS/0.197s. Controlled fixtures проверяют час, midnight,
  24th-hour expiry, ties, budget/future, exact identity/content отказы;
  прежние partial FAIL, hidden output, timeout reap, O_EXCL safeguards PASS.
- Один финальный realprocess run6PASS/0FAIL/0.585529s,10 direct children,
  plan9verified/2ignored, apply5retained/5obsolete; восстановление2tokens,
  revoke-all/reset; workspace удалён. Повторных full runs ради PASS не было.
- 120 исходных файлов неизменны. Wrapper SHA
  `68731d687e9d03ced19142cb4d1089b1b8240eb767d552719090f60372299bba`,
  binary SHA `c7a52a3058b8c09b96b84db7a75dc4cdd6def08a5cce52d6d3000fee21501527`.
- ast.parse3 PASS, diff --check PASS; main256/support206/tests219 строк.
  Промежуточный13-test PASS сохранён отдельно; final14 после расширения
  midnight-case и независимой проверки source estimate — зачётный unit run.
- Evidence `.local/work073/` private0700/0600; final manifest содержит все
  exact5 delivery и evidence hashes без самоссылки.

## Ограничения и передача

Статус: выполнено исполнителем, ожидает независимого review, не accepted.
Linux/cgroup/annual/liveWAL/production/fullGo/race/build/physical2PC NOT-RUN;
полный локальный прогон не объявляется фактическим часовым/midnight наблюдением.
Существующая APPROVE065 политика не менялась. Ни source/D4, ни frozen068,
ни performanceQA072, ни общая TASK/Davinci не изменены. Git mutations,
push/SSH/live/deploy и новые агенты не выполнялись.

## Журнал — завершение

- 2026-10-18T20:35:34+03:00 completed/handed_off: точная Python дельта и checks готовы;
  получатель root для свежего независимого review перед VPS. После записи
  freeze.json исполнитель прекращает все записи; статус не равен APPROVE.
