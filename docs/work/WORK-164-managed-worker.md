# WORK164 — локальный managed worker и native providers

22.10.2026. Исполнитель реализации. Область: `internal/worker/**`,
`cmd/agent-board/worker.go`. Общий TASK и backend изменяет координатор.

## Реализовано

- Локальные CLI `worker enroll|identity|consent|run --config PRIVATE`, отдельный
  per-run stdio bridge и скрытый Unix lifetime supervisor. Никаких SSH/inbound
  слушателей, shell interpolation, remote executable/path/env/argv.
- Явные локальные workspace/provider/model/controller allowlists и узкие presets:
  Codex `read-only|workspace-write`, Claude `default|acceptEdits`; пустое значение
  оставляет native policy. Нет approvals bypass, `--last` или implicit resume.
- Native Codex exec JSONL / exact UUID resume; Claude print stream-json / exact
  UUID resume. Profile snapshot передаётся в новый native describe; task reference
  не подменяет claim/lease. Codex invocation отключает обнаруженные обычные MCP
  connectors, Claude использует strict MCP config. Сохранённые native настройки
  и credentials не редактируются.
- Исходящий typed HTTP с project bearer + отдельной human session и host proof,
  origin/TLS/redirect guards. Enrollment/consent сохраняют payload и session scope
  до запроса для одинакового retry. Взаимный control требует explicit local grants.
- Private fsync journal до spawn, exclusive state lock, свежий process nonce,
  локальный monotonic deadline request-start+25s при серверной lease30s. Unknown
  launch не повторяется. Journal ограничен8MiB до записи; acknowledged receipt
  освобождает дублированный вывод, immutable dispatch identity остаётся.
- Checkout lock общий между enrollments: per-user config `Accord/worker-locks`,
  ключ Unix dev/inode или Windows volume/fileID. Durable launch intent остаётся
  после crash и запрещает новый запуск. Пути/identity pinned; symlink/reparse/UNC/
  известные network FS запрещены, cwd/executable перепроверяются перед spawn.
- Unix отдельный helper получает приватную stdin lifeline: worker SIGKILL/EOF
  запускает stop принадлежащей native process group независимо от worker.
  WNOWAIT удерживает leader PID до SIGKILL/проверки отсутствия живых group members,
  затем reap. Финальный proof связан случайным nonce. Windows создаёт процесс
  suspended, назначает Job Object до resume, проверяет ActiveProcesses=0 перед
  proven exit; job kill-on-close обеспечивает kernel lifetime guard.
- Stop prioritised локально после durable receipt, даже если предыдущий report
  ещё не подтверждён. Ошибка остановки → needs_recovery/delivery_unknown. Unix
  interrupt grace3s, затем force; Windows headless Job termination forced=true
  без обещания мягкой консольной остановки. Grace/exit unknown не называется stopped.
- Вывод очищается от известных локальных credentials/host proof, ANSI/control и
  URL credentials, ограничивается8KiB с явной отметкой усечения и передаётся
  валидным UTF-8 чанками≤4KiB. Native error/stdout code0 не считается success.

## Авторские проверки

PASS: focused Go unit/integration и race `./internal/worker ./cmd/agent-board`;
mock процессы не запускают native provider. Покрыты accepted ACK до spawn,
dispatch replay/identity, early stop, provider events/errors, exact session,
controller revoke, unconfigured profile, canonical identity/symlink, общий checkout
lock+unknown intent, private journal/restart/quota, secret redaction/UTF-8 chunks,
HTTP redirect/project binding, durable enrollment retry и nonce proof parser.

PASS: cross-compile worker tests для Windows amd64 и Linux amd64 (повторяется
после последнего source diff). Это не Windows runtime PASS.

NOT-RUN до source APPROVE: новые child-tree/crash/native CLI process/E2E тесты;
их пишет отдельный тестировщик после независимого approve. Физический Windows+
Claude/второй ПК остаются NOT-RUN. Production/VPS/GUI/Tandem не изменялись.

## Ограничения

### Конфигурационная подготовка не блокирует failsafe

Независимое ревью обнаружило, что discovery/verification native MCP выполнялись
под общим `Engine.mu` во время accepted ACK. Два медленных config-only вызова могли
задержать watchdog и StopAll уже работающего другого workspace после локального
monotonic deadline.

Подготовка теперь получает глубокий immutable Config и RunState snapshot и отпускает
общую блокировку на время внешних MCP проверок. Перед созданием native процесса
повторно проверяются fatal/closed, deadline, StopAll epoch, identity/generation/state
попытки, отсутствие active процесса, локальный controller/config allowlist и pins.
Accepted attempt identity сохраняется до проверки; Close/failsafe во время подготовки
не разрешает отложенный launch даже при последующем обновлении lease.

