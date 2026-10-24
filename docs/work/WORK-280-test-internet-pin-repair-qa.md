# WORK280 — QA восстановления запуска Test_internet

24.10.2026. Отдельный QA, после предварительного APPROVE WORK279.
Область записи: эта карточка и private `.local/task016-pin-repair/qa/`.
Runtime config/source не меняются; private snapshots не публикуются.

## План проверки до запуска

Проверить server host online, acquired epoch5, отсутствие активных managed runs
и workflow reservation; сохранить server-after и HEAD/status Test_internet.
Один новый обычный managed_start с developer profile id/version исходного
failed run, без task_key и workflow. Prompt явно read-only: прочитать README и
AGENTS, подтвердить запуск `PIN_REPAIR_LAUNCH_OK`; не брать задач, не писать файлы,
не коммитить, не делать посторонних сетевых обращений. Это fresh start,
не resume отсутствующего native conversation. При отказе автоматически не повторять.

Через managed/run сохранить последовательность server snapshots: real running
с native UUID и затем terminal exit. После проверить pending0, отсутствие живых
owned descendants и неизменность HEAD/status (с сохранением прежних untracked).
Новые process/E2E test suites не создаются. PASS означает только этот native smoke.

## Результат

Проверка завершена; фактический результат и пределы приведены ниже.

## Фактическая проверка — PASS

Source baseline `d2af9c882683da7067056351f72218b03b141659`;
установленный Test_internet worker SHA256
`f6291d57963620554da09f50f8908ec4b0fe735cc6e04d23aba160016780dadb`.
Проверка использует существующий production server после ремонта WORK278;
сама QA binary/config/server storage не меняла.

- Preflight: host active, свежий heartbeat/живая lease, acquired epoch5,
  managed active0. Все 3 workflow completed, страница полная (next_cursor нет).
- Новый обычный run `mr_f36f593ffcbe7f143bf1706c157a315b`, generation1,
  profile `ap_2721ef3298054ae03963c0eeb0cc0a4f` version1. Task/workflow отсутствуют.
  Ровно один start; повторных execution commands не было.
- 14:07:56 UTC queued → 14:07:58 starting/accepted → 14:08:03 observed running,
  start applied. Точный native UUID
  `01a16e3d-629b-7261-b6e0-3c720baf71ff` сохранён до completed.
- 14:08:21 UTC observed completed, output `PIN_REPAIR_LAUNCH_OK`.
  Native transcript подтверждает только два read-only чтения AGENTS.md и README.md,
  оба exit0. Это реальный Codex CLI, не SDK/stub.
- Worker terminal completed подтверждает успешный provider exit по existing
  supervised lifecycle; отдельное числовое поле exit_code публичный managed/run
  не возвращает. OS audit во время running наблюдал собственное дерево worker;
  после completed живых descendants0, живых прежних owned process groups0,
  ни одного наблюдённого owned PID не осталось. Root worker продолжил работу.
- Local journal completed, pending0; server active runs0. Исходный пользовательский
  run `mr_69cf2ea3e4eec251d2b368cc53e047d0` остался failed без native UUID.
- Test_internet HEAD до/после
  `bf77b6db0fadfc5a18da3045f0f4fd5bf510a314`, status clean до/после, exact equality.

Private evidence: `.local/task016-pin-repair/qa/`: server-after/prestart/final,
start-request/result, run snapshots и observations, local-state-after/final,
receipt-final, process-running/final и process-audit, checkout-before/after,
native-transcript и native-read-evidence. Все файлы private mode0600,
каталог mode0700. Bearer/host secret/HTTP error bodies не выводились.

При подготовке read-only workflow-list первый запрос с limit100 получил HTTP
validation refusal; после чтения контракта использован допустимый limit50.
Это не start и не execution retry. Вспомогательная обработка null pending
исправлена на корректное pending0 без изменения runtime.

Предел результата: устранённый барьер локальных pins и fresh native launch — PASS.
Повтор исходного пользовательского задания, same-UUID resume этого failed run,
полный automation workflow, Windows/Claude execution и второй физический ПК — NOT-RUN.
Новых process/E2E suites не создавалось. Изменения QA: только WORK280 и private evidence.
