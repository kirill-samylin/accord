# WORK-058 — независимое review frozen load055

18.10.2026; reviewer `/root/loadreview058`, отдельный от автора055.
**REQUEST CHANGES: REV058-01 и REV058-02.** До исправления и нового независимого
review frozen055 не является approved методом целевого опыта. Это заключение
о Python harness; production C/D и исторические APPROVE048/результат049 не
переоткрываются. Performance PASS не выдан.

## Проверенная версия

HEAD `bee0bea063fc3ff9775f558f612a596e2499b8ac`;
C source `1e9025cd6e3122f0adf6165e09845202736d004a`.
Незакоммиченный scope055 сверен по всем7 файлам freeze:

| Артефакт | SHA256 |
| --- | --- |
| `.local/evidence/load-method-055/freeze.sha256` | `760f3d3c31d7522340c35f45663e8f4716ab6325e34b99ea4fff9955177dac16` |
| `manifest.json` | `c5f62341252b3520476ed219705667e3a00be2e492b56da52a99ff0c8de81822` |
| `method.diff` | `d7f1674588a16a25868e4c483b1b7979ed34988a6bae90386cb249479b3aa400` |
| Неизменённый `catalog_cgroup.py` | `f3c8253f188b95596e791628dbe637625797859e4440370959eb2c0d4e9b1f27` |
| Tiny-smoke C binary | `59c633e286eacfb5c713cfd540857aff2152508c0801c78ec6f58a5a4029e55c` |

Все7 source/doc hashes, manifest/diff и5 evidence entries manifest совпали.
Точное имя исходного freeze — `freeze.sha256`; `final-freeze.sha256` нет.
Dirty `docs/TASK-003-working-service.md` не входит в freeze055 и не является
его drift. Новых Go-изменений в текущем diff нет. Итоговый source freeze
повторно проверен после probes, source/tests автора не редактировались.

## Блокирующие замечания

### REV058-01 / P2 — отсутствует достигнутая HTTP concurrency

Места: `tests/acceptance/catalog_pacing.py:111–118`,
`tests/acceptance/catalog_runtime.py:48–59`,
`tests/acceptance/catalog_load.py:115–127`.
Контракт053:287 требует публикации actual concurrency. Сейчас `wave_size`
равен числу участников барьера, `wave_count` и offsets характеризуют release;
активные HTTP-вызовы нигде не считаются. Документ корректно предупреждает,
что burst opportunity не гарантирует20 server in-flight, но само измерение
требуемой достигнутой одновременности отсутствует и в real smoke JSON.

Независимый synthetic probe запускает настоящий `Participant.request` с
немедленно отвечающим fake connection:400 measured calls,20 workers/wave,
наблюдённый независимым счётчиком peak active HTTP=1. Возвращённый отчёт
содержит только `wave_size=20`; такой же shape возможен при ином overlap.
Это не результат реального сервера, а воспроизводимое доказательство того,
что численность барьера не заменяет отсутствующую метрику.

Нужно: измерять достигнутый overlap клиентских HTTP attempts в границах
request→полное body/error, с потокобезопасным enter/finally-exit, публиковать
peak и его scope отдельно от planned wave size; сохранять его и на FAIL.
Это не следует называть server in-flight без серверного evidence.
Unit coverage должна различать последовательные мгновенные calls и реально
перекрывающиеся calls, включая error/transport failure и сброс active в0.
Нативные admission100rps/burst200/max64 не менять.

### REV058-02 / P2 — FAIL теряет уже измеренные задержки и длительность

Места: `tests/acceptance/catalog_runtime.py:54–59`,
`tests/acceptance/catalog_workload.py:37–52`,
`tests/acceptance/catalog_load.py:174–179`.
HTTPFailure возникает после вычисления elapsed и записи status, но latency
не возвращается. Любая ошибка worker прерывает `pool.map`, локальные records
успешных запросов остальных workers тоже не возвращаются. В FAIL JSON остаётся
только ledger/error_type: нет elapsed, measured count, частичных latency,
pacing context и фактического RPS. Это лишает неуспешный опыт предусмотренной
в назначении058 диагностики duration и slow/error samples. Сам FAIL честный;
ложного threshold PASS здесь нет, а HTTP counts/statuses сохраняются.

