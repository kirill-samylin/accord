# WORK-066 — исправления performance review063

Статус: на независимой проверке; авторская реализация выполнена. Исполнитель `/root/perffix066`, роль разработчик, экземпляр
этого запуска. Parent TASK003; пакет R063-01/02 из review063. Отдельная копия
`.local/performance-fixes-003`, ветка `codex/performance-fixes`, входной HEAD
`b9fdc155a1d0a02bf6000c3654e006a778be23cd`; review aggregate48
`42d32f7826481346d8fc16fede29bb04c674708560b36edbe8e517c61272c2d6`.

Именованные Claude binding здесь недоступны, используется наследуемая модель.
Tandem OFF по прямому указанию пользователя, claims не выполняются. Прочитаны
AGENTS Accord, процесс/parallel-work/developer Давинчи, config-contract,
performance-contract053, performance-runtime, WORK057/060 и frozen review063.
Источник review: `.local/performance-003`; эта копия не редактируется.

## Границы

Разрешены необходимые store source, новые unit/integration регрессии, этот WORK
и `docs/performance-runtime.md`. Schema/migration owner057, Go dependencies,
infra/transport/лимитер, общая TASK, Давинчи, Git commit/push/live не меняются.
Новые process/E2E принадлежат tester после независимого approve. Исторические
WORK049 latency FAIL и D056 flaky FAIL этим заданием не закрываются.

## Журнал

- started 2026-10-18T20:11:24+03:00: `/root/perffix066`, назначено → в работе;
  чистая изолированная копия на b9 проверена; единый пакет двух blockers принят.
  Первым добавлен отрицательный контроль регрессий на неизменённом b9 source.

## Реализация

R063-01: Create вызывает существующий `reopenEpic` для parent. Его повторный
lookup видит актуальный candidate после verifier reopen, а `editTask` создаёт
изолированную entity и dirty entry. Успешный reopen сохраняется в SQL вместе
с child/audit/replay; domain rejection не меняет committed RAM/SQL. В negative
control дополнительно подтверждено двойное `epic_reopened` для verifier в том
же эпике: прежний read-pointer оставался `done` после первого COW reopen.
Теперь version/audit изменяются ровно один раз. Это часть R063-01.

R063-02: local HandoffEpic возвращает `core.Task` непосредственно через generic
`transaction(..., false, ...)`. Сохранён admin-режим без expiry; полный outward
JSON graph детачится до save/commit. Clone error может откатить транзакцию,
после committed success дополнительного fallible преобразования нет.

Проверены соседние local admin callbacks: CreateToken содержит собственный
scalar TokenView; RevokeAllTokens/RecoverState возвращают числа; SetTokenScopes
получает TokenInfo из SQL и собственный roles slice; ApplyConfig получает bytes
через копирующий Snapshot.CanonicalJSON. Они не захватывают board entities.
Local Config/history/ListTokens/ListSessions читают SQL в собственные DTO.
Дополнительных таких дефектов не найдено; соседние wrappers не переписывались.
Read-only `readTaskFor` callers в Link/Note/Create не меняют shared entity;
изменения target в Link по-прежнему проходят taskFor/editTask.

Production diff ограничен `tasks.go` и `handoff.go`; контракт, migration4,
authority/lease/config/pool/admission policy и цель50ms не менялись.

## Регрессии и отрицательный контроль

Новые tests — unit/integration, не process/E2E:

- `reopen_performance_fix_test.go`: developer и verifier с тем же/другим parent;
  cached Get + raw SQL + actual Store Close/OpenExisting; ровно один reopen
  version/audit, child выдаётся через Next после restart. Create replay сохраняет
  bytes и durable dump без session renewal/audit; caller poisoning ответа
  replay не меняет. Invalid areas и metadata проверяют нулевые business writes
  во всех таблицах, неизменный кэш и restart, свободный request_id для retry.
  При expiry equality отвергнутый Create коммитит только unrelated lease expiry,
  не меняет parent, sequence, replay или coordinator session.
- `handoff_performance_fix_test.go`: естественные API timestamps и synthetic
  persisted historical Task graph с attempts/EndedAt/TokensSpent, areas, links,
  comments, nested warnings, similarity, evidence map/RawMessage, review и
  EvidenceComplete. Полный caller graph poisoning не меняет следующую API/SQL
  transaction; отдельное concurrent poisoning/read проходит под race detector.
  SQL audit abort откатывает owner/version/audit/projection и не возвращает
  successful-looking Task; retry проходит, stale CAS оставляет DB неизменной.
  Lease graphs дополнительно покрываются неизменным author060 test suite.

