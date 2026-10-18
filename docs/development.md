# Локальная разработка

Проверено 18.10.2026: macOS arm64, Go 1.27.2. Runtime установлен только
в `.local/toolchains/go`; системный PATH и пакеты не изменялись.
Модуль сервиса находится в корне проекта. `.local/` не включается в Git.

## Окружение

Из корня проекта выполните в текущем терминале:

```sh
export PATH="$PWD/.local/toolchains/go/bin:$PATH"
export GOCACHE="$PWD/.local/toolchains/cache/build"
export GOMODCACHE="$PWD/.local/toolchains/cache/mod"
export GOTMPDIR="$PWD/.local/toolchains/tmp"
export GOTOOLCHAIN=local
export GOPROXY=https://proxy.golang.org
export GOSUMDB=sum.golang.org
mkdir -p "$GOCACHE" "$GOMODCACHE" "$GOTMPDIR" .local/bin
go version
```

Абсолютный путь runtime в текущем checkout:
`/Users/shagi/Desktop/Project/agent-board/.local/toolchains/go/bin/go`.
`GOTOOLCHAIN=local` исключает незаметное скачивание другой версии компилятора;
если зависимость потребует более новый Go, сборка завершится ошибкой.
Кэши общие для локальных исполнителей, отдельный модульный cache не нужен.

## Повторная установка Go на macOS arm64

Официальный архив: [Go 1.27.2](https://go.dev/dl/go1.27.2.darwin-arm64.tar.gz).
Версия и SHA-256 сверены с [официальным JSON](https://go.dev/dl/?mode=json).
Распаковывайте в пустой каталог назначения; не накладывайте новую версию
на содержимое старой установки.

```sh
mkdir -p .local/toolchains
curl --fail --show-error --location \
  https://go.dev/dl/go1.27.2.darwin-arm64.tar.gz \
  -o .local/toolchains/go1.27.2.darwin-arm64.tar.gz
printf '%s  %s\n' \
  76812b213b1b2302c978d28fa52fa92d541704b9e7d9d5db8002c50e4018c4c5 \
  .local/toolchains/go1.27.2.darwin-arm64.tar.gz | shasum -a 256 -c -
```

Продолжайте только после `OK` и при отсутствии `.local/toolchains/go`:

```sh
test ! -e .local/toolchains/go && \
  tar -xzf .local/toolchains/go1.27.2.darwin-arm64.tar.gz -C .local/toolchains
.local/toolchains/go/bin/go version
```

На другой ОС выберите её архив и checksum в официальном JSON. macOS-архив
для Linux/Windows не подходит. Установка системных пакетов и `sudo` не нужны.

## Сборка и проверки

После появления корневых `go.mod`/`go.sum` и исходников, с окружением выше:

```sh
go mod download
go mod verify
CGO_ENABLED=0 go build ./...
go vet ./...
go test ./...
CGO_ENABLED=1 go test -race ./...
```

`go build ./...` компилирует все пакеты; имя исполняемого файла и параметры
сервера описаны в README после реализации CLI. Race detector требует C toolchain:
на проверенном Mac доступны `/usr/bin/clang` и Command Line Tools SDK. Для обычной
сборки с pure-Go SQLite `CGO_ENABLED=0` исключает системный SQLite/C toolchain.
Не меняйте `go.mod`/`go.sum` только ради запуска проверки: их владелец — разработчик.

## Доступность зависимостей

Официальный Go proxy и checksum database подтвердили скачивание:

| Модуль | Проверенная версия | Требует Go | Источник |
| --- | --- | --- | --- |
| `modernc.org/sqlite` | `v1.60.1` | 1.26.0 | [Документация автора](https://pkg.go.dev/modernc.org/sqlite@v1.60.1) |
| `github.com/modelcontextprotocol/go-sdk` | `v1.8.0` | 1.25.0 | [Официальный SDK](https://github.com/modelcontextprotocol/go-sdk) |

SQLite-драйвер описан автором как pure Go без cgo. Это проверка наличия
и метаданных, а не подтверждение транзакций сервиса или совместимости клиентов.
Выбор закреплённых зависимостей остаётся за реализацией и независимым ревью.

## Фактический результат подготовки

- PASS: официальный SHA-256 архива, `go version`, компиляция и запуск минимального
  примера, скачивание двух закреплённых модулей через проверку Go checksum database.
- PASS: минимальный пример собирается с `CGO_ENABLED=0` и с race detector.
- NOT-RUN: build/vet/test/race сервиса в этой WORK — исходников ещё нет.
- NOT-RUN: Linux/Windows, две физические машины, Claude Code/Codex и production.

Доказательства текущей локальной подготовки — `.local/evidence/toolchain/`.
Они содержат только метаданные runtime/модулей и тестовый вывод без секретов.
