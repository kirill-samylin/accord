# WORK-053 — контракт производительности Accord

18.10.2026. Архитектор `/root/perfarch053`; анализ, не implementation approve.
База C: `1e9025cd6e3122f0adf6165e09845202736d004a`, schema2.
D: isolated `932aa2a`, исправления WORK-052 проходят отдельно; их файлы не менялись.
Источники: requirements V1-09, service/config/security contracts,
[runtime049](runtime-catalog-049.md), [метод042](acceptance-042-catalog.md).
Цель: p95 API≤50ms, RSS≤128MiB, populated startup≤1s на1vCPU/512MiB;
20 sessions/2000 tasks/200000 annual events. Обязателен повтор04920/2000/20,
но его ≈2000 events не заменяют годовой объём и обычные мутации.

## 1. Установленная причина и границы доказательства

049:400 HTTP200; p50=3463.879ms,p95=10025.109ms,p99=12798.825ms,
max17163.349ms, measured91.800s, seed237.626s. Latency **FAIL**;
RSS35.723MiB/start28.268ms — наблюдённые PASS, не полный V1.
Общие leaf/unit100%CPU/512MiB подтверждены; ancestor limits unavailable.
Исходный raw049 сохраняется; здесь VPS/deploy/helpers не запускались.

C `store.go:transact` получает writer lock, загружает все task/session/note JSON,
дважды считает metricFacts/enrichEvents и вызывает saveBoard даже для GET.
`persistence.go:saveEntities` marshal+Exec UPSERT для каждой строки: WHERE
предотвращает изменение одинаковых данных, но не сериализацию и SQL prepare.
`mutation.go:mutate` дополнительно JSON-клонирует целую доску.
`events.go` фильтрует уже после SQL чтения/JSON decode всех кандидатов;
`stats.go` каждый запрос перечитывает весь журнал. MaxOpenConns1 сериализует
также Authenticate; конкуренция делает стоимость одной операции общей очередью.

Private диагностика: Go1.27.2 darwin/arm64 M1Pro, GOMAXPROCS1, SQL fixture
2000tasks+epic/20sessions/2041events, fixed clock, no leases/dependencies.
Прямые реальные Store API; fixture seeded SQL, не HTTP acceptance049.

| Измерение | Время на операцию, два повтора |
| --- | --- |
| get / tasks20 | 24.58–24.62 /25.63–25.65ms |
| filtered events20 /stats(owner) | 29.04–29.30 /29.56–29.58ms |
| heartbeat /mixed | 32.02–32.52 /28.10–28.33ms |
| loadBoard /saveBoard неизменённой доски | 5.511–5.513 /16.959–17.003ms |
| metricFacts+enrichEvents дважды /JSON clone | 1.147–1.169 /6.253–6.267ms |

Mixed CPU profile: saveBoard cumulative47.65%, SQLite prepareV2 25.57% всего
sample CPU; matching final run даёт50.95%/23.58% соответственно. Это включающий setup/GC профиль, доли не складывать; отдельные
component benchmarks подтверждают порядок стоимости. Не измерен VPS CPU profile.
Убрать только saveBoard недостаточно: оставшиеся ≈10ms при20 serial requests
уже дают очередь порядка200ms даже на этой машине; это оценка, не p95 прогноз.

Private annual diagnostic:200000 synthetic events (2041 исходных +commented),
один настоящий C запрос: filtered events522.632ms/221.87MB allocations;
stats(owner)514.192ms/218.66MB allocations. Allocations не RSS/peak heap.
Fixture проверяет scan complexity, не полноту статистических сценариев.
D loadActiveConfig после writer lock повторяет config.Parse(canonicalJSON).
Отдельный config-package опыт полного шаблона: Parse5.646–5.737ms,
3.917MB/≈49056alloc; Snapshot.Decode79.3–80.5µs. Это дополнительный D риск,
не измерение D VPS и не полная стоимость D transaction.

## 2. Выбор и неизменяемые гарантии