Независимый probe:2 warmup200, затем3 measured200 и1 measured429; все6 ответов
учтены, повторов нет. Все measured responses искусственно задержаны на10ms.
Итог FAIL сохраняет6/6 responses и429, но полностью теряет эти latency и
длительность. Ошибка также может проявляться как `BrokenBarrierError` у peer;
по HTTP ledger429 всё же диагностируется.

Нужно: собирать частичные samples/duration/pacing в доступном main состоянии
или structured failure result после завершения уже запущенных calls, включая
время non200; публиковать counts/status/error и явную неполноту workload.
Не подменять error latency успешным sample, не рассчитывать положительный
threshold verdict для FAIL и не достраивать отсутствующие requests до400.
Unit: ошибка после первой успешной волны, slow completed peer,429 и transport
failure; ранее завершённые measurements должны оставаться в FAIL artifact.

## Фактические проверки

- **PASS31**: независимый повтор `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest
  discover -s tests/infra -p 'test_catalog_load*.py' -v`, Python3.14.5,0.132s.
  Это17 pacing/accounting +14 существующих cgroup units.
- **PASS5 private probes**: воспроизведение обоих замечаний; исключение clock
  в barrier action ломает ожидающих; CancelledError worker отменяет оба
  rendezvous; реальные задержанные fake calls остаются в successful samples,
  measured wall time и slow-wave cadence.5 probes/4.140s.
- **PASS tiny method smoke**: собственный изолированный C child на Mac arm64,
 2 sessions/10 tasks/10 requests each,100rps/persisted-defaults;
 39 attempts/39 HTTP200;20 measured,4 каждого типа. Elapsed0.260361s,
  p952.244292ms. Это smoke механизма, не target performance evidence.
- Авторские unit.log и final tiny smoke JSON/log совпали с manifest; их claims
  проверены чтением и отдельным повтором выше. `git diff --check` PASS.
- Декомпозиция разумна:187/118/125/55 строк у Python modules,288 у новых units;
  новых смысловых дублей/необоснованного обхода типов не обнаружено. UI N/A.
- Barrier abort сначала устанавливает Event, затем abort двух barriers:
  реальный rate sleep прерывается без lock inversion. Error/deadline/slow
  scenarios, no catch-up, rollback clock, аргументы1..100/2..200, legacy0,
  persisted-defaults argv и400/80×5 подтверждены source+targeted checks.

Часы `time.monotonic`, Barrier action/timeout/broken/abort и Event.wait
сопоставлены с первичными [Python time](https://docs.python.org/3.14/library/time.html#time.monotonic)
и [Python threading](https://docs.python.org/3.14/library/threading.html#barrier-objects).
Инъецированные часы тестов не используются для HTTP latency timer; fake clock
проверяет cadence, реальные threads — rendezvous. Это не измерение VPS.

## Рекомендации P3 и ограничения

Новых P3 отдельно от двух блокеров нет. Нельзя исправлять отчётность изменением
limiter, TTL defaults или p95≤50ms/RSS≤128MiB/start≤1s. Unpaced legacy1h
сохранён; opt-in persisted-defaults только опускает два serve TTL flags.
D runtime compatibility этим C smoke не доказана.

**NOT-RUN:** Linux/VPS20/2000/400, annual200000 events, обычные дополнительные
mutations на годовом объёме, D process, full Go/check/race, power-loss/reboot,
providers/actual IDE clients/второй ПК, TLS, deploy/live. Effective cgroup limits
на Mac unavailable. Baseline049 остаётся latency FAIL. Метод055 не является
unchanged049; завершение smoke не означает полный API/V1/performance PASS.

Evidence: ignored `.local/evidence/load-review-058/`:
`freeze-verification.json`, `unit.log`, `probes.py`, `probes.log`,
`probe-observations.json`, `c-tiny.json`, `c-tiny.log`, `evidence.sha256`.
Следующий шаг — отдельный автор исправляет два блокера в рамках055, фиксирует
новый freeze; затем новая независимая проверка до целевого опыта.
