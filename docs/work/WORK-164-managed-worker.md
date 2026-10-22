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
