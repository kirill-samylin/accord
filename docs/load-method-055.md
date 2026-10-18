# Catalog load055: optional offered-rate method

18.10.2026. Текущий method `catalog-load-055-v2` (исправления059 к frozen v1);
реализация метода, не performance fix. Исходный055/v1 и review058 сохранены.
Нужен отдельный независимый approve перед целевым Linux/VPS опытом.
Исторические APPROVE048 и unpaced baseline049 не переписываются.

## Режим и сценарий

Без `--offered-rps`, либо при явном `--offered-rps 0`, сохраняется legacy:
unpaced describe/seed, warmup каждого worker и один общий стартовый барьер,
после которого worker выполняет свой цикл независимо. Положительный сценарий,
размеры, очередность операций, TTL1h и latency timer сохранены. При ошибке
warmup барьер теперь немедленно ломается, вместо ожидания оставшихся workers.
Новый отчёт добавляет метаданные и счётчики также для legacy.

`--offered-rps 1..100` включает finite rate (дробные значения допустимы).
Describe, создание epic/tasks и warmup дозируются одним общим monotonic pacer
с интервалом не меньше1/rate. Readiness health/ready не являются admission API
и не дозируются; их4 HTTP-ответа отдельно учитываются. Token bootstrap — CLI.
После populated restart warmup выполняется последовательно; затем создаются
ровно20 worker threads для20 sessions в default сценарии.

Измерение:20 волн, в каждой все20 workers допускаются к одному HTTP-вызову.
Барьер до и после каждого вызова не допускает опережения следующей волны
быстрым worker. Интервал между фактическими release ≥sessions/rate, то есть
≥200ms при20/100; предыдущая slow wave не даёт накопить catch-up credit.
Первая волна также ждёт sessions/rate от последнего serial warmup release.
Это burst opportunity для20 одновременно готовых клиентов; OS/socket scheduling
не гарантирует тождественное время отправки или20 server in-flight.

Оба барьера имеют timeout35+sessions/rate секунд. Ошибка любого worker,
включая non200/429, ломает оба барьера и прерывает sleep через Event.
HTTP socket timeout30s сохранён. Rollback инъецированных monotonic часов даёт
ошибку; wall clock не используется. Retry, resend, отбрасывания slow samples
или адаптивного снижения нагрузки нет. Ошибка даёт nonzero exit и отчёт FAIL;
у FAIL отсутствуют положительные threshold results. Уже запущенные calls могут
штатно завершиться перед cleanup, их ответы также считаются.
Отчёт замораживается только после завершения всех futures; ошибка одного
worker не выбрасывает observations остальных. Причинная HTTP/transport ошибка
и ошибки barriers сохраняются отдельно как `workload_errors`, без текста
ответов, payloads или токенов. Новые measured calls после замеченной отмены не запускаются.

20 sessions,2000 tasks и400 measured requests остаются defaults. Equal mix:
get, tasks(limit20), events(limit20), stats(owner), heartbeat — по80.
В отличие от legacy, каждый wave синхронизирует также тип операции, поэтому
результат055 нельзя объявлять повторением049 при другой скорости сервера.
Аргументы размеров нужны для method smoke; меньший smoke не заменяет default.
Requests-per-session, не кратный5, сохраняет legacy round-robin с остатком,
а не обещание совершенно равных counts для произвольного override.

Метод принимает2..200 sessions в paced режиме:200 — bound размера burst,
а не новый product limit. Native admission100rps/burst200/max-inflight64
не изменяется. При>64 готовых sessions возможен штатный503, при429/503 весь
опыт FAIL;200sessions не обещают успешную нагрузку. Legacy>200 не запрещён
новым методом. Быстрый unpaced400 burst проверяется отдельно:429 может быть
правильным защитным поведением, paced result не объявляется его PASS.

## Конфигурация и безопасный запуск

`--persisted-defaults` явно убирает только `serve --session-ttl 1h --task-ttl 1h`.
Token-create сначала создаёт private DB с15m bootstrap defaults; serve
использует их без SQL patch/config mutation или попытки угадать capability
по версии. Это соответствует D/schema3 explicit-flag conflict policy.
Без флага прежний C1h режим остаётся. На D он должен честно дать startup FAIL
при конфликте; автоматического fallback нет. Проверка D process здесь NOT-RUN.

Owner и machine уникальны у каждой synthetic identity; каждый token-create
по-прежнему локальный token admin. Значения токенов остаются в private temp
files и памяти, в отчёт не попадают. Процесс работает только с owned child,
loopback ephemeral port и temporary DB; существующий service URL не принимается.
Temp cleanup, token modes/umask и approved cgroup helper сохранены.

15m session TTL не продлевается во время seed. Слишком медленный seed или
намеренно низкая offered rate могут привести к expiry и FAIL — это ограничение
метода, не повод молча продлить TTL/исключить запросы/подменить данные.
После approve055 и approved product-source следующий исполнитель запускает,
подставив свои private binary/output paths:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 tests/acceptance/catalog_load.py \
  --binary /private/path/approved-binary \
  --output /private/new-result.json \
  --offered-rps 100 --persisted-defaults
