# TASK-010 — постоянные профили агентов

20.10.2026. Ветка `codex/agent-profiles`, старт clean checkpoint
`867f905d301a508df61f018f6ecda1d17f81539f`. Прямое поручение продолжить
после TASK009 реализацию профилей, затем в новых чатах координацию и запуск.
Статус: COMPLETE. Реализация, независимые review, QA и production поставка приняты (WORK143: FINAL APPROVE). Контракт — [agent-profiles.md](agent-profiles.md).

## Последовательность и владельцы

1. Root — контракт, общая TASK, operations/update, интеграция и поставка.
2. WORK140 — независимая архитектура; до реализации disposition координатора.
3. Отдельные backend/frontend авторы с непересекающимися файлами и WORK.
4. Независимый reviewer — source, author tests и ops; обязательные замечания
   исправляются до source APPROVE.
5. Отдельный tester после APPROVE — новые process/E2E; reviewer проверяет тесты.
6. Synthetic Linux update/restore/recovery, полный production backup,
   безопасная поставка и HTTPS-проверка, итоговый clean checkpoint.

## Границы и приёмка

Project-local storage/CRUD/archive/versioned instructions+settings, явная
привязка сессии к immutable snapshot. Не identity, presence или execution.
Существующие проекты/данные/ACL/epochs/cookie/leases/SHA gates сохраняются.
Текущая поставка TASK009 и full backup проверяются до изменения VPS.
Tandem и соседний davinci-gui- не используются. Credentials/runtime только
ignored .local/VPS. Следующий чат создаётся после фактического закрытия этапа.

До выполнения проверки не считаются PASS. Physical second PC, provider workers,
координация и управляемый запуск — NOT-RUN/следующие этапы.

## Архитектурное решение

WORK140: APPROVE. Приняты strict role match и ID/version pair в describe,
immutable session snapshot, actor session ID без credential ID в DTO, quota128
профилей/1024ревизий, мягкий архив, MCP12tools+resources и schema3. Приняты
новый capability floor/отдельный updater/полный backup и строгая сохранность
исторических migration backups schema1/2-64hex.db с checksum. Исполнение,
произвольные commands/env/secrets/provider URL и координация отложены явно.

## Checkpoint source review

Backend141, Next142 и operations144 реализованы; независимый WORK143:
**source APPROVE** после исправлений offline migration, old schema restore/admin,
atomic temporary resume, malformed query и unknown helper collision.
Авторские cmd/internal suites+vet/race PASS, web60units/typecheck/build PASS;
независимо reviewer повторил cmd/internal, web60 и infra144 PASS.
Это не итоговая приёмка: полный make test пока FAIL на двух прежних schema2
expectations в process acceptance. Отдельный tester145 после source APPROVE
адаптирует их и проверяет actual process/Chrome/migration/restore/update.
Linux synthetic, production update/HTTPS/full backup — ещё NOT-RUN.

## Checkpoint независимой QA

WORK145:17/17 actual local process+Chrome PASS,16screens; independent143 test
и evidence APPROVE. Проверены actual5454→3, default+2projects, старые rows/
tokens/cookie/session/lease/blob, old/new full restore, ACL/epochs, CRUD/replay,
immutable snapshots/history, concurrent409/draft, committed lost response,
late A/two tabs, viewer controls и MCP12/resources. Product source не менялся.

WORK146 actual updater rehearsal выявил P1: snapshot DELETE-header validation
ошибочно применялась к законному закрытому WAL storage. Root добавил явную
live validation с nonempty journal rejection+full integrity/schema; snapshot
strictness сохранена. Independent143 fix source APPROVE, новые146infra PASS.
9599962 Go/Next были загружены только inactive и **не активируются**; будет
новый immutable релиз с исправленным оператором. Production по-прежнему5454,
базы и работающие guards не менялись. WORK146 local8/8 послеfix PASS;
systemd/Next/UID в fixture явно simulated, Go/SQLite/crash/resume реальные.
Финальные Linux exact release, actual systemd stage/update и HTTPS/fullbackup
впереди; COMPLETE до этих барьеров не объявляется.

Final source/author-test/tester-test review143: APPROVE, включая повтор updater
8/8 самим reviewer и exact43source/testhash. Root полный `make check` после
liveWALfix: PASS (146infra,60client, Go/all/race/vet/build/format/types).
Исходный код и тесты заморожены для нового immutable Linux release.

Следующий e923273 exact Linux12+8 PASS, но actualstage выявил rootops UIDошибку:
service-created registry marker ошибочно считался root-owned helper. Stage
остановился до journal/data/pointer изменений. Исправление сохраняет прежнюю
private regular0600 authority policy, не меняет владельцев/права; independent143
APPROVE, unit8/8 и actual updater8/8, полные infra148 PASS. E923 также остаётся
inactive; production5454 сохранён. Новый immutable+Linux+actualstage обязательны.

## Итоговая реализация и изменённые области

- `internal/core/profiles.go`, `internal/store/profiles.go`, migrations/schema
  validation/persistence/sessions: project-local profiles, immutable revisions,
  optimistic version, мягкий архив/restore и immutable snapshot сессии.
- `internal/projects`, `internal/transport`, `internal/bridge`, `internal/web`,
  `cmd/agent-board`: ACL/epoch gates, scoped REST/CLI/web CRUD, MCP12tools и новые
  profile resources; явная offline existing-only migration всех boards.
- `web/src/features/profiles`, routes/types/API/navigation/styles: список,
  создание/редактирование, история, архив/restore, profile snapshot в сессии,
  явный409 rebase и безопасный повтор committed actions.
