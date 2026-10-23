# TASK-011 — координация

20.10.2026. Ветка `codex/coordination`, старт clean checkpoint
`343b93c71297c58c972f73fed4d1d4ca3a349740`. Прямое продолжение порученной
пользователем очереди profiles → coordination → managed execution.
Статус: COMPLETE. Реализация, source/test review, local/Linux QA, production
apply/HTTPS/finalize и итоговая независимая evidence приёмка завершены:
WORK150 FINAL APPROVE. Контракт [coordination.md](coordination.md), product release
`e53c18c9643b9ee086141d0f227ab485f4f88cac`. Source checkpoint f2edac9.

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

## Immutable Linux release и actual stage

QA sourcecommit `e53c18c9643b9ee086141d0f227ab485f4f88cac` (clean), product
unchanged отf2edac9/review74manifest. Linux Go SHA256
`a03f64d5ff008b7f63023117fb7b1c15c33e5617ce7ac08ffc6440610315269d`,
Next BuildID `HHViYVPwDuQx7uj4Uk1YC`, manifest SHA256
`cb97c3dd40df2aa8340dd0e3ca2573af4d44a3f956c8498734efadb978b212c9`.
Проверены53Go/1256Next members; inactive upload без переключения.
Exact Linux product13/13+updater8/8 PASS, pinnedoldcbfSHA/newELF/source/test
hashes и cleanupports подтверждены. Private release-proof.json/
linux-product-proof-01.json/linux-update-proof.json в deploy.

Actual hardened Next stage PASS: owned stagePID, LinuxNode24.19/sharp/BuildID,
NRestarts0/MemoryPeak46825472, временный unit удалён,7894free. Stage journal staged,
прежние Go/Nextcbf pointers сохранены, helpers/runtimeconfiguration PASS,markers0.
Private stage-runtime-proof.json; actualschema/update/HTTPS/finalbackup впереди.

## Итоговая реализация и изменённые области

- `internal/core/{coordination,types,catalog}.go`, `internal/store/{assignment,
  check_create,coordination,tasks,transitions,dependencies,sessions,notes,migrations,
  schema_validation}.go`: pinned task routing, audit, atomic checkcycle, explain,
  admin rules и schema4; авторские regressions в тех же пакетах.
- `internal/projects`, `internal/transport`, `internal/bridge`, `cmd/agent-board`:
  scoped ACL/epochs/preflight/native REST/CLI/resources/MCP12 и capabilities.
- `internal/web/{coordination,actions,pages,browser_page,control_pages,mutations,
  contracts,http}.go`: browser actions/recovery/metadata/permissions и tests.
- `web/src/features/coordination`, task/create screens, lib coordination/types/API/
  labels, common assignment renderer и styles: рабочий ручной контур Next.
- `infra/accord/projects` guard/upgrade/storage/snapshot validation/history,
  infra units и existing acceptance schema fixtures; новые process/browser011 и
  updater011 tests; канонические docs/AGENTS и WORK147–152.

Полный список воспроизводится diff от стартового343b93c до producte53c18c и
closing docs checkpoint. Зависимости и lockfile не обновлялись; ветка main и remote не изменялись.

## Фактическая поставка и приёмка

Рабочая панель открывается по индивидуальной ссылке подключения; адрес не хранится в текущей документации.
Production Go/Next immutable release **e53c18c9643b9ee086141d0f227ab485f4f88cac**;
LinuxGo SHA/NextBuildID/manifest выше. WORK150 approved exactLinux+actualstage
перед apply. Обновление только reviewed `projects/upgrade.py`, не initial apply.

