# TASK-011 — координация

20.10.2026. Ветка `codex/coordination`, старт clean checkpoint
`343b93c71297c58c972f73fed4d1d4ca3a349740`. Прямое продолжение порученной
пользователем очереди profiles → coordination → managed execution.
Статус: INDEPENDENT QA IN PROGRESS. WORK147 architecture APPROVE,
WORK150 общий SOURCE APPROVE; root make check PASS. Контракт
[coordination.md](coordination.md) реализован. Source checkpoint f2edac94238c2b8fc267c37d9048187902bdf849.
Сквозная QA/Linux/production/evidence review ещё не завершены.

## Владение и последовательность

- Root: канонический контракт/TASK, operations/infra, интеграция, release/VPS.
- WORK147: отдельный независимый архитектор, до реализации.
- WORK148: отдельный backend автор, core/store/projects/transports/web/CLI и
  авторские unit/integration, без process/E2E.
- WORK149: отдельный frontend автор, Next и клиентские unit.
- WORK150: независимый reviewer source/tests/operations.
- WORK151: root operations и интеграция.
- Независимый reviewer: source/author tests/operations и повтор после исправлений.
- После source APPROVE отдельный tester: process/E2E, затем review новых тестов.
- Synthetic Linux migration/update/recovery/restore → full production backup →
  reviewed capable update → HTTPS/preservation evidence → final review/closure.

## Согласование объёма

Пользователь20.10.2026 явно подтвердил «Да, этот контур» на вопрос о назначении
задач постоянным профилям, управлении зависимостями, причинах блокировки и цикле
review → QA в панели, при самостоятельном подключении работников. Contract
реализует именно этот ручной контур; managed execution остаётся следующим этапом.

## Исходные границы

Ядро уже имеет role-based atomic next, leases/fencing, зависимости и SHA/generation
review gates. TASK011 использует их как единственный источник правил. Новая
плоскость координации связывает задачи с постоянными профилями и даёт человеку
рабочие действия/объяснения в панели. Конкретные wire/data решения фиксируются
в `coordination.md` после независимой архитектуры; черновой план не приёмка.

Provider execution, автоматический dispatch, процессы работников, clone/GitHub,
credentials и полный V1 вне этапа. Browser human session не подменяет отдельного
developer/reviewer/tester и не завершает их работу от их имени. Профиль не
credential/presence/worker. Неизменный исходный документ не редактируется.

## Обязательная сохранность

Старые проекты, данные, ACL/epochs, cookie/logout, session snapshots,
leases/fencing, независимость сессий и SHA gates сохраняются. Любая новая
persisted semantics требует capability floor; schema migration — полного
all-project snapshot до DDL, additive preservation и только roll-forward.
Initial TASK009 apply и DB rollback запрещены. Tandem/соседний davinci-gui-
не используются. Credentials/реальные данные только private ignored .local/VPS.

## Исходная поставка

TASK010 COMPLETE/FINAL APPROVE: product `cbf202086431335d4657c722a0f75328a09dc4ea`,
closing checkpoint343b93c. Подробные hashes/runtime/backup/evidence — TASK010.
Это переданный исторический checkpoint; актуальность VPS проверяется до поставки.
Физический второй ПК, provider workers и полный V1 остаются NOT-RUN.

## Промежуточные результаты реализации

WORK149 Next source ready: client67/67, types/format/build PASS. Независимый150
web source slice APPROVE, включая explicit draft rebase, successful QA gate и
отключение действий при mixed task/coordination versions. WORK151 infrastructure
155/155 PASS с независимым повтором и slice APPROVE. Backend148 author tests ещё
в работе; общего source APPROVE нет. Новый tester152 пока только read-only план,
process/E2E ещё не пишет и не запускает. Production не переключён.

## Source APPROVE и общий check

