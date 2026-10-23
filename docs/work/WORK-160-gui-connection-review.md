# WORK-160 — независимое ревью подключения GUI к Accord

22.10.2026. Reviewer, отдельный от автора инструкции и production-оператора.
Область записи: только эта карточка. Implementation, GUI и runtime не меняю.

## Основания и границы

Прочитаны AGENTS, requirements/architecture, multi-project, agent-profiles,
coordination, TASK012 capability floor, CLI `projects`, bridge lifecycle и
all-project snapshot/startup-hold primitives. Пользователь подтвердил GUI
«Давинчи 2.0», второй участник — Windows + Claude. Поручение подключения
разрешает узкое изменение его инструкций с Tandem на Accord; остальной GUI
не входит в эту работу. Tandem не используется и не перенастраивается.

Начальный source draft инструкции проверен; финальный source verdict pending.

## Предварительная оценка плана

Отдельный GUI project с реальным credential-free remote, новый catalog-only
credential Mac и сохранённый credential второго участника с editor grant только
GUI соответствуют контракту. Label не используется вместо identity ID. Новая
identity не получает default access; существующие projects/ACL не меняются.

Безопасная альтернатива восстановлению потерянного credential — явная offline
`projects token-create` текущим exact capable binary. Не использовать legacy
`token create`/`infra/accord/token.py`, default owner bootstrap, recover-owner,
maintenance-reset, migrations, rollback или deployment ради этого подключения.

Для offline CLI нужны admin/maintenance lock, durable owned startup hold,
остановленный writer, full verified snapshot default/registry/all projects и
проверенная effective launch config. Output только новый private file 0600 в
private dir 0700. Файл секрета не печатается; diagnostics только allowlisted
metadata. При неизвестном исходе не выпускать новый credential вслепую.

Browser changes идут через fresh CSRF/server-issued action ID и expected
versions. Retry неизвестного результата сохраняет payload/action/project;
не создавать второй проект после timeout. Owner grant атомарен с публикацией.

## Обязательные проверки

1. До mutations: exact installed release/ELF SHA, capabilities включая
   deduplication1/coordination1/profiles1/registry1/ACL1/schema4 и guards,
   owned PID/effective paths, service health, all-project inventory/ACL baseline.
2. До offline issuance: проверенный полный backup и остановка writer под
   существующими locks/hold. После: прежние board/project/ACL rows сохранены,
   появился только ожидаемый credential; startup hold снят только своим кодом,
   Go/Next health и unchanged runtime/floor/config подтверждены.
3. После web setup: один GUI project, точный remote/branch, Mac owner и второй
   identity editor; чужие grants/projects прежние, default alias не подменён.
4. Client: private files/modes, ignored runtime/settings, exact scoped project,
   capabilities/client SHA, TLS readiness, initialize/tools/resources, fresh
   describe и scoped reads без секретов/сырых ответов в отчёте.
5. Claude: project `.mcp.json` содержит env references. Windows token только в
   окружении процесса, без literal в argv/config/history. Реальный Windows run
   и совместная работа двух физических ПК остаются NOT-RUN до проверки коллегой.
6. Один stdio bridge — одна role/session; внутренние subagents общего MCP
   не считаются независимыми автором/reviewer/tester. Разным ролям нужны
   самостоятельные клиентские процессы и server-generated sessions.
7. Активные GUI ссылки/правила Tandem заменены согласованно, исторические
   документы не объявляются актуальными. Process/E2E — отдельный tester после
   SOURCE APPROVE; новые тесты также проходят review.

Статус: **PRELIMINARY REVIEW**, финальные implementation/production evidence
ожидаются. Новый service release и полный V1 этой задачей не подтверждаются.

## Source review offline issuance

Прочитаны private `issue_remote.py` и `client.py` (содержат только код, без
credential values). **SOURCE APPROVE** для offline issuance двух новых
catalog-only credentials exact TASK012 binary: все прежние registry rows и
published board/web rows сравниваются; full snapshot, capability/effective
launch, owned hold и locks присутствуют. Возобновление unknown outcome
автоматическим повтором исключено one-time journal guard. Новый credential
коллеги не подменяет прежний, его старые grants/credential остаются прежними.