Private overlay подставляет ТОЛЬКО исходные `tasks.go`/`handoff.go` из b9 и
оригинальные reviewer063 probes. Production checkout/frozen source063 не
редактируются. Финальные новые5 + reviewer3 tests: **EXPECTED FAIL8** на b9,
**PASS8** после fix. В rich alias negative cache poisoning намеренно делает
RawMessage invalid; sequential assertion предшествует concurrent проверке,
поэтому baseline не зависит от concurrent map panic. Последние изменения
тестов добавляли explicit rejected restart и этот deterministic guard;
production source с начала positive checks не менялся.

## Фактические проверки

Pinned `/Users/shagi/Desktop/Project/agent-board/.local/toolchains/go/bin/go`,
go1.27.2 darwin/arm64. `GOTOOLCHAIN=local`; GOCACHE/GOMODCACHE/GOTMPDIR в MAIN
`.local/toolchains/{cache/build,cache/mod,tmp}`. Exact commands, timestamps,
exit codes/counts и SHA256 logs находятся в `.local/work066/results.json`.

| Проверка | Фактический результат |
| --- | --- |
| Affected store unit: новые Fix + reviewer probes + author060/authority/fault/COW, immediate/commit ambiguity, epic/global/review/lease/config/admin | PASS43 top-level, `affected-unit.log` |
| Тот же affected set с race | PASS43 top-level, `affected-race.log`,32.831s |
| Final tests после explicit rejected restart/deterministic guard: `go test -race -overlay .local/work066/review-overlay.json ./internal/store -run '^(TestPerformanceFix\|TestReview063)' -count=1 -v` | PASS8 top-level +11subcases, `freeze-race.log` |
| Final b9 overlay, тот же namespace без race | EXPECTED FAIL8 top-level, `freeze-negative.log` |
| `go vet ./internal/store` | PASS; повторён на final tests, `freeze-vet.log` |
| `go build ./cmd/...` | PASS, `build.log`; собственный binary перемещён в ignored `.local/work066/agent-board` |
| `git diff --check` | PASS |

Build относится к неизменным финальным production source; broad43 run был до
двух последних test-only additions, финальный affected8 race включает обе.
Не повторялись full check/annual/migration/load без новой причины. Existing
process QA056 здесь NOT-RUN; новые process/E2E не писались. Независимые code/
security re-review и отдельная processQA — следующий шаг root. Старые049
latency FAIL, D056 flaky FAIL, target VPS p95/RSS/start, TLS/physical2PC/providers
не закрыты. Самопроверка не является independent approve или wholeV1 PASS.

## Freeze и передача

Единственный список для root commit:

```text
internal/store/tasks.go
internal/store/handoff.go
internal/store/reopen_performance_fix_test.go
internal/store/handoff_performance_fix_test.go
docs/performance-runtime.md
docs/work/WORK-066-performance-fixes.md
```

Финальные Go SHA256:

| Файл | SHA256 |
| --- | --- |
| tasks.go | 92d345464c912fc0e371c2446aa55860ff8dcd7b1f43ce2082556b6f275ec211 |
| handoff.go | 02094901a1779778d83c9855dd69e7b6399d455a363650727835095ec099d98e |
| reopen_performance_fix_test.go | 9861457d57d4b7f0b10bcb7577461549aa794db0a0d0506d96d861e9de546757 |
| handoff_performance_fix_test.go | dbf342339a1c788bbd6abd2ac32068cc226831e288b4498d2649a4a4e000a5ac |

Полный six-file manifest с определением aggregate и hashes docs/WORK,
artifact manifest (включая inputs063, baseline sources, overlays, logs, results
и binary) — `.local/work066/source-manifest.json`, `artifact-manifest.json`.
Go-файлы <300 строк; исключений по размеру нет. Review/security064 на исходном
b9 независимо подтвердили те же2 blockers без дополнительных; их docs-only
HEAD b6076eef6bad219bb2137e9e4ad1fd612ada5e55 не меняет базу этой реализации.
Изменённые docs описывают actual fix, не принимают его самостоятельно.

- handed_off 2026-10-18T20:17:05+03:00: `/root/perffix066`, в работе → на независимой проверке;
  final source/tests/docs frozen, запись остановлена. Адресат root: fresh
  code/security re-review, затем отдельная processQA. Git commit/push/live нет.