```

## Отчёт и границы доказательств

- `method_version`, `offered_rps`, `persisted_defaults` фиксируют способ запуска.
- `elapsed_seconds`: фактическое wall время measured workload, включая
  pool/barriers/waits; legacy включает свой threaded warmup, paced исключает
  sequential warmup. При FAIL сохраняется время до завершения всех workers,
  включая поздний peer. До начала measured этапа это0 с
  `measurement_started=false`; `experiment_elapsed_seconds` в FAIL включает
  также подготовку и cleanup. `actual_measured_rps` = completed measured
  responses/это время, `actual_measured_attempt_rps` = attempts/это время;
  до measured этапа оба0.
- `measured_http_count`, `measured_attempt_count`, `measured_success_count`,
  `request_latency.count`, per-operation counts:
  первые четыре равны400, per-operation80×5 в полном default успешном опыте.
  `http_count` содержит
  attempts и completed HTTP responses по phase/operation/status, включая
  setup/readiness. Expected successful total = tasks+2*sessions+
  sessions*requests_per_session+5 =2445 для default,39 для tiny2/10/10.
  Transport failures видны как разница attempts/completed и raw error sample;
  при них опыт FAIL. Полученный status при ошибке чтения body не означает
  completed response. HTTP ошибки считаются и не retried; non200 не входит
  в measured_success_count.
- `client_http_concurrency.peak_in_flight` — достигнутый максимум overlap
  клиентских roundtrips от request до полного body/error, по всем фазам.
  `http_count.by_phase` содержит attempted/completed и peak/active каждой фазы;
  `active_in_flight` после drain должен быть0. Вход и finally-выход защищены
  одним lock, также при transport error. Это не server in-flight и не wave_size.
- `request_latency`/`per_operation`: только время от HTTP request до полного
  чтения response body; JSON encode/decode, счётчики и pacer исключены.
  Все400 успешных измеряемых samples сохраняются, включая slow. Начиная с059,
  `http_samples` сохраняет каждую timing запись всех фаз (также non200 и
  transport failure), с phase/operation/status/completed/success/error_type.
  `failed_request_latency` отдельно суммирует measured non200/transport timing;
  их нельзя выдавать за успешные samples. Пустая summary имеет count0 и null
  percentiles. `expected_measured_requests` фиксирует план без достраивания
  недостающих attempts. При FAIL `workload_complete=false`,
  `latency_is_partial=true`, percentiles диагностические, thresholds отсутствуют.
- `pacing_context`: actual wave release offsets, wave count/size, minimum
  interval, отдельный rate wait и сумма ожиданий worker. Worker wait включает
  peers/scheduling и является суммой по потокам, не дополнительным wall time.
  Failed/aborted waits также сохраняются; незавершённый rate wait измеряется
  реальными monotonic часами, поскольку release timestamp для него отсутствует.
  Serial wait относится к setup/seed/warmup. Ничто из этого не включается в
  HTTP latency и не вычитается из измеренного roundtrip.
- `experiment_status=COMPLETED` означает завершение requests, не target PASS.
  Thresholds остались p95≤50ms, sampled RSS≤128MiB, populated startup≤1s.
  Exit0 не даёт performance PASS при false threshold или неподтверждённой среде.

Cgroup helper/ancestor limitations, shared unit generator+child budget, RSS
sampling50ms и исключение client RSS остаются как в APPROVE048/049. Нужны
отдельные actual PID/unit/limits evidence целевого запуска. Этот mix не
доказывает latency всех API. Annual200000 events, дополнительные mutations,
TLS и настоящий второй ПК в этом методе не проверяются.

## Фактические проверки055

`python3 -m unittest discover -s tests/infra -p 'test_catalog_load*.py' -v`:
**PASS31** (17 новых pacing/accounting cases +14 существующих cgroup cases).
Fake clock и threads проверяют400/20/equal mix, slow sample retention,
release intervals, blocked fast worker, no catch-up, deadlines, rollback,
abort sleep/barriers,429/no retry/FAIL report, legacy/parser и TTL argv.
Это synthetic stdlib units, не process tests будущей performance реализации.

Approved C binary SHA256
`59c633e286eacfb5c713cfd540857aff2152508c0801c78ec6f58a5a4029e55c`;
Mac arm64 tiny smoke2 sessions/10 tasks/по10 requests, rate100,
persisted-defaults: **PASS метода**,39/39 HTTP200,20 measured, по4 каждого типа.
Финальная версия: elapsed0.263243s, actual75.976rps, p954.575ms,
RSS24.938MiB, populated startup17.091ms. Наблюдённые три threshold true
относятся только к tiny Mac run. Cgroup/effective limits unavailable.
Предварительный tiny run сохранён отдельно и не используется как final freeze.

Evidence: ignored `.local/evidence/load-method-055/` — `unit.log`,
`c-smoke-final.{json,log}`, `freeze.sha256`, `method.diff`, `manifest.json`.
Target20/2000/VPS, annual200k, D runtime, Go/full suite, deployment/live —
**NOT-RUN**; production и cgroup helper не менялись. Следующий этап —
независимое review exact freeze, затем отдельно согласованный целевой опыт.

API semantics сверены18.10.2026 по primary Python documentation:
[monotonic clock](https://docs.python.org/3.14/library/time.html#time.monotonic),
[Barrier/action/abort](https://docs.python.org/3.14/library/threading.html#barrier-objects),
[Event.wait](https://docs.python.org/3.14/library/threading.html#threading.Event.wait).

## Исправления059 и отдельная проверка

REV058-01/02: actual client HTTP overlap, raw timings/errors и partial
duration/pacing/counts сохраняются на FAIL после drain всех futures.
`submit/as_completed` заменяет теряющий результаты `pool.map`; общий ledger
записывает ровно один sample на actual attempt. API сверено по
[Python futures/shutdown](https://docs.python.org/3.14/library/concurrent.futures.html).
Авторские stdlib regression units воспроизводят потерю055 и проверяют
429/transport/barrier failure, поздний slow peer, phases/cleanup и400/80×5.
Текущий freeze, фактические результаты и ограничения — WORK-059.
Независимое approve новой версии ещё требуется; baseline049 p95 FAIL сохраняется.
