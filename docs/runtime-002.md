# Повторная проверка на Mac — RUN audit-002

Дата: 18.10.2026. Исполнитель: отдельный tester `runtime002 / WORK-013`.
Проверена версия `21cb3e0de49fab4696f60b6131ccd4be2534e9b7`; продуктовые
исходники, тесты, go.mod/go.sum исполнителем не менялись. Это проверка
локального пилота, не завершение V1 и не проверка второго физического ПК.

Окружение: macOS 27.0.1 (26A434), Darwin 27 arm64; локальный Go 1.27.2
darwin/arm64. Инструменты и кэши находятся внутри `.local/`; системная
установка не менялась. Роль передана профилем tester из соседнего проекта;
закрепление именованных Claude-моделей здесь недоступно. Tandem не подключён.

## Протокол

| Проверка | Результат | Фактическое наблюдение |
| --- | --- | --- |
| `umask 077; make check` | **FAIL** | fmt/vet прошли, unit test fixtures зависят от umask; make остановился до race/build. |
| `umask 022; make check` | **PASS** | fmt/vet, все пакеты, race всех пакетов, сборка бинарника. В выводе нет cached: тесты выполнялись. |
| `go mod verify` | **PASS** | `all modules verified`. go.mod/go.sum не изменились. |
| `govulncheck ./...` | **PASS** | v1.9.0, Go 1.27.2, база https://vuln.go.dev обновлена 08.10.2026 22:31:09 UTC; `No vulnerabilities found.` |
| Linux amd64 / `CGO_ENABLED=0 go build` | **PASS, только сборка** | ELF бинарник собран кросс-компилятором Mac; на Linux этим исполнителем не запускался. |

В обычном прогоне unit/integration пакеты: cmd 0.260s, store 0.578s,
transport 0.803s, acceptance 3.590s. Race: cmd 1.510s, store 4.408s,
transport 2.227s, acceptance 4.970s. Это времена выполнения тестов, не
показатели задержки сервиса. Пакет acceptance запускает пять существующих
сценариев с настоящими CLI/server/SQLite: concurrent claim, handoff архитектору,
истечение lease, idempotent retry/events pagination, restart/revoke.
Новые E2E в этом задании не создавались; описание AC-01…07 — в
[acceptance.md](acceptance.md).

Официальный сканер установлен изолированно в evidence-каталог, версия
`golang.org/x/vuln/cmd/govulncheck@v1.9.0` зафиксирована после запроса latest.
Подход сверён с [инструкцией Go](https://go.dev/doc/security/vuln/).
Сканирование означает отсутствие известных находок в доступной базе для
данного дерева/платформы, а не независимый аудит всех зависимостей.

## Обнаруженный дефект воспроизводимости

`cmd/agent-board/client_test.go:115` создаёт якобы публичный токен через
`os.WriteFile(..., 0644)`, но umask 077 фактически оставляет 0600. Корректный
readToken принимает приватный файл, и тест ошибочно пишет `accepted public
token file`. Нужен явный `os.Chmod(..., 0644)` для синтетического fixture.

`internal/store/files_unix_test.go:69` аналогично создаёт `foreign` без
последующего Chmod, а на строке 96 ожидает именно 0644. С umask 077
исходный файл уже 0600; сообщения `foreign target was changed` ложны.
Тестируемый сервис не был замечен в изменении чужого файла. Для fixture
нужно явно установить ожидаемые права до вызова сервиса.

Замечание немедленно передано координатору; исправления делает разработчик,
после них требуется ретест. PASS с umask 022 не отменяет этот FAIL.

## Воспроизведение

Из корня checkout:

```sh
export PATH="$PWD/.local/toolchains/go/bin:$PATH"
export GOTOOLCHAIN=local
export GOCACHE="$PWD/.local/toolchains/cache/build"
export GOMODCACHE="$PWD/.local/toolchains/cache/mod"
export GOTMPDIR="$PWD/.local/toolchains/tmp"
umask 022
make check
go mod verify
GOBIN="$PWD/.local/evidence/audit-002/bin" \
  go install golang.org/x/vuln/cmd/govulncheck@v1.9.0
.local/evidence/audit-002/bin/govulncheck ./...
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
  -o .local/evidence/audit-002/agent-board-linux-amd64 ./cmd/agent-board
```

Воспроизведение дефекта: перед `make check` выставить `umask 077`.
Все runtime-данные тестов синтетические. Секреты и БД не сохранены в отчёт.

## Доказательства

Файлы локальные, игнорируемые Git, префикс `.local/evidence/audit-002/`.
`source-before.sha256` содержит SHA-256 проверенных исходников, тестов,
Makefile и go.mod/go.sum. Окружение сохранено в `environment.json`.

| Файл | SHA-256 |
| --- | --- |
| source-before.sha256 | 3fb2213f3399c3e9a3fef9232c49f5105bd880bbfe646d2a7efff7c541389bf5 |
| make-check.log — umask077 FAIL | a02d86e1d9bdb05ec1c431b6407ee58b966af23909aef3f36e8f80fcc4de9145 |
| make-check-umask022.log — PASS | 69c12d7ec90352de7302357d1e15aae78409a3f4273d321103014591fb23f3f4 |
| mod-verify.log | b4537ed75f533f993f371954de47e42a793b8e5b0587577de7e27fb3e50696bd |
| govulncheck.log | 3016e51e4eac0d421674d2128bbbdefb2924b4646e0c14a1ab034977ad73fae5 |
| govulncheck-version.log | ae907a32d1f6f4b4b5bcd6c557a0560b68ed860e989cf0ab785cc320bc095fd9 |
| agent-board-darwin-arm64 | 935b29ff1469eab5f6917993895f7711a8d193436afadb3d01c877506f72ddf5 |
| agent-board-linux-amd64 | 608885f0cca1dc6578d017c8b274dbde50935f4de3c938fc472e74d9d51ab67d |

## Ограничения

NOT-RUN этим исполнителем: настоящие Codex/Claude, MCP через внешний
агентский клиент, Linux/Windows runtime, два физических ПК, удалённый Accord,
TLS proxy/OAuth, нагрузка/p95/RSS, power/network/disk fault injection,
отдельный ручной протокол. Реальный Codex и VPS проверяют другие исполнители;
их результаты не подменяются этим отчётом. UI/визуальная приёмка N/A.
Push/PR/merge/release этим исполнителем не выполнялись.