PASS: focused unit/integration/race worker+cmd. Barrier unit удерживает verification,
пока у другого активного workspace истекает deadline, и требует входа Stop до
освобождения verification; после освобождения новый native процесс не создаётся.
Отдельно проверено отсутствие разделяемых mutable allowlists в Config snapshot.
Новых process/E2E тестов автор не добавлял; actual generated-map native MCP config
probe и повтор offline диагностики выполняет QA после source APPROVE этой дельты.

### Native Codex transport overrides

QA установил причину отдельного startup FAIL на Codex `0.163.0-alpha.11.1`:
`mcp list --json` включает plugin-provided runtime stdio servers, у которых нет
transport table в пользовательском config. Прежний override только `enabled=false`
создавал новую неполную запись и Codex завершался с invalid transport до thread.

Adapter теперь читает strict metadata name/transport/enabled и формирует полные
invocation-only disabled записи: для stdio — command+args, для HTTP — отключённый
loopback placeholder URL. Исходные access-bearing URL/HTTP credentials не копируются
в argv. Неизвестный transport отклоняется. Совпадение зарезервированного имени
`accord_managed` с HTTP transport отклоняется до запуска вместо смешивания url и
command; переименование существующего подключения остаётся локальным решением.

Последующий QA exact-shape probe показал, что CLI path `mcp_servers."name"`
создаёт имя с буквальными кавычками: исходное подключение остаётся включённым.
Обязательная повторная инспекция корректно запрещала такой запуск. Исправлена
форма override: одна таблица `mcp_servers={"name"={...},...}`; имена интерпретирует
TOML parser внутри таблицы, а не CLI path parser. Transport-aware ограничения и
проверка единственного enabled managed bridge сохранены. Unit regression требует
целую таблицу и запрещает возвращение к quoted CLI path segments.

Перед native agent запуском обязательна повторная config-only инспекция именно с
полученными overrides: единственный enabled MCP — managed stdio с точным executable
и immutable attempt arguments, все остальные connections disabled. Успешный parse
или отсутствие prompt сами по себе не считаются доказательством isolation.
User config/auth/sandbox/approvals/rules не изменяются, skipGit/bypass не добавлены.

PASS: focused unit/integration/race worker+cmd, включая полные transport overrides,
HTTP collision/unknown transport fail-closed и exact post-override verification.
PASS: кросс-компиляция тестов Linux amd64/Windows amd64. Native configuration
probe для нового adapter и реальный provider smoke — NOT-RUN до source APPROVE;
их повторяет QA существующим probe. Диагностический offline runtime дефект остаётся
отдельным: его причина не выводится из этого transport startup исправления.