WORK150 общий SOURCE APPROVE; обязательные findings закрыты: stale generation
не становится admin fallback, current/positive различаются, bounded explanations
детерминированы и не теряют reviewer/tester причины, schema4 exact singleton.
Binding74source/author-test files SHA256
`5cd77400d904fe72783b3beabda63b1fa3a4137ddfc914e651a3347548cc672a`; root
независимо повторил вычисление и получил тот же hash. Backend148 полныйGo/race/vet
и отдельные finalreader regressions PASS, Next149 client67/types/format/build PASS.
Root полный make check PASS: Go/all/race/vet/build, прежнийembeddedweb suite,
infra155, Next67/types/format/build. Private log `.local/task011/full-check-source.log`.
После этого tester152 разрешены новые process/E2E. Это ещё не closure: localQA,
Linux/update/restore/production/HTTPS/evidence-review остаются NOT-RUN.

QA candidate собран root из отдельного clean managed checkout f2edac9:
Go VCSmodified=false, macOS SHA256
`daf342c2bb7d714af7d4aceea6910de31eb9e6ddc0751b926ac667284e0bf8ac`.
Private binary/proof `.local/task011/candidate`; Next localbuild upstream3101.
Tester152 получил фактический binary и разрешение actualprocess/Chrome run.

## Независимая local QA

Tester152 afterSOURCEAPPROVE: actual process-only13/13 PASS, updater8/8 PASS;
два последовательных полных actualprocess+Chrome прогона23/23 PASS с16screens
каждый, source_unchanged:true. Финальный local report
`.local/task011-qa/run-1792528339624/results.json`, предыдущий1792528244009.
Candidatef2edac9 cleanVCS/macOS SHA выше, localNext upstream3101 и BuildID вreport.
Проверены actual3→4/7DB/profilehistory/session/cookie/lease/blob preservation,
fullrestore3/4+allboardauthreset, routing/replay/concurrency, gates/checkcycle,
human/architect handoff, MCP12/resource+actualassignedcreate, browserdraft409/
rebase/committedlostretry/dependencies/transitions/checks/projectA-B/twotabs,
1440/390/light/dark. Viewport390 не физический телефон/второйПК.

История FAIL сохранена в WORK152: неверный test URL, test locator и преждевременный
URL read исправлены только тестировщиком; product не менялся. Отдельный catalog
read error на lateA/switch run1792528150784 не классифицирован (не объявлен
test-only); диагностические assertions HTTP200/реальныйJSON включены в два
следующих полных PASS. Причина прежнего единичного transient неизвестна и остаётся
ограничением evidence. Reviewer проверяет новые tests/localreports отдельно.

Updater simulation явно отделена: реальные Go/SQLite/crash/resume, systemd/Next/
UID adapter simulated. Reviewer150 независимо повторил updater8/8; actualLinux/
hardenedstage/productionguardreplacement/HTTPS/fullbackup впереди.

Финальный дополнительный required keyboard slice: actual triggerfocus→Enteropen/
autofocus/Tab+ShiftTabtrap/Escapeexactfocusreturn, POST0 и Task unchanged. После
него новый полный frozen run1792528643872 **PASS24/24**,16screens,284browser
responses, page/asseterrors0/prefetch0; sourceunchanged:true. Report SHA256
`c29a27320c1c6b49c066aa3a4682cd26b3df4bda8a07110fbe5bbd0b3e46387a`.
Дополнительный nativeOS select popup в headlessChrome **N/A**, независимо
воспроизведено безReact; функциональный выбор профиля проверен прежними forms.
Полный axe/accessibility audit NOT-RUN. Новые19tests manifest SHA256
`983d757a7668080929652e167f985a9a874b4e15da7fefed25a7ff033ee4bdae`.

WORK150 FINAL TEST SOURCE + LOCAL QA APPROVE: current19testhashes совпали с
final24report, source productdiff отf2edac9 пуст. Production readonly011 отдельно
APPROVE до запуска, scriptSHA256
`5cb2b9b29a8ca6ac2bae87b6a45ff8b71497f7208dfb2a25de5c5206b0828395`.
Переходим к immutable Linux release/exact process/update/hardened stage; production
ещё cbf/schema3, переключение и closure NOT-RUN.
