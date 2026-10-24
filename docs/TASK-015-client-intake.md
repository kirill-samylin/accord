# TASK015 — требования из настоящих Codex и Claude

Поручение пользователя 24.10.2026: проверить отправку требования из обоих
клиентов, полный цикл исполнения и управление через Accord. Исходный рабочий
контур — [TASK014](TASK-014-automation.md), контракт — [automation](automation.md).

## Объём проверки

- Настоящий Codex CLI, затем настоящий Claude Code на этом Mac. В каждом
  модель сама вызывает MCP submit и получает ID нового workflow.
- Для Codex: добавить `subtract` и три unit-теста в Test_internet; для Claude:
  после завершения первого цикла добавить `multiply` и ещё три unit-теста.
- Одна задача и один локальный коммит на цикл; существующие add/два теста,
  runner/AGENTS/README/.gitignore сохраняются. Реальные project-tests должны
  выполнить соответственно пять и восемь тестов на точных SHA.
- Исполнение ролей — существующая Codex-команда на Mac. Отправка из Claude
  сама по себе не доказывает исполнение ролей Claude или второй физический ПК.
- На одном живом аналитике проверить паузу, подтверждённую остановку,
  уточнение и продолжение из Accord; не продлевать скрыто deadline/бюджеты.

Текущие policy, local consent, ACL, fixed checks и права сохраняются.
Нет push/merge/deploy примера, зависимости не устанавливаются. Координаторы
не редактируют код. Авторизация Claude — обычный вход владельца в браузере;
коды/токены не передаются в чат или Git. Глобальный approval/sandbox не меняется.
Отказ native approval фиксируется как отказ; обход не является проверкой.

## Владельцы и порядок

Автор подключения — [WORK272](work/WORK-272-client-intake-author.md), независимый
review — WORK273, отдельная process/E2E QA — WORK274. Root ведёт этот checkpoint,
производственные UI/API действия и установку после source approve.
Реальные прогоны — только после независимого approve кандидата и QA-плана.
Закрытые runtime/evidence находятся в игнорируемой `.local/task015/`.

## Фактический результат24.10.2026

**Codex case PASS; TASK015 PARTIAL до настоящего Claude intake.** Установлен
reviewed WORK272 launcher: worker/native executable SHA проверяются отдельно,
Codex сохраняет read-only sandbox и обычные разовые подтверждения инструментов.
Вызовы describe и workflow_submit разрешены по одному, без session/always allow.
Реальная модель Codex CLI0.163.0-alpha.11.1 создала новый workflow через MCP:
`wf_9dd2b585d4e352f165cb21c2ee8fe50d`. Direct stdio имитация не использовалась.
Native intake UUID `01a16d01-4fef-7253-9c0c-e213b9bc8f3e`, настоящий exit0
после штатного CtrlD. Transcript137.394с/76309B, в согласованных пределах180+30с.

Workflow completed/version32: аналитик, developer, отдельные reviewer/tester.
Пять native invocations включают продолжение аналитика; четыре текущих run
завершены, все пять local exit/ACK/lock flags true, pending0. Review approve,
QA passed и две настоящие проверки project-tests/JUnit по5passed/0errors/
0failures/0skipped относятся к commit
`2a2a530c50c51290eb7653bcfadb979956f772ff`, parent `fc5cc899…`.
Изменены только src/math_ops.py и новый tests/test_subtract.py. Рабочее дерево
чистое; пять защищённых файлов byte-exact, AST add сохранён. Push примера не делался.

UI pause → подтверждённый stopped → clarify → resume проверены в этом же цикле.
Аналитик продолжил тот же UUID `01a16d02-6903-7150-b6cb-f658cbc6b758`, получил
сохранённое уточнение; новый board session/task generation2, continuation1,
delivered_messages1. Собственный native JSONL независимо подтверждает два хода.
Прямой local snapshot всех5ACK первого generation ДО resume не архивирован;
он не объявляется полученным. Есть stopped receipt/API/run_exit attestation и
корреляция guarded resume; финальные5ACK текущего generation проверены напрямую.
Исторический start delivery_unknown/last_error после fencing сохраняется;
stop applied не переименовывает его. Наблюдение literal OS resume argv NOT-RUN.

Первый запуск interactive TUI завершился startup FAIL/exit1: TERM=dumb и
heredoc EOF; модель/submit не запускались. После нового source review V2 задаёт
только TERM=xterm-256color и наследует настоящий PTY. Fresh preflight подтвердил
отсутствие нового workflow перед тем же request_id; прежний FAIL сохранён.
Interactive Codex сам записал только два служебных TUI-поля global config.
После закрытия собственных Test процессов approved one-shot helper восстановил
точный trusted baselinec17a7952, UID/GID/0600 с backup/fsync/readback/exit0.
Это изменение и восстановление, а не bytes-unchanged за весь запуск. Права,
sandbox и global MCP semantics не расширялись; informational provenance не заявлена.

Фактическая независимая QA — WORK274, receipt
`710ded65734dc07b3173975205ddc0a3bbf7b5f6c344d9ec59222c7e8241443a`;
финальный пакет `.local/task015/root/codex-final-1792831207436901000/`.
Дополнительный просроченный observer read409 не изменял workflow; свежая обычная
human session дала полный report без cursor и непрерывную историю.

Claude Code2.1.308 найден в bundled desktop installation; auth status без входа.
Provider gate остаётся false. Подготовлены exact executable pin и MCP-only
`--restricted --tools '' --strict-mcp-config`; это source readiness, не запуск.
Root login был закрыт; управление системным Terminal отклонено средой. Владельцу
передана команда обычного auth login для своего терминала, без пересылки кодов.
До его входа настоящая модель Claude, второй workflow multiply/8tests NOT-RUN.
Windows/второй физический ПК и Claude execution-role также NOT-RUN.

Сервер/Next остаётся4ae5292, Mac worker dfcef4b; новой поставки/миграции в TASK015
не было. Policyv1/срок25.10.2026 04:05МСК/лимиты300с-этап/1800с-цикл/12запусков
сохранены. Управление из текущего Codex desktop-чата автоматически не включено:
проверен отдельный native CLI из установленного launcher.
