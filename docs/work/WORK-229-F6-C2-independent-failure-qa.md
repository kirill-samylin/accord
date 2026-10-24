# WORK229 — независимая классификация FAIL F6V8 и C2V10

24.10.2026. Ops validation QA, только чтение закрытых артефактов. Новых native/
worker/моделей, CLI приложения/Git, HTTP, браузеров, helpers, настроек, повторов
или recovery не запускалось. R14 WIP не продолжался. Оба исходных **FAIL**
сохранены; `gate_pass=false`.

Private0600 отчёт `.local/task014/evidence229/review.private.json`:
`1a593bdf104191e5f232ab65dc24ac17e3a7e8a466474015e0311bbef5947df5`.
Индекс 74 ограниченных чтений `read-index.private.json`:
`941fc02cff69ba3a7f2c4ca09a5be443e73514b6e9fe6d6674819c3582ca0c97`.
Отчёт содержит только разрешённые проекции, пути и хеши; значений секретов нет.

## F6V8

Исходный runtime `pipeline210-f6-1792795464509963000`, results SHA
`d09251e678a4f0b82ec3024c750beecf2b93f01a2ad653f848d2ff30019f2144`.
Фактический analyst завершён. Developer создал настоящий commit
`d22a18fd8298c802a441259fea4eb8d832508226` с одним parent; loose Git object
проверен без Git CLI. Настоящий fixed check: exit1, passed1/failed1/skipped0,
архив JUnit SHA `0897e5108f19cceb7efb773952130f953bd41aa21da497180b1bf56378b09c6c`
совпадает с check result и HEAD. Это ожидаемая отрицательная попытка, не готовое
исправление и не успешная приёмка.

Exact own developer UUID `01a16af0-f396-7422-a53d-de99283be582` и cwd подтверждены
метаданными его единственного разрешённого native журнала SHA
`b564f85a17cb4580bf330ca1952ac7e9b7617471e41e0f9e3b9c4a37d39f1063`.
Архивный code-mode packet SHA `3148e1055b73124efc6a49a60f63747bd7c819a69c279bee914dd74902b86151`
сопоставлен с actual call/output по call_id и исходному коду. Код не исполнялся.
В `transition` отправлено 1161 ASCII символов `run_summary`; store разрешает 1024
(`internal/store/evidence.go:35`). Actual response: `VALIDATION`,
`evidence exceeds limits`. Начальный selector по function_call вернул 0 и не
является доказательством отсутствия вызова; корректный custom_tool_call найден.

**Терминальная очистка данных НЕ подтверждена.** В закрытой project DB workflow
`stopping`, version14/epoch3/reserved_runs4; developer run `stopping`, stop command
`queued`, T-3 `in_progress` с lease1, reservation row1 и human question T-4 ready.
В журнале analyst все пять флагов выхода/ACK/release=true; developer
native_exited/exit_durable=true, но managed_ack/lock_released/exit_ack=false,
pending1. Числовые native OS exit codes отдельно не архивированы.

При этом принадлежащие harness процессы действительно завершены: 24 generic
supervisor proofs имеют exit0/success=true/notforced, Next — известный ожидаемый
stop130/success=false, UI/MCP — exit0/signalnull, streams/requests/join закрыты.
Исходное `cleanup.complete=true/manualfalse` описывает этот внешний контур и
не доказывает native ACK0pending/lease0. Source `f6-v8/cleanup.cjs:29–39` посылает
cancel и останавливает fixture worker; отдельное ожидание terminal применяется
только к routeDivergence. Нельзя приравнять принятие cancel к применённой остановке.
Никакого восстановления архивного fixture не выполнялось.

WORK219 проверен независимо по архивным before/after bytes и TOML: только 126-byte
добавление trusted для собственного checkout; остальные parsed данные и metadata
requirements/managed_config совпадают. Классификация `QA_TRUST_ADDITION_ALLOWED`.
Текущая глобальная конфигурация не читалась.

## C2V10

Runtime `pipeline210-c2-1792795936950887000`, results SHA
`03b4915232ec3cda89ef7b837f5dc9ef4c05a3af8d77a0ec9c462c777ea6f7bf`.
Классификация root manual receipt независимо подтверждена: identity supervisor
proof proven=true, actual exit1, success=false, forced=false; stderr причины
не архивирован. Это известное завершение неуспешной операции, а не strict success0.

Собственный fixture `worker-base.json` содержит consent `workflow_get`, который
отсутствует в `managedMCPTools`. `LoadConfig→Validate→validateNativeMCPConsent`
отказывает до `worker.Open` и `Engine.Identity`. Source diagnosis привязан к
`native_consent.go` SHA `4579ce416fa00e5772fa526019a3816e5dd9faf0c20b169740f4ddfb7d2424e6`;
это вывод из source/config, не выдуманное stderr. Для четырёх настроенных ролей
также нужны семь точных поддерживаемых tools; отдельные исправленные adapters
не означают фактическую проверку C2V10.

В default/project DB managed_hosts/runs/commands/workflows/tasks/reservations0;
worker state, enroll и fixture-worker log отсутствуют: **0 native**, enrollment/
worker run не достигнуты. Остальные 20 generic proofs — strict exit0; Next stop130,
UI/MCP exit0/signalnull/fulfilled/drained подтверждены. Все joins, кроме identity
strict-success predicate, подтверждены. Исторические cleanup incomplete/manualtrue
и identity operation FAIL сохраняются. Global config comparison — **NOT-RUN**;
обхода общего guard ради проверки конфигурации не было.

## Границы

Все шесть прочитанных DB открывались только после отсутствия любых companions,
через `mode=ro&immutable=1` и `query_only`; SHA/identity/companions проверены до/
после. Данные не изменились. Полные 625 MiB tool/source повторно не перечитывались:
recorded approved source-after guards сохранены отдельно от новых фактов.
Только один Mac, Codex и synthetic fixtures; Windows/Claude/второй физический ПК
не проверены. Six-native fix/review/QA F6 и C2 controls остаются незавершёнными.
