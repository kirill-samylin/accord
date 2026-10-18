# WORK-067 — pure E config и portability identity

Родитель TASK-003; developer `/root/integrationcfg067`, свежий экземпляр.
Статус: выполнено автором, frozen для независимой проверки; approve нет.
Вход HEAD `4bacf888a1eb28f32f3b8f2f8b1e0ce9a6c6c5e2`, ветка
`codex/integration-config`, checkout `.local/integration-config-003`.
Прочитаны Accord AGENTS, требования/архитектура, профиль developer, общий
процесс/parallel-work Давинчи, config/integration contracts и review046.
Tandem OFF по прямому указанию пользователя; named Claude bindings недоступны,
модель унаследована, независимо проверит отдельный новый агент.

## Границы и решения

Только internal/config semantic/types/schema/tests, обе config JSON Schema,
локальный identity validator + targeted interchange tests, эта WORK и
[инструкция067](../integration-foundations-067.md). Без новых зависимостей,
SQL/store/core/transport/CLI, миграций, Git commit/merge/push и live/provider.
Template не изменяется. Общая TASK и соседний Давинчи не редактируются.
Config → pure github.CanonicalRepository и notifications.ValidateURL не создаёт
цикла; regex repository и destination SSRF policy не дублируются.

Coordinator подтвердил: отсутствующий/[] allowlist допустим при пустом events,
это отсутствие доверия; непустые events требуют1..16 explicit repositories.
Всегда≤16 и unique после lower-case canonicalization. project.repository —
прежние D metadata, никогда не trust. Options только capability flags;
OAuth и runtime secret readiness≥32bytes не включаются.

Endpoint duplicate identity: SHA256(canonical URL + NUL + secret_env NAME),
hex lower-case; :443 удаляется existing ValidateURL, пустой root path → /.
Helper private: runtime позже должен переиспользовать единый алгоритм при
выделении API, а не копировать policy. Разные events не разделяют identity.
Общий validIdentity и validRef закрывают P3 REV046-02 в parser/exporter/refs;
Fact.Hash и обязательство отдельного SQL pointer hash не меняются.

## Журнал

- 2026-10-18T20:15:51+03:00 — created/assigned/started; developer integrationcfg067;
  чистый HEAD подтверждён, готово к работе → в работе; границы приняты.

## Проверки

UI/manual TC N/A: нет интерфейса или runtime wiring; reproducible
Go unit/integration cases и offline usage описаны в инструкции067. Новые
process/E2E — после независимого review, здесь не создаются.

- 2026-10-18T20:18:31+03:00 — completed/handed_off; developer integrationcfg067;
  в работе → выполнено автором; unit/race/vet PASS; exact freeze ниже,
  адресат — отдельный новый reviewer координатора. После записи freeze автор
  прекращает изменения до конкретного возврата. Это не independent approve.

## Фактическая верификация

Go1.27.2 darwin/arm64, module go1.26.1. Полные команды затронутых пакетов:

```sh
export GOTOOLCHAIN=local
export GOCACHE=/Users/shagi/Desktop/Project/agent-board/.local/toolchains/cache/build
export GOMODCACHE=/Users/shagi/Desktop/Project/agent-board/.local/toolchains/cache/mod
export GOTMPDIR=/Users/shagi/Desktop/Project/agent-board/.local/toolchains/tmp
/Users/shagi/Desktop/Project/agent-board/.local/toolchains/go/bin/go test -count=1 ./internal/config ./internal/interchange ./internal/notifications ./internal/integrations/github
/Users/shagi/Desktop/Project/agent-board/.local/toolchains/go/bin/go test -count=1 -race ./internal/config ./internal/interchange ./internal/notifications ./internal/integrations/github
/Users/shagi/Desktop/Project/agent-board/.local/toolchains/go/bin/go vet ./internal/config ./internal/interchange ./internal/notifications ./internal/integrations/github
git diff --check
```

PASS unit: config1.421s, interchange0.685s, notifications0.724s, github0.667s.
PASS race: config11.516s, interchange1.404s, notifications1.493s, github1.361s.
PASS vet/diff-check, exit0. Исходники Go≤300 строк, schema532 строки — прежнее
декларативное исключение. Existing malformed evidence/canonical JSON cases
сохранены. Distributable/embedded schema parity и template parity PASS.

