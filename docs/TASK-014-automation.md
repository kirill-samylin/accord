# TASK-014 — выполнение требования командой агентов

23.10.2026. Координатор: root. Исходная опубликованная версия — `26d833f`;
рабочая ветка — `codex/native-preflight`. Это продолжение прямого поручения
пользователя: ввод требования из Accord либо Codex/Claude, автоматическое
создание и выполнение задач аналитиком, разработчиком, независимым ревьюером
и тестировщиком, управление ходом работы обоими участниками.

## Результат и границы

Одна отправка требования запускает наблюдаемый durable workflow:
анализ → план задач → реализация → review точного результата → QA →
исправление с повторными проверками либо итоговый отчёт. Человек может
приостановить, отменить, продолжить, уточнить требование и ответить на вопрос.
Вход из приложения и native MCP использует одинаковые правила и сущности.
Отдельные ручные managed runs остаются доступными.

Первый режим исполнения — последовательно в разрешённом checkout. Это явный
default, предотвращающий конкурирующих писателей; параллельные изменения
требуют отдельных workspaces и отдельного контракта интеграции результатов.
Workflow не получает права на ОС из project editor/owner: используются
локальное enrollment, controller allowlist и отдельное ограниченное разрешение
автоматизации. Подключение третьего участника к сайту не подключает его ПК.
Секреты native provider остаются на компьютере владельца.

Контракт — [automation.md](automation.md), после независимого review.
Исторические ограничения TASK011 о ручном цикле расширены этим поручением;
lease, access epoch, idempotency, независимость проверок, SHA/generation gates
и capable all-project обновления сохраняются. История и данные проектов не
подменяются демонстрационной базой. Изменения соседнего GUI не входят в этап.

## Состояние

**IN PROGRESS / NOT READY.** Контракт181 frozen и получил APPROVE183; автоматический
координатор в установленной версии отсутствует. В последнем проверенном snapshot Test_internet
вход и membership PASS, hosts/runs/profiles — 0; новый worker ещё не enrolled. На Mac GUI enrolled
worker; первый реальный background native запуск завершился до conversation
на isolated MCP preparation. Это был открытый prerequisite; позднейший195 real native GUI PASS5 и169 EVIDENCE APPROVE закрыли Mac gate. WORK196 finalized273: journalcomplete и all-project backup4проектов/9runtimeDB подтверждены. Новая автоматизация ещё не установлена.

| Работа | Владелец | Состояние |
| --- | --- | --- |
| Диагностика native preparation, авторские unit | managed_worker / WORK176 | SOURCE APPROVE; установлена62e в WORK182 |
| Реальный config-only Go process probe | managed_qa / WORK177 | 12 фаз PASS; independent evidence APPROVE |
| Установка diagnostic worker Mac | root / WORK182 | PASS; native retry FAIL: discovery timeout |
| Полный automation контракт | managed_worker / WORK181 | frozen, architecture/security APPROVE183 |
| Независимое architecture/security review | managed_review / WORK183 | APPROVE; не source/runtime приёмка |
| Core/store реализация и авторские tests | managed_worker / WORK185 | SOURCE APPROVE65; bounded PolicyGet/32checks/CheckAborted/terminal epoch/revoke bound |
| Registry/transport/bridge интеграция | root / WORK186 | SOURCE APPROVE31; actual Store→HTTP terminal ACK/lock/resume regression PASS |
| Browser/Next UI и author tests | automation_web / WORK187 | SOURCE APPROVE45; selected-scope/CheckAborted UI;103frontend/tsc/Nextbuild PASS |
| Capable schema6 upgrade/restore | automation_infra / WORK191 | SOURCE APPROVE191;54 infra units и targeted race PASS; Linux process QA впереди |
| Local worker/checks/Git proof/IPC | automation_worker_impl / WORK193 | SOURCE APPROVE97;4P1+regressions, worker/IPC/cmd race и crossbuild PASS |
| Native foundation retry | managed_qa / WORK195 | realMacPASS5 +169EVIDENCE APPROVE; root196 finalize273 complete |
| Существующие schema6 acceptance | managed_qa / WORK197 | SOURCE APPROVE189; оба actual process tests PASS8.649s |
| Новая source/QA/production приёмка | отдельные owners после кода | composite SOURCE APPROVE189; final checks и новые runtime QA впереди |

Root единолично редактирует эту TASK. Исполнители ведут собственные WORK и
получают непересекающиеся области записи. Process/E2E пишет отдельный QA
после независимого approve соответствующего кода и самих новых проб.

## Приёмка

1. Рабочий native start/stop/exact conversation resume/clarify на Mac;
   interrupted/offline/ownership uncertainty не выдаются за остановку.
2. Требование из Accord и scoped MCP создаёт один workflow при replay;
   подмена проекта/policy/host/profile, устаревшие и отозванные grants запрещены.
3. Реальный аналитик формирует валидированный план; задачи получают leases.
   CLI exit 0 без предметного результата не переводит этап в успех.
