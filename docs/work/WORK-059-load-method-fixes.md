# WORK-059 — учёт concurrency и partial evidence load055

Родитель TASK-003; developer `/root/loadfix059`, свежий исполнитель.
Статус: выполнено автором, на независимой проверке; этап: REV058-01/02,
только Python method harness. Независимого approve пока нет.
Вход HEAD `674f14c6a1a0b9e44bfde528a7e7e3d0336949dd`, branch `codex/pilot`.
Профиль developer, AGENTS, общий процесс/parallel-work, требования/архитектура,
контракт053 admission, APPROVE048, WORK055/058 и probes058 прочитаны.
Tandem OFF по прямому назначению. Named Claude binding недоступен; модель
унаследована. Независимое ревью будет выполнять отдельный новый агент.

## Границы

Запись только catalog_load/pacing/runtime/workload, узкие infra units,
docs/load-method-055.md, эта WORK и private `.local/evidence/load-fix-059/`.
Go/production, cgroup helper, чужие TASK/WORK, Git commit/push, live/VPS и
Давинчи не изменяются. Baseline049 p95 FAIL остаётся; метод055 не unchanged049.
Legacy, offered1..100, serial setup pacing, waves, deadlines/no catchup/retry,
20/2000/400, operation mix и thresholds50ms/128MiB/1s сохраняются.

## Журнал

- 2026-10-18T19:32:51+03:00 — started; назначение и чистый HEAD подтверждены,
  reviewed055 source после root EOF normalization семантически неизменен.

- 2026-10-18T19:40:31+03:00 — completed; REV058-01/02 реализованы; PASS40, PASS6 probes,
  tiny C method smoke PASS; exact freeze передаётся новому reviewer, без self-approve.

## Проверки и передача

REV058-01: `Ledger.begin/finish` считают actual client roundtrips с finally
cleanup, общий и per-phase peak/active; readiness также использует этот путь.
Это overlap клиента до полного body/error, не server in-flight и не размер
барьера. Snapshot копируется под lock; итоговый отчёт создаётся после drain.

REV058-02: каждый attempt сохраняет raw timing/status/completed/success/error,
включая non200 и transport failure. `submit/as_completed` собирает все futures,
успех позднего peer не теряется из-за чужого исключения. Отдельные per-worker
buffers/waits читаются после join; исходная ошибка предпочтительнее вызванного
ею BrokenBarrierError, все worker error types сохранены. No retry/catchup.
FAIL сохраняет measured wall time, общий experiment elapsed, pacing, план и
actual attempt/completed/success counts; percentiles partial/diagnostic,
thresholds отсутствуют. Error samples не входят в успешный p95; отсутствующие
запросы не достраиваются. Ошибка до measured этапа: elapsed0, started=false.
Cleanup-ошибка после measurement также получает уже собранный контекст.

Изменены7 файлов: четыре `tests/acceptance/catalog_{load,pacing,runtime,workload}.py`,
`tests/infra/test_catalog_load_accounting.py`, `docs/load-method-055.md`, эта WORK.
Размеры Python:233/168/137/102 и273 строк, все≤300. Старые17 pacing cases и
14 cgroup cases сохранены без изменений, добавлены9 meaningful stdlib cases.
Новых E2E нет; tiny run использует ранее approved harness/C binary.
Документ метода обновлён до `catalog-load-055-v2`; история055/058 не переписана.
Manual UI TC N/A: это CLI experiment method; команды/JSON contract описаны
в docs/load-method-055.md. UI/security product behavior не меняется.

- **PASS40**: `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tests/infra -p 'test_catalog_load*.py' -v`,0.477s.
  Synthetic HTTP phases/overlap1vs20/finally cleanup на request/getresponse/read,
  все400/80×5, после успешной волны429/transport/barrier timeout, slow peer,
  legacy partial failure, прерванный rate wait и warmup failure.
- **PASS6** private probes0.319s: те же узкие fixtures отдельно сохраняют реальные
  JSON FAIL reports в evidence, без product process.429 case:6 responses total,
  measured4/4 attempts/completed,3 successful+1 HTTP429; late successful sample
  70.134ms сохранён, elapsed0.098664s, peak2/active0, thresholds отсутствуют.
- **Ожидаемый FAIL baseline055**: финальные9 regression tests прогнаны с private
  copy пяти acceptance modules из HEAD674f14c;1 failure/10 subtest errors.
  Узкая retention regression сначала подтверждает6/6 responses и3 measured200+
  1 measured429, затем падает на отсутствии elapsed_seconds в FAIL report.
  Так воспроизводится дефект, а не только отсутствие нового API.
- **PASS tiny C method smoke**:2 sessions/10 tasks/10 requests each,
  offered100/persisted-defaults;39/39 HTTP200,20 measured,4×5;
  elapsed0.2576035s, actual77.6387rps,p954.069791ms,RSS24.390625MiB,
  populated startup17.821750ms,client peak2/active0; per-phase peak1 кроме measured2.
  C binary `.local/accord/agent-board-1e9025c`, SHA256
  `59c633e286eacfb5c713cfd540857aff2152508c0801c78ec6f58a5a4029e55c`.
  Это Mac arm64 smoke, не target performance PASS; cgroup unavailable.
- `git diff --check` **PASS**. Cgroup unchanged SHA256
  `f3c8253f188b95596e791628dbe637625797859e4440370959eb2c0d4e9b1f27`.
  Go/production delta отсутствует. Freeze/manifest/diff: private
  `.local/evidence/load-fix-059/{freeze.sha256,manifest.json,method.diff}`;
  unit/probes/baseline logs и JSON там же, manifest фиксирует их hashes.
- **NOT-RUN**: target Linux/VPS20/2000/400, D runtime, annual200k/дополнительные
  mutations, Go/full check/race, providers/IDE/второй ПК/TLS/power-loss/live/deploy.
  Historical049 p95 FAIL10s остаётся; admission100rps/burst200/max64 неизменён.

Следующий исполнитель — новый независимый reviewer exact freeze059 той же
доступной модели. Проверить оба REV058 fixes и regression evidence; лишь затем
координатор принимает решение о согласованном целевом опыте. Source055 перед
commit674f14c нормализован root только по EOF catalog_runtime.py, semantic drift
не было. Здесь Git commit/push, общая TASK и чужие WORK не изменяются.