Для browser helper дан **REQUEST CHANGES** до web mutations: запретить
автоматические HTTP redirects и валидировать origin структурно (HTTPS,
hostname, без credentials/query/fragment/постороннего path). Текущий
`startswith('https://')` слабее native client contract. Исправление маленькое;
production catalog setup ждёт повторного source review этого helper.

## Source review инструкций

**SOURCE APPROVE** набора автора: `docs/gui-accord-onboarding.md` (e2ca1269),
GUI `docs/agents/accord.md` (78bedbde), `AGENTS.md` (baabfbf7),
`CLAUDE.md` (29632365), `docs/agents/parallel-work.md` (27d0dd3c).
Тексты сверены с bridge/CLI, ACL и ролью клиента; Windows Claude `${VAR}`
command/args/env и project trust сверены с официальной документацией
`https://code.claude.com/docs/en/mcp`. Ограничения Windows runtime, отдельного
ПК, subprocess sessions и управления агентами обозначены. Секретных literal
нет; реальный origin и user data в документацию не перенесены.

Независимые `git diff --check` обоих репозиториев: **PASS**.
Реальные `.mcp.json`/`.codex` настройки координатора и финальные metadata
пакета будут проверены отдельно после готовности.

Повторное **SOURCE APPROVE** узкого уточнения supported roles:
GUI `accord.md` f8e48b74, onboarding 536b5738. Enum сверён с `core.ValidRole`;
локальные GUI devops/security/design-reviewer/llm-quality не выдаются за
server roles и не получают неявный auto-map или обход gate.

## Повторное source review browser setup

`client.py` теперь структурно проверяет HTTPS origin и запрещает redirects;
замечания закрыты, **SOURCE APPROVE**. Дополнительно прочитан
`configure_project.py`: проверяется identity, create ограничен одним точным
GUI remote/name, существующий результат находится по каталогу, неизвестный
исход повторяет сохранённые action/payload/browser cookie. Для коллеги только
editor grant; итоговый набор members проверяется точно. Private pending/raw
responses/cookies не выходят из игнорируемой `.local/`. **SOURCE APPROVE** для
этого узкого catalog setup; это не blanket approve других действий сервера.

Точные private source SHA256 prefixes: issuance 7aa89549, client 7f86c65b,
catalog setup eae86f03. Sanitized `issuance-proof` независимо прочитан:
phase complete, release `64ed870b749289b545d29cd2ec7813c6266de54d`, registry и
all-board/web preservation PASS. Snapshot/runtime — evidence оператора;
credential values reviewer не читал.

## Source review локальной настройки

**SOURCE APPROVE** `install_local.py` c374473a. Exact проверенный macOS клиент
копируется в private home; Codex launcher читает origin/token из private files,
shared Claude `.mcp.json` содержит только env references. Удаляется только
active Tandem connector; прежние settings, полный untracked connector и
guide сохраняются в ignored backup. Содержание исторического guide сохраняется
под новым архивным header. Другие dirty GUI files не включены в запись.

Независимо проверены private0700 home config и GUI `.local`, ignore rules для
runtime/Codex config, отсутствие active package/Claude references на retired
connector path. При partial local install — ручная инспекция backups; source
не разрешает слепой повтор и не обещает crash-atomic filesystem transaction.
Реальный client run остаётся для отдельного QA после этого source approve.

После установки независимо read-only проверено: private client0700,
token/connection/launcher0600, regular non-symlink; Claude только `accord_gui`
с точными env references/project args; Codex только `accord_gui` с ожидаемым
private launcher. Mac client SHA256
`f17b4bb8dfa06dbdd9168687beb79c2fa4ce9b33579f6e58cca86932738bbfb5` совпал.
Active Tandem path отсутствует, recovery backup присутствует; local config и
backup игнорируются Git. Эти проверки **PASS**, token/origin не читались.

## Review QA source и пакета Windows

**SOURCE APPROVE** independent tester process smoke 61454676. Найден и
исправлен ложный PASS contender: теперь требует успешный authoritative
`next` с явным `task:null` и без lease; generic RPC/validation rejection не
считается успешной проверкой взаимного исключения. DTO, resource shapes и
browser owner update после cooperative release сверены с source. Только свои
labelled entities, явный task key, one-shot journal и остановка при unknown.
Done evidence прямо обозначает administrative smoke существующего checkout,
не реализацию или code approval; Windows/второй ПК не засчитываются.

