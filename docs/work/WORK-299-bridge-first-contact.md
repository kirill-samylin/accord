# WORK299 — первый контакт моста: machine токена и сбой сети до сессии

Исполнитель developer, ветка `codex/bridge-first-contact` от
`codex/native-preflight` (6558880). **UNIT PASS / стенд NOT-UPDATED до
поручения**. Поручение пользователя 26.10.2026 во время прогона WORK298:
«иди изучай, что не так» — gennadiy на Mac пользователя получал
`LEASE_UNCERTAIN` на каждый `accord_describe`, включая самый первый.

## Разбор (факты)

- Экспорт сессии gennadiy `ses_e873b7605…`: пять вызовов `accord_describe`
  (18:11–18:17 МСК), все с ответом моста `LEASE_UNCERTAIN`; первый уже с
  hostname `Deniss-MacBook-Pro.local` вместо machine токена `mac-main`.
- Тот же бинарь `~/.config/accord/agent-board` (vcs 51c2b04) и тот же
  токен-файл через ручной JSON-RPC: неверный machine → чистый `VALIDATION`,
  следующий верный describe открывает сессию; повторный describe в живой
  сессии → `SESSION_ALREADY_OPEN`; 6,5 минут простоя без сессии → describe
  успешен. Карантин ни одним из этих путей не воспроизводится.
- В логе gennadiy в те же минуты: `AI_APICallError: Cannot connect to API:
  The socket connection was closed unexpectedly` (18:14, 18:17) и
  `server unavailable key=gennadiy-jira-confluence` при старте — на Mac был
  сетевой сбой. Первый upstream-вызов моста упал на транспорте; мост по
  контракту (`call.go`: transport failure → `uncertain()`) ушёл в карантин,
  хотя сессии и аренды ещё не было, и до перезапуска процесса gennadiy отвечал
  `LEASE_UNCERTAIN` на всё.
- Дополнительно: два живых экземпляра gennadiy (из `agent-board` с мостом на
  «Давинчи 2.0» и из клона `js-tasks`), агент не знает machine токена, у
  сервера нет «кто я».
- Побочный эффект проверки: пробный `next` захватил `T-2`; аренда снята
  `release` сразу же (state ready).

## Изменения

- `internal/store/sessions.go`: пустой `machine` в describe означает machine
  токена; явное другое значение по-прежнему `invalid_input`. Тест
  `describe_machine_test.go`.
- `internal/bridge/call.go`, `bridge.go`: до первой board-сессии отказ
  транспорта/нечитаемый ответ на `describe` не уводит в карантин: мост
  пересоздаёт upstream (`dial`/`reconnectLocked`), один раз повторяет тот же
  describe (тот же request_id, аренд нет) и при неудаче отвечает `UNAVAILABLE`
  с просьбой повторить. `UNAUTHORIZED`/`NO_SESSION`/`INTERNAL` до сессии
  возвращаются как есть. Карантин после открытой сессии не изменён.
  Инструкция моста: machine можно опустить. Тест
  `TestTransportFailureBeforeFirstSessionDoesNotQuarantine` (timeout, 503,
  затем успех в том же процессе без machine).
- `docs/api.md`: правило `machine`.

## Проверки

- `make fmt-check vet test build` — см. итог в коммите; целевые тесты bridge и
  store PASS.
- NOT-RUN: стенд не обновлён (серверная часть `machine` опциональна только
  после выкатки); клиентские бинари для обоих Mac нужно пересобрать и заменить
  (`~/.config/accord/agent-board`, у Ромы `~/.config/accord/agent-board`).