4. Реальная разработка проверяется отдельными review и QA с актуальными
   commit/generation и доказательствами выполненных проверок.
5. Отрицательная проверка вызывает ограниченный цикл исправлений; вопросы,
   исчерпание лимитов и неопределённость требуют видимого вмешательства.
6. Pause/cancel/resume/clarify сохраняют историю, переживают restart и
   потерянные ответы, не создают конкурирующих writers/дубли процессов.
7. UI показывает требование, этап, задачи, исполнителей, вопросы, действия
   управления и итоговые доказательства; queued не отображается как applied.
8. Миграции, старые binary guards, ACL/leases и all-project snapshot/restore
   проверены отдельно перед capable production roll-forward.
9. Native Windows/Claude, Codex и второй/третий физический ПК отмечаются
   отдельно. Cross-build, SDK и два запуска на Mac их не заменяют.

Ни один source commit, UI макет или synthetic fixture не является полным PASS
этого этапа. Версии, фактические проверки, ограничения и оставшиеся действия
фиксируются здесь по мере получения доказательств.

## Composite source checkpoint

WORK189: independent SOURCE APPROVE получен для18565/18631/18745/19120/19397;
все manifests независимо проверены без drift. Это не production/native PASS.
WORK197 два existing schema6 acceptance после отдельного review прошли actual
process tests8.649s. Root frontend103/tsc/Nextbuild и infra54 PASS; полный Go race
повторяется после последних frozen corrections. Новый native preflight WORK198
и Linux fixture WORK199 готовят отдельные testers; до source review и root GO
они не исполняются. Production по-прежнему273/schema5, Test Mac enrolment
и автоматическая policy ещё не созданы.

Final frozen composite checks: `go test -race ./...` PASS, включая весь
acceptance process package31.564s, store100.737s и worker12.526s. Frontend103
unit/TypeScript/Next production build PASS; infra54 PASS. Эти результаты
предшествуют отдельным actual native pipeline и Linux restore QA; production
273 не менялся. Подготовленные source/docs проверены на known local tokens,
provider-key/private-key patterns: findings0, private runtime не staged.

## Release/native checkpoint после source approval

Проверенный candidate `3e6ac81ee81909e3c33a2c86e593f4781357878d` собран из
clean checkout для Linux/macOS/Windows и Next. Go/Next загружены в неактивные
release directories с проверкой hashes; production273/schema5 не переключён.
Подробности — [WORK200](work/WORK-200-automation-release.md).

WORK198v2 actual Codex: exact file write и positive owned exit PASS, single
local commit FAIL. В событиях нет `git add/commit`; сообщение модели об отказе
ОС не является наблюдённым OS-denial. Отдельно зарегистрирован hash drift
user config, происхождение не установлено; глобальный конфиг не восстанавливался.
Узкий default-off Git permission contract [WORK201](work/WORK-201-native-git-permissions.md)
проходит отдельное review до реализации/нового native запуска.

WORK199 первый actual Linux fixture run остановился на public API seeding до
миграции. Old binary capability rejection PASS; оба собственных server process
завершились с положительным exit proof, source/binary pins не изменились.
Migration/snapshot/restore NOT-RUN. Отдельный V2 исправляет project-open flow
через штатные CSRF/action; требуется review и новый bounded root GO.

Позднейший WORK199V2 после SOURCE APPROVE202 и отдельного GO: actual old seed
четырёх boards/9DB, format2 schema5 snapshot и candidate schema5→6 с сохранением
всех прежних logical rows/четырёх schema5 backups PASS. Проверка отказа старого
CLI остановилась на неверном ожидании внутреннего текста ошибки вместо
редактированного CLI сообщения. После этого restore не выполнялся. Оба own
servers exited0, source/binary pins неизменны. Узкий отдельный V3 source fix
проходит review; полного Linux PASS пока нет.

WORK199V3 уточнил отказ: фактический old CLI exit1, пустой stdout и штатное
CLI сообщение PASS; logical rows не изменились. Однако полное сравнение bytes
state не совпало, поэтому общий FAIL и restore NOT-RUN сохранены. Изменённые
пути первоначально не сохранены: SQLite WAL/SHM — только гипотеза, не диагноз.
Перед изменением assertions нужен отдельный reviewed diagnostic с before/after
maps. Подмена сравнения более слабым ради PASS не выполнена.

WORK201 CONTRACT APPROVE202 получен; реализация узкого локального Git opt-in
идёт отдельно в WORK204. Это не native permission activation и не config RPC PASS.

WORK204 теперь **SOURCE APPROVE202** после P1 effective legacy-config rejection;
23-file fingerprint `eea3f68a765ec122278cda976774493cbc0859e5ea22ee947f3f98cce53c9714`.
Author/full targeted race и independent13cases PASS; actual config RPC/commit/
OS-negative ещё NOT-RUN. Отдельный QA готовит config-only WORK206 перед model turn.
WORK205 добавил read-only native `worker automation-preview` для локальной
подготовки pins/consent без собственного Go helper. Six-file SOURCE APPROVE202,
independent worker/cmd unit race PASS. Команда ничего не включает и не меняет
config/state/token/server. Новый полный Go race выполняется до нового candidate.

