# WORK-176 — приватная диагностика native preflight

Автор: managed_worker. Ветка `codex/native-preflight` от опубликованного
`26d833f`. Независимый контракт WORK169 APPROVE получен до реализации.
Область: internal/worker preparation/discovery diagnostic + авторские unit tests;
служба, local config/binary, GUI, production и native starts не изменялись.

## Основание

Первый реальный GUI native запуск WORK172 завершился до conversation с общим
сообщением ошибки isolated native MCP preparation. Provider session отсутствует,
задача не получала lease. Приватный код конкретного этапа не сохранялся.

Read-only config-only пробы в фактических GUI cwd и LaunchAgent environment
успешно прочитали discovery и проверили exact immutable bridge overrides.
Эти результаты не устанавливают причину первоначального отказа. Возможная
ошибка readNative/ownership/Wait остаётся гипотезой; данный change диагностический,
не исправление доказанного runtime root cause.

## Реализация и приватность

Новый unexported preparationFailure хранит allowlisted stage и исходную cause.
Error() всегда возвращает прежнее общее сообщение; Unwrap сохраняет errors.Is/As,
включая errOwnership. В journal RunState.local_diagnostic записывается только
фиксированный stage. Если вложенная ошибка содержит ownership diagnostic,
сохраняется composite `<fixed preparation stage>__<valid ownership code>`.
Все части проходят закрытые allowlists, результат ограничен128 ASCII bytes;
unknown/forged stages заменяются фиксированными invalid/unspecified codes.
Ни paths, executable/connector names, PIDs, argv, env, stdout/stderr, prompt,
provider output, token, host proof, ни тексты исходных cause не сериализуются.
Receipt/API/UI fields не расширялись; сообщения failed остаются общими.

Различаются:

- immutable execution binding, private run/attempt directories, bridge files,
  local executable lookup и Claude config publication;
- native discovery и post-override verification: start/read/size/timeout/wait,
  JSON shape, metadata, transport и exact binding mismatch;
- incompatible reserved/unknown override transport;
- journal/fatal/closed/state changes, deadline, authorization и pinned paths
  до и после внешних configuration reads.

readNative получил unexported fake-process starter seam. Production путь всегда
использует прежний startProcess. Пять секунд, limit1MiB, stop/wait calls,
process census/PGID proof, workspace lock, session binding, permission/isolation
и deadline fencing не ослаблены. Atomic timeout flag только различает причину
отказа. В cleanup errors.Join сохраняет ownership cause, отдавая приоритет
final Wait proof перед Stop/read errors. Никакой новый doctor CLI/RPC/config
surface не добавлен.

## Файлы

`internal/worker/preflight_diagnostic.go`,
`preflight_diagnostic_test.go`, `preflight_launch_test.go`,
`discovery.go`, `codex_config.go`, `bridge_config.go`, `preparation.go`,
`execution.go` и эта WORK. Каждый source/test файл менее300 строк.

## Проверки

PASS: worker+cmd unit/integration с race detector. Новые unit cases используют
только fake process/I/O: start/read/limit/Wait failures, underlying ownership
uncertainty и cleanup preservation; exact1MiB success boundary; реальный5s timer
на fake blocked process; invalid stages/forged ownership; parser/transport/binding;
private files/immutable attempts; post-inspection deadline/pins/controller refusal,
no native spawn и private journal persistence без diagnostic в public receipt.

При первом авторском запуске один новый тест поймал ошибку собственной closure
fixture (ожидаемый composite stage передавался как исходный stage); исправлена
fixture, production code ради этого не ослаблялся. Финальный race run прошёл.

PASS: Windows amd64/Linux amd64 test cross-compilation и git diff --check.
Это не native Windows runtime и не проверка двух физических ПК.

NOT-RUN: новые actual config-only process probes и native conversation после
этой дельты. Их пишет/выполняет отдельный QA177 только после source APPROVE;
новый native запуск также требует отдельного GO координатора. Новое поручение
пользователя проверить другой проект/третьего участника требует собственных
project/controller IDs и локального scope; GUI IDs для него не переиспользуются.
Обязательная native приёмка остаётся открытой, root cause не объявлен fixed.