| Барьер | Результат |
| --- | --- |
| Архитектура/source/author tests | APPROVE147/150, root make check PASS |
| Root full checks | Go/all/race/vet/build, embedded web, Next67/types/format/build, infra155 PASS |
| Независимая local process+Chrome | PASS24/24,16screens, requiredkeyboard, sourceunchanged |
| Independent updater repeat | PASS8/8, отдельно declared host simulations |
| Exact Linux product/updater | PASS13/13+8/8, actual ELF/source/tests binding и cleanup |
| Actual hardened stage | PASS, owned process/Node/sharp/BuildID, NRestarts0/7894free |
| Actual capable apply | PASS, full snapshot доDDL, guard replacement и preserved3DB |
| External HTTPS readonly | PASS7/7,56screens,858browser200/errors0/prefetch0, ownlogout/browserclosed |
| Finalize/runtime | PASS, journal complete, actual ownedGo/Next/effectiveconfig/helper hashes/capfloor |
| Final all-project backup | PASS, format2/schema4+web2/default+registry/history, validated наVPS |

Production сейчас содержит только default (0дополнительных проектов), сохранность
подтверждена для3реальныхDB. Default+2projects/7DB проверены на отдельных synthetic
Linux/local fixtures. HTTPS реально прочитал2existingtasks coordination detail/
resource/tab; profile/task/dependency/check mutations в production не выполнялись,
они проверены local и exactLinux. Профили/provider prefs не запускали workers.

Fresh pre-snapshot:
`/var/backups/accord-web/projects-20261020T204410Z-98c7caa68b0e4312a5db97aaa09301cc`,
manifestSHA `a4e06648b561c5667071fc9ca5d64a201326951a1a9e69b845a13fe6dd56d013`
(его bytes совпали с последнимTASK010 snapshot, поскольку исходные данные те же).
Final full backup:
`/var/backups/accord-web/projects-20261020T204623Z-d90daa53b0614bfab1cdcbf041a9abca`,
manifestSHA `28b9886fd12e53d0559091d94b7771346a86b016e500ac46b1d96c31fc600a39`.
Оба повторно validated наVPS; реальныеDB не скачивались. Исторические migration
copies сохранены. Authority marker UID999/regular0600/bytesSHA совпадают с
pre-snapshot; root-owned helper policy к нему не применялась. Старый guard3
проверенно заменён новым4; effective services/timers active,NRestarts0,markers0.

Private evidence `.local/task011/deploy/`: release/transfer/Linux proofs,
stage-runtime-proof.json, apply-preservation-proof.json, apply/finalize operator
outputs, final-runtime-proof.json и https-final-01/production-readonly-results.json.
HTTPS reportSHA `06aa3b53658adf869192d83f6a0c19dd06c92bfa3233cd8e2154d39fc59603a2`,
approved scriptSHA `5cb2b9b29a8ca6ac2bae87b6a45ff8b71497f7208dfb2a25de5c5206b0828395`,
scriptunchanged. Final journal complete не сбрасывал credentials/membership/
sessions/history и не использовал DB rollback. Diagnostic cache incident и узкая
проверенная архивная remediation остаются в WORK151, independently approved150;
неизвестные recovery/temp artifacts не удалялись.

## Границы и следующая передача

Это завершение согласованной ручной координации и её безопасной поставки:
итоговый WORK150 FINAL APPROVE, mandatory findings отсутствуют. Физический второй ПК, native Claude/Codex новый сквозной цикл,
provider support/execution, managed workers и полный V1 — NOT-RUN. Полный axe/
accessibility audit NOT-RUN; native OS select picker headless N/A, required
keyboard диалога PASS. Одиночный historical catalog transient не классифицирован,
дальнейшие полные прогоны прошли; история сохранена, rootcause не выдумана.
Backup остаётся только на прежнемVPS. Remote push/merge/PR не выполнялись.

Следующий прямо порученный отдельный чат — «Управляемый запуск»: сначала
контракт/независимая архитектура, затем bounded implementation/review/QA/delivery.
Pinned profile не credential/identity/session/process; current schema4 floor,
ACL/epochs, snapshots, leases/fencing, SHA/current-latest gates и полный backup
обязательны. Managed execution в этом чате не начинался.
