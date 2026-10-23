# WORK186 — registry, transport и интеграция автоматического цикла

23.10.2026. Автор/координатор root. Независимое architecture/security
APPROVE183 получено для frozen automation contract181 до реализации.
Core/store реализует отдельный author185; reviewer183 не автор реализации;
новые process/E2E будут отдельной QA стадией после source approve.

## Область записи

Root: `internal/projects/` authority/current epoch wrappers, execution-capability
native authentication и lifecycle scheduler; `internal/transport/` typed REST/MCP
и resources; `internal/bridge/` scoped intake/own-step tools; worker integration,
capable infra и browser/Next UI могут получить отдельных исполнителей с точными
границами. До передачи эти области принадлежат root. Общие core/store DTO и
миграции принадлежат185; изменения согласуются по экспортированным интерфейсам.

## Обязательства интеграции

- Registry→legacy guard→board transaction держатся через authoritative mutation;
  scheduler не хранит human bearer и не создаёт human/coordinator sessions.
- Native execution capability не проходит catalog/browser/admin/intake/host
  routes, forced attempt binding действует и при отсутствующих wire полях.
- Один workflow intake и receipts в UI/MCP; strict schemas, bounded resources,
  current grants до replay, без автоматического расширения native tool consent.
- Local worker исполняет только owner-approved aliases/providers/check presets;
  exact Git/test proofs, process control и recovery подтверждаются отдельно.
- UI принимает требование, показывает setup команды/компьютера/лимитов, этапы,
  вопросы, реальные причины ожидания и управление; не показывает макетный успех.
- Schema6 обновление и restore только capable all-project; реальный сервер
  остаётся273 до отдельной успешной source/QA приёмки.

## Статус

IN PROGRESS. После contract approve добавлены registry authority guard,
service wrappers, bounded scheduler loop и local automation/check consent
validation с авторскими unit fixtures. Они ещё интегрируются и не являются
source-ready поставкой. Первый targeted worker test остановлен на compile из-за
двух ещё не дописанных185 store helpers; это не PASS и не ошибка native запуска.
Tests повторятся после compile-ready185, без временных permissive stubs.
Browser/Next делегирован отдельному автору187 с непересекающимися web paths;
source READY, авторские TypeScript/build/96frontend/full web-race PASS, отдельная
приёмка не проведена. Infra/preflight/capabilities переданы191, source READY,
54infra units и projects/cmd race PASS; Linux runtime не проверялся.
Worker product/tests и cmd/worker.go переданы отдельному автору193 вместе с
root local config/consent draft; root больше не редактирует эти paths. Root
сохраняет projects/auth/scheduler, transport, bridge и cmd/mcp lifecycle wiring.
Browser/process/native tests новой автоматизации — NOT-RUN. Native foundation
GUI остаётся FAIL на discovery timeout; отдельный QA184 проверяет гипотезу
Background process class без модели и изменения постоянной службы.

23.10 продолжение: реализованы registry-backed REST/MCP intake/control,
plan/question, workflow read/report resources; marker AutomationAvailable не
экспонирует исполнение на standalone Store без registry authority. Owner local
policy CRUD использует host proof через typed context, не idempotency payload.
Worker cap registration/attestation имеют отдельные REST, не MCP tools.
Bridge скрывает intake/control у managed children, execution child имеет только
scope tools; фиксированные check preset IDs отправляются в приватный IPC вне
bridge mutex, status resource не стартует процесс. cmd/mcp binding связан с193.
Scheduler запущен lifecycle serve, unexpected infrastructure error останавливает
HTTP; cancel/join до registry.Close, без background loops в Registry.Open.

Авторские проверки нашли и исправили интеграционные ошибки до приёмки:
Candidates требовал guarded context; target execution identity мешал проверке
legacy credential на default board (185 исправил узкую credential-validation
область). Registry Describe форсирует отсутствующие immutable поля и отвергает
противоречащие. Проверены реальные registry+SQLite REST dispatch→accepted→cap→
Describe→Next→own workflow read и отзыв источника credential для legacy иcatalog.
Есть MCP SDK integration intake/resource/REST exact receipt, strict JSON/query,
project isolation и revoked replay. Это in-process integration, не native CLI.
Полные race пакетов projects/transport/bridge/cmd выполняются; source freeze и
independent review ещё впереди. Production/schema6/native automation NOT-RUN.

## Source freeze 23.10