- `infra/accord/projects/{upgrade,upgrade_storage,upgrade_verify}.py`, guard,
  snapshot/restore validation и `infra/accord/next/host.py`: отдельный capable
  updater/roll-forward, floor schema3/profiles, full backup, exact preservation,
  проверка effective systemd и actual owned process. Initial TASK009 apply не
  повторялся, старый Go/rollback БД не использовались.
- `tests/process/task010`, `tests/process/task010-update`, `tests/browser/task010`,
  author Go/web/infra tests и две прежние acceptance schema assertions; контракт,
  requirements/architecture/API/AGENTS, WORK140–146 и эта TASK.

Полный перечень файлов воспроизводится diff от стартового867f905 до app cbf2020
и итогового документационного checkpoint. Зависимости не обновлялись.

## Проверенная поставка и фактическая приёмка

Production Go/Next release: `cbf202086431335d4657c722a0f75328a09dc4ea`.
Go linux/amd64 clean VCS, SHA256
`66f949aa6022ade9cf4881c1a8cd65201b35d94c6635db942fcd6f73d42546aa`.
Next BuildID `hI1Cu6Lg4v0sMm34KPKnO`; manifest SHA256
`94186245b3d7ff02546f592343ef32ec44cf9b48a3e00f99ce07d6bec1651d06`,
1256regular members проверены. Рабочая панель:
Рабочая панель открывается по индивидуальной ссылке подключения; адрес не хранится в текущей документации.

| Проверка | Фактический результат |
| --- | --- |
| Root make check после WALfix | PASS: Go/all/race/vet/build, web60units/typecheck/build/format, infra146 |
| Последний marker-only fix | PASS: independent review, helper8/8, полные infra148/148 |
| Отдельная local process+Chrome QA145 | PASS17/17,16screens; approved source snapshot, provenance отдельно от clean release |
| Отдельная updater QA146 + independent repeat | PASS8/8; настоящие Go/SQLite/crash/resume, systemd/Next/UID simulated |
| Exact cbf Linux product/updater | PASS12/12 и8/8, SHA/process/source binding и cleanup ports |
| Actual hardened Next stage | PASS, owned systemd PID,0restarts,7894free; прежние pointers до apply сохранены |
| Production apply | PASS: full pre-snapshot, floor/guards, additive migration, старые rows/blobs сохранены |
| Production readonly HTTPS | PASS6/6,52screens1440/390/light/dark;805browser200, page/CSP/asset errors0, prefetch0, logout PASS |
| Finalize и final runtime | PASS: journal complete, actual Go PID/argv/ELF SHA, NextBuildID, helpers/effective launch/backup, service-UID guard |
| Финальный full backup | PASS: format2 validated на VPS, default board3/web2, registry и historical migration bytes включены |

В production сейчас только default board:0дополнительных проектов. Сохранность
подтверждена для3реальных DB (registry/default board/web); production проекты
для теста не создавались. Default+2projects/7DB проверены отдельно на synthetic
Linux updater fixture, не выдаются за production данные.

Private evidence в ignored `.local/task010/deploy/`: release-proof.json,
linux-product-proof-03.json, linux-update-proof.json, stage-runtime-proof.json,
apply/finalize-operator-output.txt, final-runtime-proof.json и
https-cbf-02/production-readonly-results.json. HTTPS report SHA256
`de89a71dbac0f4f4db77a40e61f7be650d9c73dcdf2dee9212efb1bb1e403017`;
approved script SHA04360016413b57b5e7ef7596da917f68812516d2de5c00202e0f4d30d7ee359f.

Final full backup на VPS:
`/var/backups/accord-web/projects-20261020T194706Z-397ba9372ebe4b998088dade1ccd329a`,
manifest SHA256
`a4e06648b561c5667071fc9ca5d64a201326951a1a9e69b845a13fe6dd56d013`.
Pre-snapshot сохранён и повторно validated; его manifest SHA256
`f045cef6da31aa86e5610860d6a3af72f11d1ce1afad00b30387d41291244e59`.
Реальные БД не скачивались и fixtures не попадали в production. Services/timers
active, Go/Next0restarts, recovery markers отсутствуют. Финальные checks не
сбрасывали tokens, memberships, sessions, leases или action history.

Первый HTTPS-cbf-01 **FAIL** остаётся отдельным report: тест ошибочно требовал
пустой browser array вопреки `omitempty`. Отдельный tester исправил только test,
independent review APPROVE, повтор cbf-02 полный PASS; product не менялся.
9599/e923 и прежние FAIL в checkpoints выше остаются историей, не финальной
production версией. Изначальные две acceptance expectations также исправлены
и вошли в последующий полный make check PASS.

## Ограничения и передача

Это завершение профилей и их безопасной поставки. Presence/identity/coordination,
provider connection/credentials и worker execution этим этапом не добавлены.
Роль/provider/model — сохраняемые настройки, без подтверждения поддержки
модели и запуска процесса. GitHub/OAuth не добавлен. Physical second PC и полный
V1 — NOT-RUN; Chrome390 является viewport. Backup пока только на прежнем VPS.
Remote push/merge/PR не выполнялись.

Следующий прямо порученный отдельный чат — «Координация»: сначала контракт и
архитектура поверх TASK009/010, затем реализация с отдельными авторами,
независимым review и QA. После его завершения — «Управляемый запуск». Профили
не подменяют session/credential; все schema3 floor, ACL/epochs, leases/fencing,
SHA gates, immutable snapshots, project scope и full backup обязательны.
