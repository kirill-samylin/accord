# WORK-002 — Изолированный Go toolchain

- Родитель: [TASK-001](../TASK-001-pilot.md).
- Роль: DevOps-инженер; профиль `davinci-gui-/docs/agents/roles/devops.md` прочитан.
- Приоритет: высокий, снимает отсутствие компилятора для разработки ядра.
- Статус: выполнено; принятие результата — координатором.
- Исполнитель и запуск: `/root/toolchain`, 18.10.2026; редактор этой WORK — он же.
- Ветка: `codex/pilot`, незакоммиченные файлы; хеши результата ниже.
- Область записи: `.local/toolchains/`, `.local/evidence/toolchain/`,
  `docs/development.md`, эта WORK. Корневые go.mod/go.sum не редактировались.
- blocked_by: нет. related_to: WORK-003 (разработчик ядра).
- Ограничение среды: именованная модель Sonnet здесь не закреплена;
  используется модель текущего запуска по правилу корневого AGENTS.md.
- Tandem не используется по прямому указанию пользователя.

## Цель и входы

Подготовить воспроизводимый Go runtime без системных изменений. Входы:
корневой AGENTS.md, TASK-001, профиль DevOps, общий процесс и формат WORK.
Проектные требования ещё формируются, поэтому подготовка не задаёт поведение API.
Согласованная область не включает CI, release, remote или сервисные зависимости.

## Результат

Установлен Go 1.27.2 darwin/arm64 в
`/Users/shagi/Desktop/Project/agent-board/.local/toolchains/go`.
Версия и архив выбраны из `https://go.dev/dl/?mode=json`, checksum проверен:
`76812b213b1b2302c978d28fa52fa92d541704b9e7d9d5db8002c50e4018c4c5`.
Официальный архив скачан по HTTPS; распаковка выполнена после проверки SHA-256.
Никакие системные файлы, настройки shell и соседний проект не изменялись.

[Инструкция](../development.md) содержит окружение, установку, команды
build/vet/test/race корневого модуля и ограничения. Кэши и GOTMPDIR находятся
в `.local/toolchains/`; `GOTOOLCHAIN=local` запрещает автоматическую подмену Go.

Официальный proxy подтвердил и скачал с checksum database:
`modernc.org/sqlite@v1.60.1` (Go 1.26.0),
`github.com/modelcontextprotocol/go-sdk@v1.8.0` (Go 1.25.0).
Документация SQLite подтверждает pure Go без cgo; зависимости сервиса выбирает
и закрепляет WORK-003. Это не runtime-проверка драйвера или SDK-клиентов.

## Проверки и версия

- PASS: `go version` → `go version go1.27.2 darwin/arm64`.
- PASS: `go build` минимального main.go и запуск → `agent-board toolchain OK`.
- PASS: тот же пример с `CGO_ENABLED=0`.
- PASS: `CGO_ENABLED=1 go build -race` и запуск; доступен clang/Command Line Tools.
- PASS: `go mod download -json` для обоих указанных модулей без изменения
  корневого модуля. JSON содержит подтверждённые module sums.
- NOT-RUN: тесты/build/vet/race сервиса — исходники отсутствовали на этапе WORK.
- NOT-RUN: Linux/Windows, физические две машины и реальные клиенты Codex/Claude.
- N/A: UI, развёртывание и remote не входят в эту инфраструктурную подготовку.

Локальные доказательства: `.local/evidence/toolchain/` (JSON релизов/модулей,
checksum, go-version.txt, go-env.txt, исходник smoke и бинарные файлы).
Проверенные SHA-256:

- `docs/development.md`: `76e5ce338f46d72f28514b9c16af271ce4789f67fc5d13ac3b401f7b9c746c7a`.
- smoke main.go: `5b944a8665001b9ac8cfc19b05a45111d2546f3aa91c818dc6822e12fd45b1df`.
- selected-go.json: `6f2e3b195b5b8ff840b369835ba25dd57b64d55c7837b8168f59b99947dc42bd`.

## Передача

Следующий исполнитель: WORK-003 и координатор. Runtime готов, команды окружения
переданы координатору. Родительскую TASK эта WORK не завершает; независимое
ревью проектного результата и acceptance выполняются далее отдельно.

## Журнал событий

| ID | Время | Тип | Автор | Статус | Причина / доказательство |
| --- | --- | --- | --- | --- | --- |
| 002-1 | 2026-10-18T12:36:21+03:00 | started | /root/toolchain, DevOps | в работе | Прочитаны входы; начаты официальные downloads |
| 002-2 | 2026-10-18T12:37:35+03:00 | completed | /root/toolchain, DevOps | выполнено | SHA-256, version, build/run, purego/race smoke PASS; передача WORK-003 |
