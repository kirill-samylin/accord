# Process/browser проверки единственного Next.js интерфейса

В TASK019 старые SSR/browser/control/refinement suites удалены вместе с HTML UI.
Защиты авторизации/CSRF/ACL/аренд/retry продолжают проверяться в Go и Next suites.
`upload-process.cjs` сохраняет отдельные HTTP upload/timeout boundary проверки;
его protocol — JSON Accept, без разбора HTML. Shared harness и lifecycle не удалены.
Исторические результаты прежних SSR suites остаются в docs/work и TASK004–006.
Новые process/E2E проверки запускаются только после независимого source approve.

## Next.js клиент TASK-007

Новый набор `next-acceptance.cjs` запускается после независимого approve и
freeze Go binary, Next production build и исходников. Это проверка
единственного Next→Go пути; исторические SSR результаты не заменяют её.

```sh
export ACCORD_QA_NODE=/absolute/path/to/node24
export ACCORD_QA_BINARY=/absolute/path/to/bin/agent-board
export ACCORD_QA_DIR=/absolute/private/new-next-run
export ACCORD_QA_SOURCE_MANIFEST=/absolute/private/source.json
export ACCORD_QA_PLAYWRIGHT=/absolute/path/to/node_modules/playwright
export ACCORD_QA_CHROME=/absolute/path/to/Chrome
export ACCORD_QA_PYTHON=/absolute/path/to/python3
"$ACCORD_QA_NODE" tests/web/next-acceptance.cjs
```

Сначала соберите `make build web-build`. Default Next rewrite фиксирует Go
`127.0.0.1:3101` во время build: порт должен быть свободен, runner не завершает
посторонние процессы. Next получает свободный loopback порт, равный browser
Origin в Go. Параллельные запуски этого набора не поддерживаются. Во время
прогона нельзя пересобирать Next или заменять binary. Только synthetic данные,
новый приватный QA_DIR и cleanup собственных процессов.

`ACCORD_QA_NEXT_SCOPE=journeys` ограничивает проверку пользовательскими формами;
`no-deadlines` пропускает медленные upload границы. Только `full` включает весь
набор; scope записывается в JSON и subset PASS не считается полной приёмкой.
Реальные Go/Next/Chrome, native MCP, offline/SIGSTOP и браузерные таймеры
разделяются в доказательствах. Headed Chrome с отдельным private profile нужен
для проверки настоящего hidden/visible, а не имитации focus в headless.

JSON, скриншоты и логи остаются в `.local/`. Наличие PNG не даёт visual APPROVE:
нужен фактический осмотр. Фактическая версия, команды, результаты и ограничения —
[WORK-128](../../docs/work/WORK-128-next-qa.md), независимое ревью —
[WORK-127](../../docs/work/WORK-127-next-code-review.md).