Минимальный путь сохраняет JSON entities, SQL источник истины, DTO и SQLite
writer serialization. Не менять СУБД, HTTP/auth protocol или dependencies.
Нужны вместе: dirty persistence, изолированный per-entity candidate,
проверяемый immutable snapshot, bounded event query, persisted stats projection.
Prepared UPSERT без этого помогает SQL prepare, но оставляет O(board) работу;
полный event cache отвергнут из-за200k memory/startup и повторного scan.
Read-only WAL pool сам по себе не решает CPU/JSON; отдельный async snapshot
потребовал бы новой модели revocation/expiry и здесь не выбран.

- Auth/token scopes перечитываются SQL в authoritative transaction, включая replay.
- CAS, singleton, task fencing, exact-SHA gates, независимые server sessions,
  одно владение/session и audit не переходят в eventually consistent cache.
- now берётся после writer lock. `now == expires_at` уже expired;
  reads/replay не продлевают task/role/session. Sweep не является авторизацией.
- Domain failure сохраняет только предусмотренную материализацию expiry;
  candidate mutation, lease renewal, replay и её audit не коммитятся частично.
- DB/commit/connection error не возвращает старый успешный cached result.
  Неопределённый commit инвалидирует кэш; retry проверяет durable replay.
- Ни ошибки, ни частично обновлённые индексы/проекции не скрываются за PASS.

## 3. Snapshot, SQL connection и владение

Для процесса один dedicated writer `*sql.Conn` + context-aware transaction gate.
Сохранить SQL writer lock до чтения state/auth/config. Предпочесть driver
`_txlock=immediate` на writer DSN: pinned modernc v1.60.1 sqlite.go395–400
поддерживает его; private probe проверяет lock уже при BeginTx. Убрать dummy
UPDATE для lock, не выполнять BEGIN через DB Exec с Tx другого connection.
Полностью no-op/read transaction без expiry/replay/projection writes не должна
менять data_version соседнего процесса; rollback либо чистый read commit.
Pool capacity минимум2: retained writer и spare для существующих Authenticate,
Config/history/local lists/Ready; иначе прежний MaxOpenConns1 даст deadlock.
Все mutating/admin/recovery transactions маршрутизируются через writer gate.
Перечислить все `s.db` callers; PRAGMA busy_timeout задаётся для каждого нужного
connection, а не единожды случайному pool member. Backup имеет отдельный
existing handle и прежний consistent protocol. Close прекращает приём,
дожидается/отменяет owned operations, закрывает writerConn прежде DB;
не оставляет занятый pool/blocked goroutine.

На том же writer connection после lock читать PRAGMA data_version. Сравнение
валидно только внутри одной connection generation. External same-DB commit
меняет token; собственный commit не меняет. При mismatch полный reload state,
индексов и current config до применения операции. Retain только committed
immutable board. Свои изменения publish после успешного commit вместе с cache
state; local admin не обходит это правило. Raw SQL через spare connection
обнаруживается как external. При reconnect/badconn/panic/rollback uncertainty
сбросить всю cache generation; новый connection начинает с cold reload.
Не угадывать значение data_version и не читать его после release writer lock
для маркировки старого snapshot как нового. Restore/DB replacement только offline.

Private modernc test подтвердил token2→2 на own commit и2→3 после другого
connection и последующего writer lock; process/restart/error cases ещё NOT-RUN.
Обычный s.db.BeginTx без закреплённого connection не подходит для этого token.

D active version/hash читать в той же transaction. Parse/compiled schema cache
key=(persisted version,hash,connection generation), bounded текущей редакцией;
canonical bytes/hash проверяются, invalid config fails closed. Не использовать
mutable Store.cfg как источник правил. Snapshot immutable; Workflow maps либо
private Decode на transaction, либо полностью immutable internal representation.
D auth/role TTL fixes052 сохраняются без переписывания их продуктовых правил.

## 4. Candidate, dirty persistence и стоимость обычных мутаций

Начать candidate с отдельных maps и immutable shared entities; при первом
изменении entity полный JSON detach **одной** task/session/note, включая D
RawMessage evidence, attempts, lease, links, slices/maps/pointers. Это безопаснее
неполного ручного shallow clone; не весь board. Все write accesses проходят
editTask/editSession/editNote/editRole с dirty journal. Reviewer составляет
исчерпывающую карту прямых записей/алиасов, включая expiry, reconcileEpics,
gates, handoff, duplicate/link/unlink, note/admin/recovery и config apply.
Обойти edit barrier прямой записью shared pointer запрещено.

