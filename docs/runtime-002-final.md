# Финальный ретест TASK-002

**PASS** согласованного локального пилота после WORK-015 и независимого
APPROVE [WORK-016](audit-002-recheck.md). Проверял отдельный tester
`finaltest002`, WORK-017, 18.10.2026 16:10–16:11 Europe/Moscow.
Исходники, тесты и зависимости не изменялись. База `21cb3e0` с frozen-дельтой;
43 Go/build/module-файла до и после совпали с reviewer manifest SHA-256
`699678fd6d9912d554ea10e4b663db5de84d8eda48d4c9652a54ec6f81343432`.

Окружение: пользовательский Mac, macOS27.0.1 arm64, Go1.27.2 darwin/arm64;
go.mod объявляет Go1.26.1. Использованы локальные GOCACHE/GOMODCACHE/GOTMPDIR
из Makefile и GOTOOLCHAIN=local. Все данные и процессы синтетические,
без подключения к реальной доске, Tandem или VPS.

## Результаты

| Проверка | Результат | Exit | Время |
| --- | --- | --- | --- |
| make check, umask077 | PASS: format, vet, все unit/integration/process, race, build | 0 | 10.886s |
| File-security fixtures, umask022, count1 | PASS: 4 верхних сценария, включая sidecar public/symlink/dangling | 0 | 0.519s |
| go mod verify | PASS: all modules verified | 0 | 1.053s |
| govulncheck ./... | PASS: No vulnerabilities found | 0 | 3.304s |
| Linux amd64, CGO0, build | PASS: только компиляция | 0 | 1.035s |
| Windows amd64, CGO0, build | PASS: только компиляция | 0 | 8.869s |

Для исключения кэширования `make check` использовал локальную обёртку Go:
она добавляет `-count=1 -v` исключительно к `go test`, остальные команды
передаёт штатному Go без изменений. Это не переносит неподдерживаемый
GOFLAGS в дочернюю сборку process-тестов. Makefile не редактировался.
Обычный process-набор прошёл за3.830s, тот же набор в race-запуске за4.410s:

| Существующий process-тест | Обычный | Race-запуск |
| --- | --- | --- |
| TestCLIConcurrentClaim | PASS | PASS |
| TestCLIArchitectureHandoff | PASS | PASS |
| TestCLILeaseExpiry | PASS | PASS |
| TestCLIRetryAndFilteredPagination | PASS | PASS |
| TestCLIRestartAndRevocation | PASS | PASS |

Проверены настоящие отдельные процессы CLI/server с временной SQLite.
Race инструментирует unit/integration и Go test/harness. Существующий
acceptance TestMain собирает дочерний CLI обычным `go build`; дочерний сервер
в process-сценариях не инструментирован race detector. Это ограничение
существующего набора, а не отдельный подтверждённый race-прогон сервера.

govulncheck: существующий scanner `v1.9.0`, Go1.27.2, официальная БД
`https://vuln.go.dev`, DB updated `2026-10-08 22:31:09 UTC`. PASS означает
результат этой проверки и этой версии БД; это не полный security-аудит.

## Воспроизведение

Из корня checkout задать PATH и GOTOOLCHAIN/GOCACHE/GOMODCACHE/GOTMPDIR,
как в [предыдущем протоколе](runtime-002.md). Содержимое локальной обёртки
`.local/evidence/final002/go-uncached`:

```sh
#!/bin/sh
if [ "$1" = test ]; then
  shift
  exec /Users/shagi/Desktop/Project/agent-board/.local/toolchains/go/bin/go test -count=1 -v "$@"
fi
exec /Users/shagi/Desktop/Project/agent-board/.local/toolchains/go/bin/go "$@"
```

```sh
umask 077
make check GO="$PWD/.local/evidence/final002/go-uncached"
umask 022
go test ./internal/store -count=1 -v -run 'TestDatabase(PrivateBeforeSQLiteIOInPublicDirectory|RejectsUnsafeExistingFilesWithoutChangingTargets|RejectsPubliclyWritableDirectory|RejectsSQLiteURIWithoutChangingItsTarget)$'
umask 077
go mod verify
.local/evidence/audit-002/bin/govulncheck ./...
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o .local/evidence/final002/agent-board-linux-amd64 ./cmd/agent-board
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o .local/evidence/final002/agent-board-windows-amd64.exe ./cmd/agent-board
```

## Доказательства

Локальный игнорируемый префикс `.local/evidence/final002/`. JSON содержит
времена, команды, exit codes и summary; raw logs не публикуются в Git.
Acceptance binary SHA в обоих прогонах совпал с итоговым `bin/agent-board`.

| Файл | SHA-256 |
| --- | --- |
| source-before.sha256 и source-after.sha256 | 699678fd6d9912d554ea10e4b663db5de84d8eda48d4c9652a54ec6f81343432 |
| environment.json | 31b76409b4c17d66d17c954cf16a6b234bf8cea74a89723770391b294dc3f84c |
| results.json | 9762561cdc5bcdb2237eed988d26d88909d190ceb67affbca39004dd0709de17 |
| summary.json | d46f0a34c3cc5f0bf432dbee3c0faf4fbdda3f0b56abc1fcd2aa113573146c96 |
| make-check077.log | 32f01e0b2559a9b8e1ae9e5ad8280ea60d55c49cab217a48d4ed2ebc0198ee65 |
| fixtures022.log | ab31eae84021e821db7dc62b1af8ec14965aa2a8546b89f3ecb6c97e818d7820 |
| mod-verify.log | b4537ed75f533f993f371954de47e42a793b8e5b0587577de7e27fb3e50696bd |
| govulncheck-version.log | ae907a32d1f6f4b4b5bcd6c557a0560b68ed860e989cf0ab785cc320bc095fd9 |
| govulncheck.log | 3016e51e4eac0d421674d2128bbbdefb2924b4646e0c14a1ab034977ad73fae5 |
| windows-build-result.json | 5f2c5cec7a8b12df37e001c57888d1549a8b0655a8b9383212743dbee74d6568 |
| go-uncached | cd466c3de4ac6b27d2418a1d15b83a20b063baa308c9b2a06cb2086ba87309ff |
| bin/agent-board — вне префикса | b7abf6b5925b83a5cbba984b47d967735a5d3583bd2253ea8fe206a051234eee |
| agent-board-linux-amd64 | 822969fcc9212282f6c039d26be7142500e1302bbba2e60493e596bf88641482 |
| agent-board-windows-amd64.exe | a141718895cda62a7b7f60523be0b528afe91dd4406d371d1f9092a57e253112 |

## Границы вывода

Блокирующих сбоев финального набора нет; исходный umask077 FAIL закрыт
исправлением fixtures и фактическим полным ретестом. Новые тесты здесь
не писались, их независимый approve принадлежит WORK-016.

NOT-RUN этим tester: реальные Codex/Claude, второй физический ПК,
Linux/Windows runtime, удалённый Accord/TLS/OAuth, нагрузочные цели V1,
power/network/disk fault injection. Отдельный реальный Codex smoke описан
координатором в [codex-002](codex-002.md), здесь не повторялся.
UI/design review N/A: UI в пилоте нет. Публикацию выполняет координатор;
этот PASS не означает полную V1 или готовность удалённого сервера.
