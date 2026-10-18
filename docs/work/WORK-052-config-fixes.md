# WORK-052 — исправления полного review package D

- Исполнитель: fixd052, developer, свежий экземпляр WORK052-20261018.
- Статус: в работе. Координатор — единственный редактор общей TASK.
- Checkout: `.local/config-fixes-003`, `codex/config-fixes`.
- База: `932aa2a3017da0b4fc06032e5134d4ed8fdf2d5b`; frozen WORK043 не меняется.
- Входы: AGENTS, requirements/service-contract/config-contract/configuration-runtime,
  WORK043; MAIN review050/security051, единый пакет REV050-01…06.
- Прочитаны developer profile, общий процесс и parallel-work из Davinci.
  Именованные Claude модели не закреплены в этой среде; наследуемая модель.
- Только source/unit/integration этого checkout, этот WORK и private evidence.
  Tandem OFF; без claims, Davinci/main/чужих отчётов, commit/push/live.
- Новые process/E2E принадлежат отдельному tester после code/security approve.

## Журнал

| Время | Событие | Исполнитель | Статус | Причина |
| --- | --- | --- | --- | --- |
| 2026-10-18T19:08:37+03:00 | started | fixd052 | в работе | Получен полный пакет шести подтверждённых defects, читается frozen source и отчёты |

## Исправленный объём

| Review ID | Изменение | Собственная регрессия |
| --- | --- | --- |
| REV050-01 | Исторический вид role lease определяется сохранённым RoleGeneration>0; sessionLive, heartbeat и renewal не зависят от текущего singleton flag. Schema delta не нужна. Legacy nil role_expires_at сохраняет session deadline + fencing поколения | TestSingletonExpiryRemainsFencedAcrossConfigChanges; TestLegacySingletonWithoutRoleDeadlineUsesGenerationAndSessionExpiry |
| REV050-02 / S051-01 | areaRoleAllowed общий для claim и live Areas update, проверяется после canonicalization и ownership; coordinator planning незахваченной работы сохраняется | TestLiveAreaUpdatesRequireEditorRole: off/warn/block × alias/child; неизменность task/version/lease/audit, foreign session/stale generation, разрешённый analyst |
| REV050-03 | Для отмены epic всегда действует canCancel: configured grant либо requester; имя human/coordinator не добавляет прав | TestEpicCancellationUsesConfiguredGrantOrRequester: обе роли × grant false/true × чужой/свой requester с отдельным token principal |
| REV050-04 | epicClosable использует successful для done child; explicit closure и reconcile используют единый положительный gate; старые superseded/stale verifier cycles сохраняют прежнее исключение | TestIncompleteChildEvidenceBlocksEpicClosureAndReconciliation: warn/off, reject explicit close, persisted pre-fix epic reopens, history неизменна, полное evidence закрывается |
| REV050-05 | Blocking overlap входит в чистый eligibleClaim; поиск не изменяет пропущенный task/owner/warnings. Explicit task_key возвращает AREA_OVERLAP. Queue Describe/Next и totals/groups Stats используют общую классификацию | TestAutomaticQueueSkipsAreaBlockedCandidate: busy priority2 перед free priority3; skipped epic остаётся unowned; queue counts согласованы |
| REV050-06 | CreateInput.Areas использует omitzero вместо omitempty: omitted сохраняет наследование, nonnil [] сохраняет отсутствие наследования и отдельный hash; optional MCP schema не меняется | TestCreateAreasPresenceIsPreservedForIdempotency; TestCreateAreasRemainOptionalAndPreserveExplicitEmpty |

Новые DTO/миграции/зависимости не вводились. Optional pointers ready/areas у
update проверены на различение omitted и явного false/[]; остальные D pointer
поля уже сохраняют наличие через ненулевой указатель. Других mutation slice с
разной nil/empty семантикой и omitempty не найдено: Events.Types — read filter;
TokenInput.Roles defaults по len==0, не отдельная nil/[] команда.