API: [MCP transport configuration](https://learn.chatgpt.com/docs/extend/mcp),
[configuration reference](https://learn.chatgpt.com/docs/config-file/config-reference).

### Диагностический checkpoint после runtime QA

После `0b05610` независимый QA воспроизвёл нестабильный `needs_recovery` в offline
сценарии: активность управляемой группы прекращалась до30s, но положительный exit
proof терялся. Причина на этом checkpoint ещё не установлена; гипотеза о Darwin
EPERM не считается доказательством и сигнальные ошибки не игнорируются.

Добавлены только allowlisted stage codes: waitid errno, census error/missing leader,
kill/interrupt errno, group exit timeout, helper exit/missing nonce proof/timeout.
Они проходят через nonce-bound helper proof и сохраняются в приватном поле
`RunState.local_diagnostic`. Public error остаётся общим; Receipt/API/UI не получают
коды, raw stderr, пути, PID, prompt, provider output или credentials. Неполный proof
по-прежнему оставляет `needs_recovery` и durable checkout intent.

Отдельная подтверждённая source ошибка исправлена: `StopAll` запускает Stop для
всех групп одновременно. Последовательные grace periods могли оставлять поздние
группы активными дольше общего deadline. Ошибки и uncertain state каждого процесса
сохраняются отдельно. Авторский barrier fake test требует входа всех Stop до
освобождения первого; новых process/E2E тестов автор не добавлял.

PASS: повтор focused unit/integration/race worker+cmd и кросс-компиляция тестов
Linux amd64/Windows amd64. Добавлены unit проверки закрытого набора/private-only
диагностики, переноса helper stage и concurrent stop. Реальный offline repro с
новой диагностикой остаётся NOT-RUN до независимого source APPROVE; после него
QA повторяет существующий сценарий. Intermittent runtime дефект ещё не назван fixed.

Unix supervision охватывает управляемый native CLI и принадлежащую ему process
group. Потомки, намеренно вышедшие из неё через `setsid`/double-fork с отделением,
могут продолжить работу вне этих границ. Проверка отсутствия живых членов PGID и
освобождение checkout lock не доказывают отсутствие всех возможных процессов,
пишущих в checkout. Worker не заявляет полную Unix sandbox/tree containment.
Windows Job Object охватывает своё назначенное поддерево сильнее process group;
фактическая Windows runtime проверка по-прежнему NOT-RUN. Независимый QA после
source APPROVE обязан проверить и явно зафиксировать сценарий detached descendant.

Crash с ambiguous journal или durable checkout intent требует локальной проверки
дерева; по сохранённому PID worker не убивает и автоматически новый процесс не
создаёт. Resume возможен только при сохранённом точном provider UUID и proven
exit. Clarification — следующий CLI turn; очередь/порядок обеспечивает сервер.
Native auth/permissions должны быть подготовлены владельцем каждого ПК; отказ
доступа виден как failure. Полный V1 и две физические машины не объявлены PASS.

Файлы разделены по ответственности, каждый менее300 строк. Использованы уже
имеющиеся Go stdlib/x/sys; зависимости не обновлялись.

Первичные API: [Codex non-interactive](https://learn.chatgpt.com/docs/non-interactive-mode),
[Claude headless](https://code.claude.com/docs/en/headless),
[Claude CLI](https://code.claude.com/docs/en/cli-reference),
[Claude permissions](https://code.claude.com/docs/en/permissions),
[Windows Job Objects](https://learn.microsoft.com/en-us/windows/win32/procthread/job-objects),
[Job accounting](https://learn.microsoft.com/en-us/windows/win32/api/winnt/ns-winnt-jobobject_basic_accounting_information).

### Локальное согласие на отдельные native MCP tools (default off)

Реальный native smoke QA дошёл до conversation, но `accord_managed.describe`
получил отказ native approval policy. Отказ не обходился. Подготовлена локальная
опция `native_mcp_approved_tools`: map только configured `codex`/`claude` в массив
уникальных точных имён 12 bridge tools (`comment`, `create`, `describe`, `events`,
`get`, `heartbeat`, `link`, `next`, `note`, `release`, `transition`, `update`).
Панель не может задать эту опцию. Отсутствующая map, null/пустой provider array
не создают permission overrides. Ни один текущий runtime config не получил grant.

При непустом Codex grant только таблица фиксированного `accord_managed` получает
`enabled_tools` ровно выбранных операций, default approval `prompt` и явные
per-tool policies для всех 12: выбранные `approve`, остальные `prompt`. Это
ограничивает каталог и не наследует более широкий grant из merged tool tables.
Global `approval_policy`, sandbox, authentication и пользовательские файлы
настроек не меняются. Для Claude добавляются точные MCP `--allowedTools` выбранных
операций и `--disallowedTools` остальных известных bridge операций; strict MCP
config сохраняется, builtin tools/native managed deny/ask rules не меняются.
Consent map глубоко копируется для подготовки immutable invocation snapshot.

Установленный Codex parser config-only подтвердил поддержку per-tool `approve`
на отключённом синтетическом connector; неизвестный enum отклонён. `mcp get`
показывает exact `enabled_tools`, но `mcp list` этого поля не раскрывает; ни один
из них не раскрывает effective tool approval policies. Поэтому обязательная
повторная list-инспекция по-прежнему доказывает exact bridge binding/isolation;
состав consent override проверяют unit и disabled parser evidence, без заявления
об effective runtime approval introspection. Пробы не запускали модель/MCP tools.
Без отдельного человеческого согласия native smoke после denial не повторялся.

PASS: worker+cmd unit/integration/race, включая default-off/strict enum/duplicate
validation, exact selected Codex catalog и policies, exact Claude allow/deny names,
изолированный snapshot и передачу локальных grants в provider invocation. Новых
process/E2E тестов автор не добавлял. Реальный native smoke с explicit grant,
Windows Claude runtime и второй физический ПК — NOT-RUN. Эти проверки остаются
за независимым QA после source review и требуемого локального согласия.

Источники: [Codex MCP approval settings](https://learn.chatgpt.com/docs/extend/mcp),
[Codex config reference](https://learn.chatgpt.com/docs/config-file/config-reference),
[Claude headless tool consent](https://code.claude.com/docs/en/headless),
[Claude CLI tool restrictions](https://code.claude.com/docs/en/cli-reference),
[Claude MCP permission names](https://code.claude.com/docs/en/permissions).
