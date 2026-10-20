# WORK-145 — независимая process/Chrome QA постоянных профилей

20.10.2026. Отдельный tester145 после source APPROVE WORK143.
Write ownership: tests/process/task010/**, tests/browser/task010/**,
две прежние schema assertions tests/acceptance/{backup,migration}_process_test.go
и этот WORK. Product/infra source, каноническая TASK и VPS не изменялись.
Updater fixture отдельно у tester146. Credentials и все данные только synthetic,
private ignored .local. Tandem/соседний проект не использовались.

## Реализация проверок

- Прежние process tests сохранены: backup metadata теперь ожидает3, migration
  schema1→3 дополнительно проверяет наличие обеих profile authority tables.
  Прежние historical bytes/hash/restart/fencing assertions не ослаблены.
- Новый actual-process harness использует точный baseline5454, проверенный
  по source revision и binary SHA до запуска. Создаёт default+Alpha+Beta,
  реальные memberships, legacy tokens, browser cookie/session, live lease,
  note, events/replay и design blob на настоящем прежнем executable.
- Под остановленным writer запускает реальную projects migrate-storage, сверяет
  все прежние rows каждой registry/board/web table и blob hashes. Повтор миграции
  ничего не меняет. Реальный format2 snapshot2→new restore/reset и snapshot3→
  restore сохраняют task history/notes/repositories/idempotency/blobs/profiles/
  revisions; historical events остаются, security state сбрасывается отдельно.
- Native CRUD/replay/version conflict/history/archive/restore; strict Describe
  pair/role, immutable snapshot и leases; same names разные stable ID/projects.
  Viewer, downgrade, revoke→regrant и cached mutation/replay отклоняются.
- Настоящий stdio MCP сохраняет12tools; profile resources читаются, чужой profile
  и SID отклоняются. Bridge sanitization upstream errors сохранена.
- Browser JSON CRUD, validation/new action capability, conflict/draft/history,
  viewer direct POST и stale browser actions проверяются через реальный Go.
- Missing schema3 revision table оставляет startup failclosed до listener,
  старая таблица не чинится, прежние blob bytes не удаляются. Старый5454
  отвергается новым capability guard; новый бинарник разрешён.
- Chrome: реальный POST сначала committed через route.fetch, затем ответ
  потерян; retry полностью совпадает body/path/action и не создаёт duplicate.
  Конкурентное update даёт409 и сохраняет dirty draft/base version до явного
  применения к новой версии. UI history/archive/restore, две вкладки и настоящий
  задержанный ответ A после switch B, viewer без controls, snapshot в session
  и unbound legacy отдельно. Screens1440/390/light/dark, отсутствие horizontal
  overflow/page errors/asset failures/private prefetch.

## Фактические результаты и границы

Local17checks actual process+Chrome: PASS, после внесённых reviewer усилений.
Финальный report: `.local/task010-qa/run-1792505088641/results.json`,
SHA256 `0f1a21f57728245c5e877a1776fb143fd690c24568cf0df13c04381b14db02e5`.
17/17 PASS;16screens. Все10 test-source hashes совпали с проверенным тогда
snapshot, changes-during-run отсутствует. Последующая test-only коррекция
production script отдельно зафиксирована ниже; сам local process/Chrome code
остался прежним. Независимый final review143: APPROVE new test sources, local evidence и
production readonly test safety; reviewer пересчитал report и все10source hashes. Node24.19.0, Chrome155; darwin/arm64. Это локальный build из approved
product source, source_revision:null честно сохранён — clean release provenance
и фактический Linux SHA даёт отдельно координатор при Linux rehearsal.
Проверенный local Go binary SHA256
`ff598499f30ca6da6b3caa4a074fdbe056f0bd77339f49b9e4031895375d3043`.
Baseline5454 executable SHA256
448602f68a24688b86ba2aaec06a36e064765d22b4bb3b3c5f6d7bd92797e95d
проверяется по root build proof; фактическая точная строка фиксируется report.

В промежуточных QA runs выявлены ошибки самого теста, исправлены без product
изменений: native full Sessions ошибочно смешан с summary ControlSessions;
после400 reused bound capability вместо новой; late response держал первый GET
до появления shell; mobile theme toggle требовал открыть desktop; session profile
находится во вкладке «Сессия». Полный повтор выполняется на новом fixture.
Эти early FAIL не объявляются product defects или итоговым PASS. Отдельно
замечен screenshot paint artifact при быстром viewport resize: после scroll0,
fonts.ready и двух animation frames повторная съёмка16screens корректна,
DOMassert подтверждает одну шапку. Detail390dark/light визуально сверены;
исправления product source не потребовались.

Production readonly script дополнен profile list/resource/owner controls и own
session, остальными прежними screens. Разрешает только собственный login,
explicit project open/logout; не создаёт production profiles/tasks/projects.
Reuse reviewed TASK009 production-support с guaranteed cleanup/report; diagnostics
не содержат реальные поля/токен. Root проводит deployment/HTTPS отдельно.

Linux actual release process-only, updater146, production HTTPS/full backup и
физический второй ПК: этим tester NOT-RUN; результаты координатор фиксирует
после фактического исполнения. Полный V1/provider/worker execution не заявляется.

## Первый HTTPS run cbf и узкая корректировка теста

Root выполнил approved readonly script на production revision
`cbf202086431335d4657c722a0f75328a09dc4ea`. Report
`.local/task010/deploy/https-cbf-01/production-readonly-results.json`: **FAIL**
после3успешных checks, до screenshots; cleanup logout PASS, browser closed.
36browser responses200, page/CSP/asset errors0. Ошибка отмечена прежним stage
MCP_READS, хотя предыдущие MCP checks уже прошли.

Причина — ожидание теста для пустого browser списка. Фактический
`internal/web/browser_page.go` использует `profiles,omitempty`: пустой список
законно отсутствует, Next отображает `page.profiles ?? []`. Тест требовал массив
без проверки отсутствующего поля. Исправление только в readonly script:
отсутствующее поле нормализуется в[], иное значение по-прежнему обязано быть
массивом; null/string/object не превращаются в успешную проверку. Native MCP
resource остаётся строго массивом: core.ProfilesPage имеет `json:profiles` без
omitempty, Store создаёт пустой[], поэтому native assertion не ослаблен.

Добавлены отдельные stages PROFILE_BROWSER_READ/PROFILE_RESOURCE_READ/
PROFILE_OWNER_CONTROLS. Read-only allowlists, Origin/auth/cookie/ACL checks,
обязательные cleanup и приватность не изменены; production profiles не создаются.
Product source/data не исправлялись. Это test-only correction после прежнего
approved script SHA1d0801; старые reports/hashes сохраняются как история своего
snapshot, не объявляются совпадающими с исправленным production test файлом.
Независимый review143: APPROVE readonly HTTPS rerun, corrected script SHA256
`04360016413b57b5e7ef7596da917f68812516d2de5c00202e0f4d30d7ee359f`.
Product/operator immutable cbf не менялся; actual HTTPS повтор выполняет root.