Выбор omitzero проверен по [encoding/json Marshal](https://pkg.go.dev/encoding/json#Marshal)
и [jsonschema ForType](https://pkg.go.dev/github.com/google/jsonschema-go/jsonschema#ForType),
а также по локальному исходнику зафиксированного jsonschema-go v0.4.3.
Тег сохраняет optional schema без ручного исключения transport. Go.mod1.26.1
поддерживает его; фактический toolchain go1.27.2 darwin/arm64.

## Изменённые файлы

Production: internal/core/types.go; internal/store/claim_rules.go, gates.go,
mutation.go, notes.go, runtime_roles.go, sessions.go, stats.go, tasks.go,
transitions.go. Unit/integration: internal/store/config_fixes_areas_test.go,
config_fixes_sessions_test.go, config_fixes_gates_test.go,
config_fixes_idempotency_test.go; internal/transport/config_fixes_test.go.
Документация в границах задания — только этот WORK; эксплуатационные инварианты
configuration-runtime сохранены и теперь применяются также на обходных путях.

## Фактические проверки

Собственные evidence находятся в `.local/work052/` этого checkout. Чужие runtime
логи review050/security051 прочитаны как входы, не засчитываются своим PASS.
Все команды используют абсолютный MAIN `.local/toolchains/go/bin/go`,
GOTOOLCHAIN=local, GOCACHE=MAIN/.local/toolchains/cache/build,
GOMODCACHE=MAIN/.local/toolchains/cache/mod, GOTMPDIR=MAIN/.local/toolchains/tmp.

| Проверка | Результат | Evidence |
| --- | --- | --- |
| Новые8 top-level unit/integration store+transport | PASS | regressions-pass.log |
| Те же новые тесты поверх production HEAD932aa2a через private overlay | Ожидаемый FAIL:6 defect suites + отдельная serialization suite; legacy compatibility PASS | baseline-regression-proof.log, baseline-overlay.json; frozen043 не изменялся |
| Затронутые store/bridge/transport,50 top-level | PASS | targeted.log |
| Новые сценарии + CAS/concurrent claims/global current verification, race11 top-level | PASS | race.log |
| go vet ./cmd/... ./internal/... | PASS | vet.log |
| git diff --check;15 изменённых Go файлов≤300 строк | PASS; максимум271 | checks.json |
| Full cmd/internal/runtime/process suite | NOT-RUN повторно по границе задания; предыдущий baseline не объявляется новым PASS |
| Новые process/E2E, два процесса CLI apply/reload, backup/restore/crash/old binary | NOT-RUN; отдельный tester после актуального approve |
| Независимые code/security review новой дельты | NOT-RUN автором; передача свежим проверяющим |
| VPS/deploy/live apply/merge/push/физические2ПК/UI/providers | NOT-RUN; вне этого WORK |

Целевой прогон включает существующие tests: stale/current SHA+generation,
latest/superseded verifier obligations между эпиками, каскадное reconciliation,
negative review/test gates, claim/lease expiry, rejection rollback/replay,
dynamic tools/role/human checks, overlap warn, notes и totals/groups statistics.
Новая source дельта не перепроверяет историческое evidence по новой schema.

Ранние авторские retries в regressions.log/regressions-final.log содержат
исправленные ошибки тестовых входов (nil aliases в строгой config schema) и
сравнения nil/empty Go DTO после JSON replay. Они не являются финальным
результатом проверки source; окончательный протокол — regressions-pass.log.

## Source freeze и передача

- Проверенная версия: base HEAD932aa2a + собственная uncommitted source дельта.
- Aggregate15 Go source SHA256:
  `13899d45e8fddb4e6074d2b1395fbe0274245295f26d07311c5747ef5a1f2732`.
- Freeze: `.local/work052/source-manifest.json`, source.patch и changes.patch
  (последний включает этот WORK). Manifest фиксирует каждую source SHA256,
  patch SHA256 и отдельный SHA256 WORK и финальных evidence.
- После freeze source/tests не менять до приёмки либо явного возврата.
- Root получает все6 исправлений одним пакетом для свежих code/security review;
  это author completion, не независимый approve и не завершение TASK003.
- Старые process fixtures намеренно сохранены: persisted compatibility TTL15m
  против serve30s/1s; обязательный bootstrap owner; schema ожидание2→3. Нельзя
  ослаблять startup/security guard ради их прохождения.

| Время | Событие | Исполнитель | Статус | Причина |
| --- | --- | --- | --- | --- |
| 2026-10-18T19:15:04+03:00 | completed | fixd052 | на проверке | Все6 review defects исправлены;8 новых regressions,50 targeted,11 race и vet PASS; source freeze перед свежим independent review |