Сериализовать/upsert только dirty entities; metadata/note sequence/roles только
при изменении. Готовить SQL statement один раз на transaction/table; закрывать.
Events, replay, config/scopes и projection updates входят в тот же commit.
No-op/rejected candidate не трогает committed snapshot; expiry имеет отдельный
candidate boundary до доменной мутации. Replay возвращает durable bytes без renewal.

Returned Task/Note/Session/Claim/Events/Stats/Config DTO полностью detached.
Текущие viewTask `v:=*t` и `*n` для cache недостаточны: caller mutation результата
не должна менять следующую transaction. Детачить весь outward graph, не только
верхнюю структуру; RawMessage bytes тоже. При Marshal/clone error — отказ.

Metric facts считать для affected closure: touched tasks, dependents, review
assignments/targets, эпики и их transitive dependents. До/после берутся из
committed/candidate версий. Все глобальные gate invalidations остаются видимы.
Transaction-local memo gate predicates и reverse dependency/verification indexes
исключают многократный scan всей board для каждой вершины. Cache memo нельзя
переносить между изменёнными snapshots без точной invalidation.
Numeric task order, role/session task ownership, area и title-trigram candidate
indexes принадлежат snapshot, обновляются с dirty delta, rebuild при reload.
Не вводить несогласованный cap graph/description/groups ради latency.

Heartbeat имеет ту же изоляцию/проверки, но меняет только свою session и возможную
task lease. Бounded heartbeat — первый срез, не окончательный fix: ordinary
create/update/comment/next/transition/release/link/note также обязаны пройти
стоимостную и семантическую проверку. Полный board JSON clone/save в них не
допускается в финальном результате. Recovery/import могут быть явно maintenance
workflows, но нельзя молча исключить обычную API mutation из V1-09.

## 5. Schema4: events и статистика без сканирования годового журнала

Предложение согласовано координатором для синхронизации контрактов: D остаётся
schema3, performance получает schema4, ещё не начатый E runtime — schema5.
**До production implementation** root обновляет config/integration contracts и
назначает единственного migration owner. Этот документ сам схемы не меняет.

Events остаются canonical raw JSON. Schema4 добавляет expression/composite indexes
(task_key,cursor), (type,cursor), (task_key,type,cursor), используя одинаковые
json_extract expressions в DDL/query. SQL применяет filters+high_water и LIMIT+1;
для нескольких types bounded merge отсортированных cursor streams без full scan.
NextCursor/HasMore сохраняют прежний highest-scanned смысл, включая gaps и
nonmatching tail: при наличии следующего match найти последний actual cursor
перед ним, при исчерпании — последнюю строку до high_water. Не подменять opaque
list cursors или включать события после snapshot. EXPLAIN подтверждает index
search для каждого supported filter combination, не только favourable fixture.

Статистика: материализованная projection из **event-time Data**, не join старого
события к нынешней task. E/import не получает права подставлять клиентскую
историю audit/time/facts: imported payload не становится authoritative event;
legacy existing events сохраняются с incomplete признаками. Минимальные таблицы:

- projection_control: reducer_version, built_cursor, dirty, chronology status;
- stats_prefix: (group_kind=total|role|owner|epic,group_key,time_key) ordered key,
  cumulative additive metrics и presence/incomplete counters;
- blocked_prefix: те же group keys/time, integral elapsed blocked-ns и active
  blocked count после boundary; blocked_open связывает task со start facts.

time_key — точная UTC пара(seconds,nanosecond), не UnixNano int64 для всех
RFC3339 дат: zero/legacy year1 выходит за его диапазон. Row-value key comparison
сохраняет порядок; bounds не округлять. Обычный chronological append обновляет
хвост prefix в transaction события. Одинаковый timestamp coalesced. Для [from,to) additive metrics — разность двух
prefix `<to`/`<from`. Group presence отдельна: zero-metric events тоже создают
группу; дополнительно groups из overlapping spans и current queue.
Blocked integral F(t)=area_at_boundary+active_count*(t-boundary); F(to)-F(from)
даёт clipping обоих краёв, open intervals ограничены min(to,now). Group metadata
берётся с начала span. Terminal stop, repeated block, cancelled/restart и
incomplete legacy должны совпадать с текущим reducer. CurrentQueue вычисляется
из authoritative post-expiry snapshot и актуальных gates/config, не prefix.

