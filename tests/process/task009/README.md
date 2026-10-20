# TASK009 synthetic process и Chrome QA

Запускает только собственные процессы на свободных loopback3101 и3100. При
занятом порте отказывается до HTTP-запросов. Не останавливает systemd/nginx,
не читает VPS-конфигурацию, не подключается к Tandem и не использует реальные
данные. Все токены/БД/отчёты сохраняются в **новый** owned0700 каталог внутри
`.local/task009-qa/`; файлы токенов0600. Прежний verified binary нужен для
реальной миграции web1→2, cookie/lease/blob preservation и downgrade rejection.

## Локальный full run

Нужны проверенные новые Go binary и production Next build с upstream3101,
Node24, Playwright и Chrome. Переменные с путями задаются явно при отличии от
локальных defaults в harness. Root должен заморозить binary/dist на время QA.

```sh
ACCORD_TASK009_BINARY=/absolute/reviewed/agent-board \
ACCORD_TASK009_OLD_BINARY=/absolute/verified/legacy-agent-board \
ACCORD_TASK009_REVISION=ACTUAL_PRODUCT_COMMIT \
ACCORD_TASK009_SOURCE_DIRTY=ACTUAL_PRODUCT_SOURCE_STATUS \
ACCORD_TASK009_SOURCE_AGGREGATE=VERIFIED_SOURCE_HASH \
NODE24 tests/process/task009/run.cjs
```

Дополнительные overrides: `ACCORD_TASK009_QA_DIR` (новый private каталог),
`ACCORD_TASK009_PYTHON`, `ACCORD_TASK009_GO`, `ACCORD_TASK009_PLAYWRIGHT`,
`ACCORD_TASK009_CHROME`. Credentials не передаются через argv/env.

## Linux process-only rehearsal

Не требует Chrome, Playwright, Node frontend или Go toolchain: только Node24,
Python3/SQLite, два проверенных Linux Go executable и то же расположение source
helpers. Перенести в новый private root **код** `tests/process/task009/`,
`tests/browser/task009/` (нужен для test manifest), `infra/accord/projects/` и
`infra/accord/web/backup.py`. Реальные БД/credentials туда не копировать.
Относительный layout сохраняется; `repo` определяется по расположению harness.
Не использовать symlink/shared-writable ancestry или чужой существующий runtime.

```sh
ACCORD_TASK009_BROWSER=0 \
ACCORD_TASK009_BINARY=/absolute/reviewed/linux-agent-board \
ACCORD_TASK009_OLD_BINARY=/absolute/verified/legacy-linux-agent-board \
ACCORD_TASK009_GO_VERSION=go1.27.2 \
ACCORD_TASK009_REVISION=ACTUAL_PRODUCT_COMMIT \
ACCORD_TASK009_SOURCE_DIRTY=ACTUAL_PRODUCT_SOURCE_STATUS \
ACCORD_TASK009_SOURCE_AGGREGATE=VERIFIED_SOURCE_HASH \
NODE24 tests/process/task009/run.cjs
```

`GO_VERSION` без доступного Go tool записывается как operator build provenance,
не как фактический вывод `go version -m`. Next/Chrome проверки в таком запуске
NOT-RUN. SQLite snapshots/restores работают только с generated fixtures внутри
этого нового root. Script не заменяет active production state и не вызывает
поставочные `operate.py`/systemd. Наличие PASS Linux rehearsal не является
production deployment PASS или проверкой двух физических ПК.

Полный `results.json` фиксирует hashes обоих binary, test manifest до запуска,
Go/Node/Chrome/Next provenance по доступным данным, actual checks и cleanup.
Изменение test source во время run делает результат FAIL. Новые тесты и их
результаты требуют независимого review прежде, чем task может быть закрыт.

Для production только отдельный reviewed
`tests/browser/task009/production-readonly.cjs`: root предоставляет приватные
config/token paths через `ACCORD_QA_*`, tester не получает их содержимое.
