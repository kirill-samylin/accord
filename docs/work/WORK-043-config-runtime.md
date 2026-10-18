# WORK-043 — D: исполнимая конфигурация и workflow

- Исполнитель: configruntime043, роль разработчик; экземпляр WORK-043-20261018.
- Статус: на проверке; source freeze. Владелец общей TASK — координатор.
- Каталог: `.local/config-runtime-003`, ветка `codex/config-runtime`.
- База: `2dfd86bf1627775a8fdd87c650d9a3ada4cae17f`.
- Входы: requirements/architecture/api/service-contract/config-contract;
  parser APPROVE036; source C APPROVE041 (process QA C отдельно).
- Границы: runtime D, schema3, config/scopes CLI, read HTTP, unit/integration.
  Main, E, live/deploy, commit/push не входят. Tandem OFF по указанию пользователя.
- Профиль: `/Users/shagi/Desktop/Project/davinci-gui-/docs/agents/roles/developer.md`.
  Именованные Claude модели недоступны; используется наследуемая модель.

## Решения и проверяемые свойства

- Каждый runtime transaction загружает immutable active config после writer lock;
  Store.cfg остаётся только bootstrap input. Snapshot переносится в isolated
  mutation candidate, а карты конфигурации не меняются во время операции.
- Bootstrap сохраняет семь ролей, реальные TTL/project, старые права/reopen;
  legacy scopes — точный массив семи ролей. Davinci применяется только явно.
- Новый владелец bootstrap задаётся явно. `token create --owner` является явным
  вводом владельца только новой DB; существующая конфигурация не расширяется.
- Новые token defaults — configured agent roles. Human scope выдаётся только
  явной локальной командой и при allowlisted owner.
- Единственная новая зависимость — golang.org/x/text v0.42.0 для Unicode Fold.
  v0.43.0 проверена, но потребовала несвязанный x/sync bump; выбран совместимый
  v0.42.0, существующие зависимости сохранены. API проверен по официальной
  https://pkg.go.dev/golang.org/x/text/cases (Fold — Unicode casefold).
- Legacy typed Evidence fixtures перенесены на raw-map constructor без изменения
  смысловых assert. Прямые изменения cfg.Notes заменяются реальным ApplyConfig.
- Evidence MCP schema переопределяется generic object; дополнительная schema
  валидация/точные числа принадлежат store/config.

## Проверки

Выполняются developer unit/integration; текущий промежуточный прогон не является
независимым approve либо process/E2E. Новые process/E2E — tester после approve.

## Журнал

| Время | Событие | Автор | Статус | Причина / версия |
| --- | --- | --- | --- | --- |
| 2026-10-18T18:40:10+03:00 | started | configruntime043 | в работе | Назначен полный runtime D на базе 2dfd86b; границы подтверждены root |

## Итог реализации

- schema3/private prebackup/atomic migration, compatibility active config и exact
  seven-role legacy scopes; исходные config version/DB schema/YAML version разделены.
- Config CLI init/validate/get/history/apply, CAS/audited no-op, CONFIG_IN_USE без
  освобождения leases; read-only authenticated config/history HTTP без raw_yaml.
- Runtime active snapshot после SQLite writer lock; scopes CAS/audit; dynamic
  roles/grants/tools/reopen; custom роль проходит настоящий SDK→bridge→done.
- Evidence raw map + typed getters, точные integers/duplicate/type guards,
  canonical verdict/result с legacy outcome; hard floors/session provenance,
  global latest verifier obligations security/design/custom и historical kind.
- Human members/category/complexity claim+done; отдельная singleton role lease
  и generation fencing, в том числе role1s/session1h/task1h; bridge role deadline.
- Draft/priority/areas metadata, epic owner warn/off/block/handoff CAS, aliases/
  single-editor/overlap, Unicode Dice/duplicate override/duplicate_of, общий DAG.
- Notes limits/rules/TTL hints из snapshot; historical stats не переписаны.
- Backup/recovery/restore schema3; минимальный infra delta — два supported tuples.
- Обновлённая эксплуатационная инструкция: docs/configuration-runtime.md.
  Все изменённые Go-файлы укладываются в300 строк, исключений нет.

## Фактические проверки на frozen source

- PASS: `go test ./cmd/... ./internal/...` — все7 пакетов, включая developer
  unit/integration D и существующую регрессию C/native. Лог:
  `.local/work043/unit-integration-final.log`.
- PASS: targeted race store/bridge/core: CAS, audit rollback, live config, quota
  concurrency, competing claims, native custom role/evidence, role TTL fencing,
  typed getters. `.local/work043/race-targeted-final.log`.
- PASS: `go vet ./cmd/... ./internal/...`; `go build` в private local output;
  `git diff --check`. Логи vet-final.log/build.log в том же каталоге.
- Реальная версия локального инструмента: go1.27.2 darwin/arm64; go.mod1.26.1.
- Infra synthetic unit:23 PASS/1 FAIL. Существующий negative fixture
  tests/infra/test_recovery.py:128 использует уже поддерживаемую schema2 как
  unknown. Root подтвердил, что main C QA APPROVE045 уже заменил её на99 и добавил
  accepted2; эта approved дельта будет объединена координатором. Старый process/
  infra fixture здесь намеренно не правился. Лог infra-unit.log сохраняет FAIL.

## Ограничения и передача

- Независимые code/security review — NOT-RUN автором; нужны перед apply/live.
- Полный объединённый `go test ./...`/новые D process/E2E — NOT-RUN. По
  отдельному разрешению root механически адаптирован существующий
  tests/acceptance/workflow_test.go:86,92: EvidenceOf/map и typed getters; исходные
  assert сохранены. PASS: `go test ./... -run '^$'` — все8 пакетов, в том числе
  реальная сборка acceptance binary в TestMain; runtime cases не запускались.
  Лог all-packages-compile.log. C QA main1e9025c объединит root; D tester после
  approve обновит реальные schema3/startup expectations и новые сценарии.
- C process QA не объявляется закрытым этими developer tests.
- Actual two-process CLI apply/reload, процессные crash/recovery/old binary,
  production VPS, OAuth/webhooks/UI и второй физическийПК — NOT-RUN в этом WORK.
  Здесь cross-connection SQLite integration, настоящий in-process SDK/bridge.
- Main/E не изменялись. Commit, push, merge и live/deploy не выполнялись.
- Source manifest и binary SHA: `.local/work043/source-manifest.json`.
  Перед ревью сверять SHA файлов из manifest, включая новые untracked files.

## Журнал завершения

| Время | Событие | Автор | Статус | Причина / версия |
| --- | --- | --- | --- | --- |
| 2026-10-18T18:57:00+03:00 | completed | configruntime043 | на проверке | Runtime D реализован; local developer checks PASS, известный stale infra fixture FAIL; source freeze для независимого review |

## Ожидаемые runtime-расхождения старых process fixtures

Это отдельно от успешной компиляции. newClient сначала создаёт DB через token
create (compatibility TTL15m/15m), а b.start затем явно задаёт task TTL30s/1s и
sessionTTL1m: новый runtime правильно вернёт CONFIG_CONFLICT. Старт пустой DB до
токена без bootstrap-owner вернёт CONFIG_BOOTSTRAP_REQUIRED. Backup fixture
ожидает schema1 (main C QA меняет на2), combined D должен ожидать3. Это требует
адаптации операционных входов fixtures, а не ослабления runtime startup checks.

| 2026-10-18T18:58:17+03:00 | handed_off | configruntime043 | на проверке | Root разрешил механическую Evidence fixture compatibility; all-packages compile PASS; окончательный source freeze, commit/merge принадлежит root |
