# WORK-056 — независимая process QA конфигурации D

- Исполнитель: qad056, tester, отдельный от авторов043/052 и reviewer054.
- Статус: на проверке; функциональный результат FAIL QA056-01. HEAD d5a3a12bcf5a87d854171288139788bff0e2ab9a; codex/config-qa.
- Входы: AGENTS, requirements, service/config contracts, configuration-runtime,
  WORK043/052, APPROVE code/security054 (MAIN docs/review-054-config-fixes.md).
- Запись: tests/acceptance, tests/infra, этот WORK, docs/acceptance-056-config-runtime.md,
  private .local/evidence/config-qa-056. Source/internal/cmd/infra не меняются.
- Tandem OFF; Davinci/main/TASK/commit/push/deploy не меняются.
- Профиль tester и общий процесс/parallel-work Davinci прочитаны. Именованные
  Claude модели недоступны; наследуемая модель как у автора — ограничение среды.
- Метод: ordinary child CLI/server/stdio + настоящие sockets и SQLite; race harness
  отдельно. PerformanceC FAIL не пересматривается; нагрузка не запускается.

## Журнал

| Время | Событие | Автор | Статус | Причина |
| --- | --- | --- | --- | --- |
| 2026-10-18T19:24:39+03:00 | started | qad056 | в работе | APPROVE054 подтверждён, source manifest сохранён; тесты после approve |

| 2026-10-18T19:40:31+03:00 | completed | qad056 | на проверке | Функциональный QA FAIL QA056-01, независимые остатки PASS; source unchanged; передача root/057 и test reviewer |

## Результат и blocker

Отчёт/TC/протокол: [acceptance056](../acceptance-056-config-runtime.md).
Source HEAD не менялся;155 tracked cmd/internal/infra/go.mod/go.sum hashes
равны до/после. Test delta:8 новых Go файлов config_* в tests/acceptance;
адаптация5 existing process fixtures/helpers;2 schema3 Python infra кейса.
Новый набор нуждается в независимом review; tester не выдаёт себе approve.

QA056-01 OPEN: concurrent real CLI apply иногда возвращает raw
`agent-board: database is locked (261)` вместо CONFIG_CONFLICT. Полный
make check FAIL; guard не ослаблен, source не исправлялся. Разрешённый
единственный targeted повтор PASS: winners1/version3/history3/audit_delta1;
это flaky evidence, не закрытие defect. Full FAIL-run остановился до poststate
assert; partial writes или два winner в нём не доказаны. Усиленный тест теперь
сохраняет/asserts poststate до exact-code отказа. Private cas-single-repro/
содержит оба raw результата, кандидаты, CLI, poststate и checked snapshot.

## Проверки и freeze

- Обычный targeted final process:33 top-level PASS (15D+18C), до fullcheck FAIL.
- Единственный полный make check: fmt/vet PASS,142 source tests/7packages PASS,
  acceptance32 PASS/1 FAIL (QA056-01); до infra/race/build Make остановился.
- Первая launch attempt с указанным bin/go1.27.2 отказала до vet, потому что
  executable отсутствует. Найденный bin/go — фактически Go1.27.2; путь исправлен,
  error log сохранён. Dependencies не устанавливались, Makefile не менялся.
- Независимый остаток: Python41 PASS (mocked systemd), build PASS;
  race174 top-level/8packages PASS с явным skip CAS process (общий FAIL остаётся).
- После diagnostics delta один разрешённый targeted CAS PASS, format/diff-check
  PASS; полного повторного make на delta нет из-за открытого source blocker.
- source manifest SHA d8bc768d9f9c4dee837106eacab78b1e1d5504b5df2efef1d117fc8c48da7782;
  fullcheck test freeze SHA8122cad55d67aabee8689a3cca9f7cb47da0026e9baea4c9886000a8d808ca00;
  final test freeze SHA344114d04e20f86819df4ce837fd1c99d6552e2fa67eeebde2621a0faacbd4f2.
- Private evidence .local/evidence/config-qa-056/:
  results.json, manifest-check.json, evidence-manifest.json, acceptance-final.jsonl,
  full-check*.json/log, independent-results.json/logs, cas-single-repro/.

## Границы и следующий исполнитель

Все C asserts и12tools сохранены. Schema1 exact bytes проверяются offline
migration+stale apply до runtime Sweep; затем serve/Sweep/restart и прежние
semantic/history asserts. Нормализация entity JSON при обычном Sweep не названа
сохранением любых runtime bytes. Schema2 fixture создан historical C binary;
старый binary schema3 не меняет. Backup/restore/reset schema3 сохраняет config,
notes/history, отзывает credentials и fences oldSID. SIGKILL после ack — process
crash drill, не power-loss/inflight migration доказательство.

PerformanceC FAIL остаётся у architect053. VPS/live D/providers/UI/OAuth,
реальные Claude/Codex, второй physical PC, Linux/Windows, load, power-loss и
arbitrary inflight commit kill — NOT-RUN. Source correction принадлежит WORK057;
после source approve обязателен CAS retest и review новой test delta.
Общая TASK/main/Davinci, commit/push/deploy не менялись.