Final204+205 full `go test -race ./...` PASS: store102.833s, worker13.130s,
acceptance30.352s, web35.919s; все остальные packages также PASS. Web/infra source
не менялись после103frontend/TypeScript/Next build/54infra PASS3e6, их результаты
сохраняются с прежними source pins. Новый clean candidate готовится перед
отдельным WORK206 config-only process; actual native pipeline пока NOT-RUN.

WORK199V4-P2 actual Linux **PASS7/7**:4projects/9runtimeDB, format2 snapshots,
schema5→6 со всеми прежними logical rows и4историческими copies; отказ actual
old273; offline restore с отзывом authority/повышением epochs и сохранением
истории/reservations. Только exact SQLite SHM/empty-WAL aux отмечены отдельно;
непустой WAL, основные БД, snapshots/history не исключаются из строгой проверки.
Report `e0eb6cba41711cb163f5136afa43193a9521a63a0388a1ccf4f9ddd9c469bc26`,
independent EVIDENCE APPROVE202 получен. Typed offline fixture остаётся storage
proof с synthetic invalid refs, не live workflow/native/systemd/Next proof.

WORK206V3 actual config-only **FAIL**:2MCP calls доказали5/5disabled connectors,
один native app-server не выдал permission digest. Owned exit proven, forcedfalse,
config/source/binary pins неизменны, model0. Raw dialog продуктовый helper не
раскрывает, поэтому причина ещё не классифицирована. Нельзя объявить legacy/
profile/OS-denial без фактов. Отдельная QA-only private diagnostic WORK207
получил CONTRACT APPROVE202; source review ещё впереди до ONE config RPC,
product8c не расширяется.
Native commit/OS-negative/full pipeline и production activation всё ещё pending.

WORK207 ONE diagnostic run завершился FAIL до config response: initialize
получен, затем notification `remoteControl/status/changed` с metadata
`emittedAtMs` (int64 по schema установленного CLI) отвергнут QA parser.
Root прочитал только whitelist field names/types и method enum из installed
schema; raw values не публиковались. Native wait known exit0 после EOF cleanup,
broker exit1, owned proof proven/!forced; config/source/binary pins unchanged,
model0. Это конкретная protocol incompatibility до проверки permissions, не
OS-denial. Отдельный207V2 source разрешает только штатную notification metadata;
SOURCE review и новый root GO до повторной диагностики обязательны.

WORK207V2 после нового SOURCE APPROVE202/GO: diagnostic CAPTURED, три fixed
responses, native known natural0/EOF/drained, model0/config/pins unchanged.
Safe projection показывает expected selected profile/never/no legacy/five
disabled MCP/requirements null. Effective profile отличается от intent только
typed-null map entries. [WORK208](work/WORK-208-native-config-compatibility.md)
задаёт узкую protocol/typed-null совместимость без изменения rights/overrides;
CONTRACT APPROVE202 получен, separate author208 делает regression units/source
delta. Independent evidence review подтвердил207V2 CAPTURED diagnostic-only.
Это ещё не обычный product permission PASS. QA209 готовит только план normal
proof/native commit/direct synthetic OS-negative, до source approval208 новые
process harness sources не создаются.

WORK208 FINAL SOURCE APPROVE202 получен на5-file fingerprint
`10cd3bbc4754ceb675f0169e94e9bc74ec3721a2f454ee4d298c6d8c64e0b3c4`.
Промежуточный c3d сохраняет REQUEST CHANGES из-за нового blanket include_ reject;
финальный delta удаляет эту регрессию, сохраняя exact permission-instruction
true-only и прежние guards. Author targeted/full worker race и independent
regression races PASS. Root полный Go race на final frozen composite ещё идёт.
QA209 PLAN APPROVE получен, отдельно пишет минимальный A product-proof harness
с missing-candidate gate. Candidate build/runtime ещё впереди. WORK203 source
получил3P1 REQUEST CHANGES (credential cleanup/caps/cleanup deadline), tester
исправляет private v2 до новых source review; native pipeline не запускался.
WORK210 PLAN APPROVE согласовал genuine negative-review fix и controls/UI,
это не process/runtime acceptance.

Final208 composite: root `go test -race ./...` PASS на exact10cd3…,275 source
pins unchanged. Manifest SHA
`d203adea5ef3f7f6538354a6b67462f38e1bea54b3207d73cc5b7532312df1f0`.
Private proof `.local/task014/root208/final-all-packages-proof.json`;
чистый commit/build готовится до actual QA209A. Web/infra source не изменились.

Это сохраняет общий статус **IN PROGRESS / NOT READY**. Новый полный native
pipeline, отрицательный fix/control, browser UI и production
activation ещё обязательны. [Инструкция](automation-guide.md) описывает
реализованный интерфейс кандидата, а не уже установленный автоматический режим.