Новые unit/integration cases: explicit trust vs metadata/capability; absent/[]
disabled; case duplicates, ASCII/256bytes и16/17 repos; immutable decoded nested
slices; HTTPS public-DNS syntax policy и diagnostics privacy; endpoint identity
:443/root/event subscriptions; four exact events vs pure notifications.Encode;
16/17 endpoints; synthetic env не expanded и secret readiness не проверяется;
старый template без включённых интеграций, auth.github всё ещё unsupported.
Interchange: empty/NUL/256/257bytes, Unicode byte boundary, все6 typed reference
positions, NUL не доходит mapping; Encode/Parse parity, finalfile/staging cleanup.

Ожидаемый FAIL исходного P3: overlay только HEAD export.go/validate.go, без
редактирования source tree; текущие targeted regression tests запускаются с:

```sh
/Users/shagi/Desktop/Project/agent-board/.local/toolchains/go/bin/go test -count=1 -overlay .local/evidence/work067/baseline-overlay.json -run 'TestIdentityParseExportParity|TestNULReferencesNeverReachMappingOrExport' ./internal/interchange
```

Exit1, `export identity length=6 valid=false: <nil>` и
`NUL reference reached authoritative mapping task <nil>`. Это подтверждение
старого дефекта, не скрытый FAIL текущей реализации. В исправленной версии
весь targeted набор PASS. Fact.Hash не менялся; обязательная equal-time SQL
regression с Fact.Hash + normalizedMatch + Binding остаётся runtime owner.

Logs и baseline overlay/copies — private `.local/evidence/work067/`:
unit.log, race.log, vet.log, diff-check.log, baseline-p3.log. manifest.json
фиксирует их SHA256 и templates unchanged. Freeze14 файлов (эта WORK включена
в private manifest после последней записи); ниже13 content hashes без selfhash.
Полный frozen diff — private source.diff; untracked additions включены.

NOT-RUN: полный product main check, новые process/E2E, SQL/schema5, durable
replay/outbox/import, runtime readiness/gates, реальные GitHub/providers,
DNS network, OAuth, live/VPS/deploy, Linux/Windows/второй ПК. Runtime E всё ещё
ждёт performance/schema4 independent approve+QA и собственной реализации.
Security/code independent review этого exact freeze — следующий этап.

## Freeze — явная область передачи

```text
d04c05ffd6825e2dd2aab1e07392d95bfa568da0e4d60959b20ffdab12e20b78  docs/integration-foundations-067.md
dd6ac1d31194a689cea59fcf4a1bf225e8655746aea2c8631ef456be53af1124  internal/config/agent-board-v1.json
bb7856a5e4514921a3216d9d0766b6dc03d9a12cfc7875306808ad6e06712f03  internal/config/evidence_test.go
bb2ec44080735fc5a1b07e9d59baa4d5e81ff925fd0967091c316bdfeac1b0b2  internal/config/integrations.go
7db6352f2dbb072d5f4f6a9bdb9abce7da7ba007ed9323836bd52c4593ec8c39  internal/config/integrations_test.go
a27ee66a94fd2a462f20a73e2a1150875f8384d7f8af2d14e843b7ab89456222  internal/config/semantic.go
a662be2268b22781e97bb487173f75e6e44ffd279725d7efc3fbebbd386e016e  internal/config/types.go
e842e5827624cac7c67dabe81a22db69301e250429688f13e781ec60e46a761a  internal/interchange/export.go
f2957b7978cffaff31e03738b6faedfe6d7e28c07b8eac68d3f6b4109a85055d  internal/interchange/identity.go
8b21d175015f02171ef9d33709cab34f0aa6c780bc1f549fe143f7f653969e73  internal/interchange/identity_test.go
2fa9d9107819995daac5248078031d8de23f8774f2257b78044fdab86d897ddb  internal/interchange/parse.go
7305b9ff7124ae3326113009727fe431352fa354ad1868a4a266e39c9749498a  internal/interchange/validate.go
dd6ac1d31194a689cea59fcf4a1bf225e8655746aea2c8631ef456be53af1124  schema/agent-board-v1.json
```