Любые backwards timestamps/нестандартный повтор start, historical rewrite или
неподдержанный reducer shape не получают неверный быстрый ответ: явный dirty/
compatibility path с исходным reducer. Это correctness fallback, не performance
PASS; такие fixtures проверяются отдельно, а найденные частые случаи требуют
улучшения projection до приёмки. Нельзя навязать монотонные часы продукту.
Тесты должны доказать отсутствие этого fallback на штатном annual workload.

Tokens/prefix не через SQLite REAL/float64. Cumulative history может превысить
int64, хотя результат узкого окна валиден: использовать exact decimal big.Int
TEXT либо эквивалентную checked exact representation. Разность проверяется на
контрактный int64; исходный limit_exceeded сохраняется. Counts/presence/elapsed
ns также checked; duration сравнивать с oracle с документированной численной
точностью существующих seconds DTO, без молчаливого округления timestamps;
legacy большие интервалы сохраняют Time.Sub semantics через oracle/fallback.

Raw events + projection + built_cursor атомарны. INSERT/UPDATE/DELETE event
triggers помечают projection dirty; штатный writer очищает dirty только после
успешного обновления всех derived rows до exact high_water. Другой same-version
процесс использует тот же protocol. Внешняя SQL поправка истории требует rebuild
до stats, даже если max(cursor) не изменился; data_version сбрасывает RAM cache.
Нельзя отвечать stale projection, запускать async rebuild и временно выдавать
неполные цифры. Event payload/replay bytes миграция не переписывает.

## 6. Миграция, ресурсы и последовательность реализации

Перед schema3→4 — private verified snapshot по существующему migration protocol.
DDL/indexes/streaming backfill/control/user_version4 в **одной** write transaction;
bounded chunks cursor iteration без удержания200k decoded events. После backfill
проверить coverage/version/invariants и differential totals/ranges, commit до
listening. Ошибка/cancel/disk-full/DDL conflict оставляют schema3, verified backup
сохраняется. No partially published schema4. Migration duration измерять отдельно
от обычного populated restart; не называть initial migration секундным startup.
Restart schema4 не сканирует raw annual events и не пересчитывает projection.
Backward binary отвергает schema4; downgrade только offline restore прежнего
snapshot+revoke-all/recovery reset, с явной потерей последующих изменений.
Проверить backup/restore metadata4 и будущую4→5 совместимость E.

RAM cache хранит board/compact indexes/current config, **не200k event DTO**;
projection persistent. SQLite cache budget и bounded statements измерить вместе
с board/result buffers. Streaming migration может пользоваться temp disk, но
не маскирует CPU/RSS/wall-time; rollback и free-space failure обязательны.
Файлы реализации делить до≈300lines по ответственности.

Порядок и предлагаемые границы файлов:

1. D052 independent review/QA и root sync schema/contracts; frozen combined base.
2. `store.go`, новые `transaction_snapshot.go`/`snapshot_copy.go`: gate/connection,
   immutable cache lifecycle и deep response detachment; cross-process tests.
3. `persistence.go`, `mutation.go`, `maintenance.go`, domain store files:
   dirty journal/COW barriers, incremental prepared persistence, graph indexes.
4. `migrations.go`+`migrations_performance.go`, `events.go`, новые
   `stats_projection.go`/`stats_prefix.go`/`stats_blocked.go`: schema4 и reducer.
5. `stats.go`, D config_bootstrap/runtime snapshot: indexed reads/config decode
   reuse; all API regression, independent code/security review затем tester.
6. Review новых process/load fixtures; immutable binary, isolated VPS repeat;
   ordinary deployment/merge — только координатором в разрешённом объёме.

## 7. Проверки, нагрузка и критерий завершения