SOURCE READY, independent review ещё не проведён.29 файлов root перечислены в
private `.local/work186/source.json`, manifest SHA256
`77de3acaae478c9e0b3f2edaf0fbe27ff841d9ea73dbcf52c1f7a405514ac67a`.
Worker193 и shared core185 остаются отдельными областями review; manifest не
подменяет их fingerprints. Root прекращает product edits до findings.

Final author checks: полный `go test -race` projects23.130s, transport3.672s,
bridge14.869s PASS; cmd2.672s PASS отдельным последним запуском после193helpers
стали compile-ready. Все tests in-process/SDK/SQLite; nativeCLI не подменяются.
Meaningful regressions дополнительно покрывают localpolicy badproof/revokeproof,
обе legacy/catalog цепочки cap authentication и credential revoke; archive до
claim → schedulerstop → poll/accepted/stopped через archived readguard.
Обычный archive при живой task lease по-прежнему отвергается. Для этих worker
readguard операций дополнительно требуется текущий writer principal; viewer
получает403, не возможность OS control. Full managed regression PASS.
Последний diff whitespace check PASS.29 source-файлов frozen для reviewer189.

Serve lifecycle отменяет scheduler, завершает HTTP и ждёт loop до registry.Close;
ошибка scheduler закрывает admission. Исторический failure firstrace (viewer401
вместо403), частичная193compile и все предыдущие тестовые failures не скрываются:
они исправлены до финальных author PASS. Production/native/новые processE2E
TASK014 остаются NOT-RUN и отдельной QA стадией после source approve.

## Текущий checkpoint после WORK195/196

Прежний discovery prerequisite закрыт actual native GUI PASS5 в WORK195 и
независимым evidence APPROVE169. Root WORK196 выполнил finalize273: journal
complete, strict HTTPS и actual services/version pins проверены; свежий
all-project backup4проектов/9runtimeDB (+5history snapshots) подтверждён.
Это закрывает baseline prerequisite capable updater, но не TASK014 приёмку.
Web187 повторно SOURCE READY43 (100 frontend/14 targeted web race PASS);
infra191 получил scoped SOURCE APPROVE. Core185 повторно freeze60/race93.220s
прошёл, PolicyGet bounded history добавляется отдельным последним delta.
Worker193 source review продолжается; ownership uncertainty P1 требует
исправления/регрессий. Root29 product pin неизменён. Новая автоматизация
на VPS и Test_internet Mac пока не установлена.

## Cross-domain HTTP terminal regression

По finding189 выполнен узкий reopen root: два новых in-process unit файла
`internal/worker/automation_http_fixture_test.go` и
`automation_http_integration_test.go` принадлежат root (исключены из193 manifest).
Настоящий SQLite Store выполняет enrolled policy→dispatch→native session/lease→
pause→owned stop terminal report. Worker remote отправляет run_exit по HTTP,
получает фактический Store ACK с old attestation epoch/new current epoch;
AckReport/flushAutomation освобождают local checkout lock. Повторная acquisition
проходит, core resume использует прежний native UUID/run и новую generation.
HTTP fixture проверяет действительный bearer/session/path; registry checkpoint
заменён typed test authority, native process здесь не запускается. Его real
полномочия отдельно проверены registry integration выше.

Targeted race PASS (private http-terminal-race.txt). Первоначальный тестовый
fixture имел запрещённый нулевой continuation budget; это отдельно исправлено
в fixture до PASS без ослабления product gates. Не native/process acceptance.
Новый root SOURCE READY31: `.local/work186/source.json`, SHA256
`55d366d105fa9e263c548fb814ba7804d4660e4b74610afd64f3b0a1976531b4`.
Дополнительно api.md описывает bounded PolicyGet, terminal ACK metadata и
check_aborted. Независимое review нового delta ещё впереди.

Общий integration race всех product packages PASS (store97.413s, worker10.910s,
web35.227s); весь go test FAIL из-за двух старых schema5 assertions в existing
process acceptance. Отдельный tester WORK197 обновил их до6 и actual authority
checks; изменённые process tests ещё ожидают собственного source review.
Это не final composite/source-pin run: authors вносили узкие review corrections.

Final frozen composite checks: `go test -race ./...` PASS, включая весь
acceptance process package31.564s, store100.737s и worker12.526s. Frontend103
unit/TypeScript/Next production build PASS; infra54 PASS. Эти результаты
предшествуют отдельным actual native pipeline и Linux restore QA; production
273 не менялся. Подготовленные source/docs проверены на known local tokens,
provider-key/private-key patterns: findings0, private runtime не staged.
