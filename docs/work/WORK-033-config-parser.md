# WORK-033 — offline config parser и evidence schema

- Родитель: [TASK-003](../TASK-003-working-service.md).
- Роль: разработчик; экземпляр `configparser003`; статус: реализация и unit-проверки.
- Единственная область: `internal/config/**`, `schema/**`, `templates/**`,
  go.mod/go.sum (одна YAML dependency) и эта WORK.
- Входы: AGENTS, requirements1.1, service-contract D, config-contract§2/3/5/6;
  общий процесс, developer profile и parallel-work соседнего проекта прочитаны.
- Именованные модели Claude здесь не закреплены; используется унаследованная
  модель. Tandem отключён пользователем. Живой сервер/чужие файлы не меняются.

## Интерфейс и границы

`config.Parse(data, Options) (*Snapshot, Result)` не делает IO; Snapshot хранит
private canonical bytes, SHA256 и возвращает независимые копии через Decode /
CanonicalJSON. Options явно включает реализованные integration capabilities и
настоящие toolDescriptions транспорта. Result содержит typed Problems/warnings.
TTLs выдаёт task/role/session/heartbeat. RequiredFloor и ValidateEvidence
сохраняют встроенный минимум; Complete=false при отсутствии дополнительных
полей даже в warn/off. Lease/provenance/gates проверяются store, не parser.
ResolveArea и AreasIntersect не обращаются к файловой системе.

Полный Davinci template требует явной замены OWNER. Bootstrap compatibility
profile и active config/CAS/schema3 — следующий integration owner; здесь их нет.
Поставляемые schema/template зеркала embedded assets сверяются тестом.

Используется stable `go.yaml.in/yaml/v3 v3.0.5`; официальная YAML организация
поддерживает v3 security fixes, v4 пока только prerelease. Источники:
https://github.com/yaml/go-yaml (Version Intentions),
https://github.com/yaml/go-yaml/releases/tag/v3.0.5.
Ни одна существующая dependency не обновлена.

JSON Schema shape проверяется существующей google/jsonschema-go. Его v0.4.3
ошибочно распознаёт json.Number как string при type-check; только disposable
shape view использует native числа. Source/canonical и evidence schema numbers
сохраняются json.Number; restricted evidence validation использует exact big.Rat,
asserts URI/date-time и Go regexp без remote loader. Decimal exponent ограничен
32KiB до allocation big.Rat как защита ресурса, а не преобразование float.

## Журнал

- 2026-10-18T17:59:38+03:00 started: прочитаны контракты, назначены области,
  координатору передан API; dependency проверена по официальным источникам.
- 2026-10-18T18:06:05+03:00 implementation: strict YAML AST, shape/defaults,
  semantic refs/permissions/TTLs/safety/capabilities, evidence validator и tests.
  Initial targeted unit PASS; финальный race/vet и manifest впереди.

## Финальный результат и передача

Реализованы bounded YAML AST/JSON input, duplicate/tag/anchor/alias/merge/nonstring
rejection, unknown shape fields, role/category/user/reopen/tool references,
материализованные defaults → canonical SHA256, TTL и bridge hints, safety-rule
block floor, integration capability failures, path aliases/collisions, stable
12-tool grants/description-byte budget и отдельный schema-byte report.
Evidence: exact-number restricted vocabulary, local JSON Pointer refs/acyclic
limits, Go regexp, asserted formats, built-in byte/type/SHA/HTTPS floors,
custom schemas, block/warn/off completeness без обхода встроенного минимума.
Builtin field redefinition принимается только при exact canonical match;
ослабление схемы отклоняется явно. Details/API — `internal/config/README.md`.

- PASS: `go test -race ./internal/config -count=1`, 5.943s.
- PASS: `go vet ./internal/config`.
- PASS: FuzzParse, 3s fuzz budget, 310 executions + 4 baseline seeds, 4.270s total.
- PASS: `go mod verify`, all modules verified.
- PASS: owned `git diff --check`; все Go файлы ≤300 строк.
- PASS: schema/template mirrors equal embedded assets; canonical roundtrip and
  snapshot isolation, hostile inputs, runtime capability rejection, exact >2^53
  evidence bounds, refs/cycles, Unicode byte limits, area collisions.
- NOT-RUN: independent review, process/E2E, schema3/CLI/live apply integration.
- Общий полный suite не запускался: непересекающаяся параллельная C-разработка
  ещё в работе; package-level checks — фактический объём этого автора.
- Git commit/push/deploy и общая TASK этим исполнителем не менялись.
- go.mod/go.sum: только direct stable go.yaml.in/yaml/v3 v3.0.5 и два checksum;
  промежуточная v3.0.4 удалена до передачи, старые зависимости не менялись.

2026-10-18T18:10:46+03:00 completed: targeted race/vet/fuzz PASS; source frozen
для независимого review и последующей store/CLI интеграции. Manifest в
`.local/evidence/config-parser-003/manifest.sha256`; хеш передан координатору.

2026-10-18T18:11:55+03:00 freeze correction: default note description aligned
with the current transport's actual embedded text. Targeted race rerun PASS
7.574s and vet PASS; no behavior or grant changes. Manifest regenerated.