Developer: differential oracle старого reducer vs projection на randomized
истории/границах, all group modes/zero groups, int64 extremes/corrections,
incomplete legacy, block/cancel/reopen/gate invalidation, exact replay bytes.
Alias test мутирует каждый возвращённый nested object/slice/map/RawMessage;
следующая API/DB transaction неизменна. Fault после dirty entity/projection/audit/
replay до commit, canceled context и commit ambiguity не публикуют candidate.
Fake PASS по query plan или benchmark недостаточен.

Два Store и отдельные процессы на одной DB: concurrent claim/CAS/config apply,
external token revoke/scopes change→auth+replay deny, external task/session/
config change→следующий snapshot свежий. Restart/connection replacement/reset,
rollback, expiry equality и role/task/session TTL из D052 обязательны.
Projection invalidation проверить и append, и UPDATE старого event без изменения
cursor; обычный raw read не продлевает lease. Все прежние C/D gate/audit tests.

Tester после approve: исходный04920sessions/2000tasks/20requests,400 measured,
get/tasks20/filtered events20/stats(owner)/heartbeat поровну, populated restart;
отдельно reviewed annual fixture200000events и обычные API mutations на нём.
Annual история содержит completion/rework/tokens/missing/expiry/warnings,
blocked spans/cross-window, notes/config/admin и крупные pagination gaps.
SQL seed допустим как явно synthetic fixture, проверенная oracle + hash;
actual runtime/API operations идут через реальный binary, без mocked store.
Измерить cold first call, warm, per-operation p50/p95/p99/max, alloc/CPU/lock wait,
RSS sampled + cgroup peak, startup/restart, seed/migration time отдельно.
Обязательны normal/race targeted и один полный check; independent review отдельно.

Admission policy C:100rps/burst200/max-inflight64. Не ослаблять её ради нагрузки.
Ускоренный unpaced seed/400 burst может получить429; исходный exact049 rerun
сохранить полностью,429 — admission outcome, не успешный latency запрос.
Дополнительный **отдельно reviewed** paced method ≤configured rate,20 sessions,
те же400 operations/mix и возможность20-way burst нужен для latency внутри
policy; count/error/429 и actual concurrency публикуются. Не переименовывать его
в unchanged049 и не считать retry/backoff частью исчезнувшей latency. Burst
saturation429 проверяется отдельно. Если требуется именно unpaced400 all200,
root согласует конфликт admission/нагрузки до приёмки, не меняет limiter молча.

PASS только после всех применимых semantic/review/runtime checks и p95≤50ms
для всех обычных API в reviewed workloads, RSS≤128MiB/start≤1s на target VPS.
Большие graph fan-out, много групп/максимальные payloads и совместная нагрузка
Tandem — измеряемые риски, не обещание из toy point. TLS overhead/effective
ancestor limits отмечаются отдельно; loopback HTTP не доказывает TLS p95.
049 FAIL и annual provider NOT-RUN остаются до новых фактических результатов.

## 8. Первичные API-источники и evidence

Проверены18.10.2026: [SQLite isolation](https://www.sqlite.org/isolation.html),
[data_version](https://www.sqlite.org/pragma.html#pragma_data_version),
[expression indexes](https://www.sqlite.org/expridx.html),
[Go sql.Conn](https://pkg.go.dev/database/sql#DB.Conn),
[transaction PrepareContext](https://pkg.go.dev/database/sql#Tx.PrepareContext),
[math/big.Int](https://pkg.go.dev/math/big#Int),
[time.UnixNano](https://pkg.go.dev/time#Time.UnixNano),
[SQLite row values](https://www.sqlite.org/rowvalue.html). External links — обоснование API,
не доказательство текущей реализации. Подробный протокол — [WORK](work/WORK-053-performance-analysis.md).
Private `.local/evidence/performance-053/`: overlay Go benchmarks/probe,
logs/pprof/source hashes. Narrow cached sketch45–376µs и annual indexed
query32.2µs/prefix20owner lookup388–389µs — только cost feasibility; в sketches
нет полного reducer, expiry/config/gates/response pagination semantics.
Они не являются production fix, acceptance PASS или прогнозом VPS p95.