**SOURCE APPROVE** public Windows handoff: UTF8-BOM PowerShell launcher
3812d5e4 и README 098d689f. Native Windows/PowerShell5.1, HTTPS/project
validation, token из private file в process env и восстановление finally,
интерактивный Claude, обычные trust/approval и no bypass. Runtime Windows
не выполнялся; исходники не содержат actual origin/token.

## Независимая проверка live preservation

Read-only проверка через pinned SSH admin transport (не Tandem network):
полный format2 snapshot заново validated, manifest SHA совпал с issuance
journal. Все прежние строки registry_meta/identities/credentials/projects/
repositories/memberships — точные subset текущих; количества additions
соответственно0/2/2/1/1/2. Новые memberships ровно owner+editor одного проекта.
Capability marker SHA неизменен относительно snapshot. Current exact
`64ed870b749289b545d29cd2ec7813c6266de54d`, capabilities, effective launch и
all-project backup command, owned Go PID/ELF/argv и active Next — **PASS**.
Вывод allowlisted booleans, без адресов, identities, credential hashes/values
или сырых rows. Изменяющих операций reviewer не выполнял.

## Финальный source QA с реальным Next UI

**SOURCE APPROVE** live smoke 6863610b и UI f953bd0f. Добавлен независимый
rendered Next desktop Chrome прогон team/task при live holder и после done.
UI разрешает только same-origin GET/HEAD и собственные login/logout; TLS не
ослаблен, проверяет текущее состояние, единственного держателя и исправленное
описание. Найдено и исправлено screenshot privacy omission: дополнительно
маскируются task creator/holder chips и ссылка server session. Ошибки и cleanup
выводят только sanitized stage/kind, без raw replies/credential.

Public Windows ZIP независимо проверен: ровно8files; SHA256SUMS покрывает все
остальные entries и совпадает после распаковки в память. Клиент exact SHA
`00a1a61df1ca3c723151fdcb570c2cae2a9ac84259f3facbeea3c0cb7659b612`, PE amd64;
MCP env-only с project argument, launcher UTF8-BOM. Архив SHA256
`54852249e7b3e57a035dae7c5d57d7e8717014f86fc12aa2071e87ad16033bdb`.
Проверки package **PASS**; Windows execution по-прежнему **NOT-RUN**.

## Финальный verdict по фактической QA

**APPROVE** source + tested connection scope. Независимо прочитаны WORK161,
private live/UI proofs и текущие source hashes: approved 6863610b/f953bd0f
не изменились. HTTPS/stdio/browser workflow29/29 **PASS**, live/done Chrome
phases по2 UI assertions **PASS**, logout **PASS**, page errors/failed assets/
unexpected business mutations0. Все четыре masked screenshots reviewer
просмотрел: live ровно один task holder, done ни одного, исправленное описание
и generation2 видны. Blocking findings0. Только свои TEST E-1/T-2/N-1;
задача/эпик завершены, общий smoke не выдан за product code approval.

Фактический рабочий объём: общий GUI project/ACL, private Mac config и public
Windows handoff, native coordination и web наблюдение/коррекция незахваченной
задачи. **NOT-RUN** actual Windows/Claude, второй физический ПК и реальная
совместная разработка. Actual Codex CLI discovery прошёл, business describe
**BLOCKED BY CLIENT APPROVAL**, `never` не обходили; это не успешная agent
session и не полный V1 PASS. Управление OS-процессами **N/A/NOT IMPLEMENTED**,
release/reclaim не называется stop/resume модели.

Финальный all-project backup после GUI/setup/smoke подтверждает координатор;
evidence будет проверен отдельно. Новый service release не выполнялся.

Final backup evidence получен и независимо проверен read-only на сервере:
format2 `validate` **PASS**, manifest SHA совпал, активный GUI со schema4
включён, current exact64ed и ELF SHA прежние, capable floor/effective launch/
all-project backup command/owned Go runtime/Next active **PASS**. Финальные
whitespace checks обоих репозиториев **PASS**. Итоговый verdict **APPROVE**
в перечисленных границах подключения; обязательств Windows/2PC/V1 не закрывает.
