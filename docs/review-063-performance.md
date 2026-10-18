# WORK-063 — независимое code review performance / schema 4

**REQUEST CHANGES.** Найдены два блокирующих дефекта изоляции snapshot.
Проверен commit `b9fdc155a1d0a02bf6000c3654e006a778be23cd`,
`codex/performance`, base `30abe1674e972ea28272dacdb18392070840e0ed`.
Финальный source aggregate 48:
`42d32f7826481346d8fc16fede29bb04c674708560b36edbe8e517c61272c2d6`.
Manifest `.local/work057/source-manifest.json` SHA256:
`67c8d3d8b7e336214007140796b6fd14d6c1cc92b6eba78560b142c6e5ae69ce`.
Независимо сверены все 51 author file hashes и пересчитан aggregate 48 —
совпадение. Также сверены15 author artifacts hashes. Evidence:
`.local/work063/manifest-check.json`, `author-artifacts-check.json`.
Production/test files не менялись во время финального recheck; HEAD сверён.
Прежние aggregates не подменяют эту финальную привязку.

## Блокирующие замечания

### R063-01 · P1 · Create обходит COW при открытии завершённого эпика

`internal/store/tasks.go:38` переводит lookup эпика на `readTaskFor`, но
`tasks.go:72–75` продолжает напрямую менять `epic.State` и `epic.Version`.
Этот указатель общий с committed snapshot, а ключ эпика не попадает в dirty
journal. Воспроизведены обе ветки:

1. Закрыть пустой эпик (`done`, version2), успешно создать в нём developer task.
   В retained snapshot эпик становится `open`, version3, а SQL сохраняет
   `done`, version2. После reload/restart новая задача оказывается в закрытом
   эпике; следующий Next её не выдаёт. Audit reopen и SQL entity расходятся.
2. После закрытия эпика вызвать тот же Create с `areas:["../outside"]`.
   Команда отвергнута, но следующий Get возвращает `open`, version3 вместо
   исходного `done`, version2. Candidate rejection загрязняет committed cache.

Нужно менять эпик через write barrier/reopenEpic, включая порядок с verifier
reopen того же эпика. Нужны регрессии success→durable reload и rejected create
после изменения эпика; нельзя ограничиваться возвратом правильного DTO.

Доказательство: `.local/work063/reviewer_probe_test.go`, tests
`TestReview063CreateIntoDoneEpicPersistsReopen` и
`TestReview063RejectedCreateDoesNotPolluteEpic`: **FAIL** на финальном commit; повторены после freeze.
Отрицательный контроль: только `tasks.go` восстановлен в private overlay из
base `30abe1674e972ea28272dacdb18392070840e0ed`, оба probes **PASS**.
Production checkout не редактировался.

### R063-02 · P1 · Local HandoffEpic возвращает ссылки на retained snapshot

`internal/store/handoff.go:46,52`: результат записывается во внешнюю переменную
`out` и возвращается после `adminTransaction`. Внутренний generic transaction
детачит только собственный result типа `struct{}` (`maintenance.go:49–52`),
поэтому Task из `handoffEpic`/`viewTask` остаётся shallow copy candidate.
После commit candidate становится retained cache.

Воспроизведение: закрыть эпик, снова открыть, вызвать local HandoffEpic и
изменить `*result.FirstDoneAt`. Следующий обычный Get показывает изменённую
дату (2025 вместо сохранённого 2026), хотя caller не выполнил SQL/API mutation.
Ссылки `CreatedAt`, `Areas`, `Links` и остальные nested поля требуют той же
изоляции. При конкурентном использовании это также допускает data race.

Нужно вернуть Task через generic transaction с detachment **до commit** либо
явно детачить `out` внутри admin callback с обработкой ошибки до commit.
Не достаточно детачить после commit с потенциальной ошибкой без отката.

Доказательство: `TestReview063HandoffResultDetached` в том же private overlay:
**FAIL** на финальном commit. Публичные author060 alias tests этот local admin путь не включают.

## Проверенная область

Прочитаны AGENTS Accord, процесс/parallel-work/reviewer Давинчи, requirements,
source TZ §15.2, service/config/performance053 contracts, approved D054 fixes,
WORK057/060 и performance-runtime. Model binding Claude недоступен; review
выполняет отдельный агент той же наследуемой модели. Tandem OFF по прямому
поручению. Source, production tests, Давинчи, общая TASK, Git и live не менялись.

Проверены retained immediate writer и context gate, все `s.db` callers,
DSN busy_timeout/cache_size, data_version generation и invalidation,
config canonical/hash cache, SQL authority до replay, expiry boundary,
COW/dirty persistence, mutation rejection, commit ambiguity, nested DTO,
affected dependency/review/epic closure, numeric/title/area/ownership indexes,
bounded events/high-water, schema 4 exact prefix/blocked integrals, legacy
fallback, dirty triggers, streaming migration/backup/rollback и infra readiness.

Основные механизмы разумны: нет board JSON clone/save на обычной мутации;
постоянный cache не хранит raw annual events; prefixes exact decimal big.Int,
UTC seconds/nanoseconds, query range полуоткрытый; invalid history идёт через
oracle. Собственные probes выше показывают, что покрытие barrier/DTO неполно.
Соседние admin results проверены: Config bytes копирует Snapshot API,
TokenInfo/RecoveryResult не ссылаются на board entities; local DB getters
декодируют собственные значения.

## Карта write paths и outward DTO

| Путь | Barrier / результат проверки |
| --- | --- |
| Create task/epic | `putTask`; parent done epic обходит barrier — R063-01 |
| Update, Transition, Handoff через bearer, Link target | `taskFor` → `editTask`; изменения attempts/evidence/RawMessage вложены в detached entity |
| Next, Heartbeat, Comment, Release | `editTask`/`owned`, session candidate `editSession`; active owner/area overlays учитывают dirty delta |
| Claim epic owner, reopen/reconcile | `editTask`; общий global/review/epic affected closure; исключение прямого Create указано выше |
| duplicate_of, unlink, warnings | source/target прошли `editTask`, slices private; предупреждения пишутся в candidate/event |
| Describe / generations | отдельная session + dirty entry; Roles map private в каждом fork |
| Note create/delete | новая запись + dirtyNotes либо `editNote`; outward generic detach |
| Expiry / Sweep / recovery | expired/recovered task/session через edit barriers; expiry отделён от rejected domain candidate |
| Token/config/scopes/revoke-all | authoritative SQL в admin transaction; config переключает приватную runtime representation; integer/string/bytes results не разделяют entity state |
| Local HandoffEpic | entity edit корректен, наружный Task не детачится — R063-02 |
| Get/Lists/Events/Stats/ReadConfig/History | generic JSON detach до commit; local getters читают SQL и создают независимые DTO |
| Persistence / projection / indexes | dirty prepared UPSERT; events/projection/replay атомарны; index delta только после commit, error/panic инвалидируют generation |

Без новых замечаний по прочитанным writer/config authority, durable replay
retry, bounded events, exact stats/fallback и atomic migration механизмам.
Это ограниченная оценка source и targeted checks, не частичный функциональный
approve и не замена security/process/runtime приёмки.

## Фактические проверки

Toolchain: `/Users/shagi/Desktop/Project/agent-board/.local/toolchains/go/bin/go`,
Go1.27.2 darwin/arm64; GOTOOLCHAIN=local, cache/mod/tmp из MAIN toolchains.
Все evidence находятся в `.local/work063/`; точные команды, counts, exit codes
и SHA256 логов/overlay/probes записаны в `results.json`. Исходные 13 race и 17
contract tests выполнены до финальной index-only delta; affected events race,
все 21 performance units и 3 собственных probes повторены после freeze.

| Проверка | Результат |
| --- | --- |
| Targeted performance store unit: Extra/Immediate/Projection/Indeterminate/Events/Migration | PASS, 21 top-level; final recheck PASS 21 |
| Targeted race: Extra/Immediate/Indeterminate | PASS, 13 top-level; final affected events race PASS 1 |
| D config CAS, area role, cancellation, incomplete evidence, role TTL, global/superseded/cross-epic gates, concurrent claim, event pagination | PASS, 17 top-level |
| Python migration readiness/guards | PASS, 5 tests |
| Existing D056 strict process CAS/audit rollback через неизменённый private overlay | PASS, 2 повтора на финальном source |
| Собственные focused unit probes | FAIL, 3 tests / 2 blockers; final recheck FAIL 3 |
| Private base tasks.go negative control | PASS, 2 tests |
| git diff --check | PASS |

Тестовый PASS не является функциональным/ресурсным PASS. Полный комплект
cmd/internal/race/build здесь не повторялся: один полный прогон принадлежит
автору/координатору. Новые product process/E2E не создавались и не запускались.
Existing D056 process testcase прочитан: он требует ровно один успех,
строгий CONFIG_CONFLICT для проигравшего, version/history/audit CAS poststate,
zero partial writes при audit/invalid-candidate fault. Самостоятельно повторён
дважды с новым real CLI из final source (binary SHA в final-qa056.log);
новые process tests не писались. Полная D QA этим не закрыта.
Replay correction самостоятельно пройдена в 21 units, включая expired session
после смены SQL scopes; replay не продлевает lease. StartupRetry классифицирует
настоящий typed sqlite.Error BUSY/BUSY_RECOVERY, не строку, и повторяет только
admission/startup, не business body/commit.

Финальная events delta прочитана отдельно: static INDEXED BY выбирает exact
schema 4 task/type/composite stream index; query plan regression проверяет
именно этот index и SEARCH/no temp sort. Авторский VDBE MAX probe проверен:
Last/SeekLE + max-optimization, полного predecessor range scan нет.
Все затронутые Go-файлы≤254 строки; новых дублей, ненужных dependency/API
обновлений и ослабления limiter обнаружено не было.

Авторский annual diagnostic и полные unit/race/vet/infra logs прочитаны;
их 15 artifact hashes совпали. Финальные single-call числа: migration 29.270 s,
restart 4.248 ms, events 20 — 0.487 ms, process peak RSS 79,904,768 bytes. Fixture SQL
synthetic, streaming 200k/2000, compatibility=false, oracle differential;
задачи API и raw annual history не моделируют полный reviewed target workload.
Эти прогоны не повторены ревьюером и не приписываются ему. Отдельного security approval этот отчёт не заменяет.

## Ограничения и следующий шаг

Старый WORK049 p95=10025.109ms **FAIL** остаётся открытым. Local synthetic
200k diagnostic не доказывает p95≤50ms/RSS≤128MiB/start≤1s на 1 vCPU/512 MiB;
полный required process QA, target VPS resources, unchanged049 и reviewed paced
admission workload, TLS, physical2PC/providers здесь **NOT-RUN**.
Admission 100 rps / burst 200 / max-inflight 64 не изменён. Backup retention/byte budget
принадлежит WORK-062. Schema4 owner 057, future E schema5; live/deploy нет.

Вернуть разработчику оба blockers одной дельтой. После исправлений нужен
свежий freeze, affected unit/probes и независимая повторная проверка; затем
отдельный tester создаёт process/E2E. Общая TASK не считается завершённой.
