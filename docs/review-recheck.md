# Повторное независимое ревью TASK-001 / WORK-008

Дата: 18.10.2026. Вердикт: **APPROVE** для указанной версии локального
пилота 0A. Блокирующих замечаний code review нет. R1/R2 закрыты.
Это разрешает отдельному тестировщику начать process/E2E; это не завершение
общей приёмки, не security approve и не готовность production/V1.

Проверяющий — отдельный агент, не автор WORK-003/004/008. Прочитаны AGENTS,
TASK, требования, архитектура, API, README, initial review/security и WORK-008;
профиль reviewer и общий процесс Давинчи применены только к качеству проверки.
Именованные Claude-модели здесь недоступны; автор и reviewer используют одну
доступную модель среды. Tandem прямо отключён пользователем. Source не менялся.

## Проверенная версия

Ветка `codex/pilot`, HEAD `9537cb737ee3d1f1e3d63eeccc6a81e2fa681698`.
Реализация ещё не закоммичена, поэтому идентичность задаёт SHA-256 manifest
ниже: **46 файлов**, digest текста manifest:
`9cc3437a3f55c5a46859d8610997fb7e14ceb0a0cbcfc3fcd3864d2b03d006b4`.

Включены все implementation/test Go-файлы cmd/internal, go.mod/sum, Makefile,
AGENTS и canonical docs/README. Текущие TASK/WORK, review/security и локальные
временные probes не входят в source manifest. Последующие новые E2E-тесты
должны пройти отдельное ревью; изменение проверенных файлов требует сверки
затронутой области и нового manifest. Копия:
`.local/evidence/review-recheck/manifest.sha256`.

## Результат проверки замечаний

**R1 закрыт.** `internal/store/gates.go` содержит общий recursive predicate
для explicit close и reconciliation. `mutation.go` вызывает reconciliation
после успешной domain mutation, до replay insertion и общего SQLite commit.
Новая blocking edge к done child открывает его epic. Переоткрытие источника
распространяется через завершённых промежуточных потребителей и разные эпики.
Child done/evidence/attempts сохраняются как история. SQL failure откатывает
edge, reopening, audit и replay вместе. Implicit review-target cycles fail closed.

**R2 закрыт.** Обязательства reviewer/tester ищутся по target глобально,
включая проверки другого эпика. Late assignment открывает epic target.
Для текущих SHA/generation действует последняя подходящая назначенная проверка
по числовому server T-N, включая T-9 → T-10. Pending/negative не обходится
старым positive; новый positive заменяет historical negative/stale.
Superseded positive больше не открывает явную старую edge: пользователь явно
выполняет unlink/link на новый источник. Unlink не отменяет обязанность роли.
Preplanned generation=0 связывается при первом claim и не даёт stale approve.

**Contract drift устранён.** REQ-12/API согласованы с правом live holder
на unlink только с его текущим generation. REQ-20/ADR-003/README различают
session/singleton renewal и task lease renewal: unrelated create продлевает
session, но не task. ADR-002 честно описывает JSON-per-task/session, отдельные
events/idempotency и транзакционную проверку ссылок; SQL FK для вложенных ссылок
не обещаются. Пределы локального пилота и SDK evidence сохраняются явно.

## Фактические проверки

Все команды выполнялись на локальном Go 1.27.2/macOS arm64 с переменными
GOTOOLCHAIN/GOCACHE/GOMODCACHE/GOTMPDIR из development.md.

PASS: шесть новых авторских top-level regression тестов и их reviewer/tester
subcases плюс три независимых временных unit/integration probes:

```sh
go test -overlay .local/evidence/review-recheck/overlay.json ./internal/store -run 'Test(Recheck|DoneConsumerLink|ReopenSource|ImplicitReview|CrossEpicVerification|LateCrossEpic|SupersededVerification)' -count=1 -v
```

Проверены late assignment, positive/negative replacement, SQL-trigger rollback,
exact replay/audit, cross-epic transitive cascade, same-SHA generation change,
implicit cycle и explicit edge replacement. Независимые probes дополнительно
проверили SQLite restart: late obligation и epic reopening сохраняются,
exact create replay не порождает событий, положительная проверка разрешает
закрытие, которое также сохраняется после restart; unrelated create продлевает
только session; отдельный non-requester principal как live holder вправе unlink
с актуальным generation, stale generation отвергается. Вывод: `targeted.txt`.

PASS: три первоначально падавших независимых R1/R2 probes первой итерации:

```sh
go test -overlay .local/evidence/review/overlay.json ./internal/store -run 'TestReviewProbe(LinkMustReopenDoneEpic|CrossEpicReviewObligation|CrossEpicSourceReopen)$' -count=1 -v
```

Вывод: `original-probes.txt`. Старый probe с неоднозначным ожиданием renewal
не использовался: новый независимый probe проверяет уже уточнённый контракт.

PASS: три существующих targeted transport/CLI integration сценария:

```sh
go test ./internal/transport ./cmd/agent-board -run 'Test(SDKStatelessToolsShareRESTState|SDKLeaseWorkflowAndRevocation|CallDescribeCreateGetAndEvents)$' -count=1 -v
```

Вывод: `transport.txt`. Official SDK и REST используют один Store, CLI
сохраняет describe/create/get/events контракт после исправлений. Это локальная
интеграция с SDK, не проверка Claude Code/Codex.

Все временные probes/overlay/выводы — только в игнорируемой
`.local/evidence/review-recheck/`; production source и авторские тесты не правились.
Авторский targeted store race PASS зафиксирован в WORK-008; это evidence
автора, не собственный race-прогон reviewer. Общий make check выполняет координатор.

## Качество и ограничения

Все production/test Go-файлы не превышают 300 строк. Неизменный исходник ТЗ
больше 300 строк — обоснованное исключение как входной артефакт. UI N/A.
Бизнес-правила gates централизованы; новых необоснованных дублей между REST/MCP
не обнаружено. Go generic `any` служит serialization/type parameters, обхода
доменной типизации этим не найдено. Зависимости не менялись; API transport
закреплены модулями и компилируются указанными integration tests.

SEC-01/02 source прочитан в связанной дельте, но профильный повторный security
вердикт выдаёт отдельный агент; этот отчёт его не заменяет.
NOT-RUN reviewer: полный make check/vet/race, новые process/E2E, реальные
Claude Code/Codex, две физические машины, Linux/Windows, remote TLS,
нагрузка, fault injection процесса/диска во время commit и production acceptance.
Новые process/E2E пишет tester после данного approve, затем их проверяют отдельно.
Push/PR/merge/release не выполнялись и данным approve не авторизуются.

## Manifest

```text
fd022e37a8d2937d16923285dd125c7531800db7cff2c7e0035448f0e4cec4aa  AGENTS.md
c00892be4f2d52262e0553a291e96c532c3f4afdbda0b9d26bbfd06710327493  Makefile
4b12544add7ca286a006497caa45407828efee418ddb9f10d68c7881ce076aea  README.md
ae9702f6beea7a112b510b4bc6f52e6da52f42ba82bbbfdea6f5c6b8dafa63ca  cmd/agent-board/client.go
54fd07eb51bc1c2c669f6ab5a22dd9fdac80bfa0d100b6735d6b4eccc4c6eaa6  cmd/agent-board/client_test.go
f13e4a9d4668ee7559a254795d028ba50d26bc3c3c3552e7227fdd3c686e23cb  cmd/agent-board/main.go
5555bf7e57e26017072f5c967384d09e212a066c01fd604ba4c744928bf31026  cmd/agent-board/token.go
744cabea103810a2555616fa223ec3bdee2ff9dbed625e6092562bc49800ec8b  docs/README.md
f48dd31319021188f71c33b5c6c5a3d585fdc2b60044a21b8c58d38e4d65f8cc  docs/api.md
9d70c3f723ddf8ee0cdc7cfad1601e20eb529393b682d4511eba441ea1ee03dd  docs/architecture.md
76e5ce338f46d72f28514b9c16af271ce4789f67fc5d13ac3b401f7b9c746c7a  docs/development.md
51d47fb49eee44c4363c1e7dd65225b4cab2a3ba7978345b3592c6e19ac8c6da  docs/requirements.md
1dba415550670385421ea544f6ba82457c0cbc871637bb5a01ce48a3af439361  docs/roadmap.md
ec852912088618e18e5f30284a7085018fb32671f5568f323ba496dd0419cd8f  docs/source/TZ-agent-board-service-v1.0.md
f1ee024ee7e93d6c54dabaf7a60e523c733a5e9a4855ed4b10ad7f9541fcc6c0  go.mod
83418331dd831bcf5abbd9646375e96e00bdfb8427450da8fc089c8ae6c41222  go.sum
8ffe8ba1ba6898ae6b3e2934014f04065e7590ee906de3b213d0f47d261baed6  internal/core/types.go
a8ee5a490f93d9219a0f3b304a1b2de04c94e41a00e16313be4ff7c30f8b8b86  internal/store/attempts.go
31e991211961509490208f2abbe4a20a46631df3ebdf1fa491d44f9e546859a3  internal/store/auth.go
d85db430b4cc84173e35a09d1d6baf156cdd819a5f68b37d0a47f5667afc4aeb  internal/store/dependencies.go
d1ab7291bb546a3c94aa0911332c24974964688cc26a8e76cd85635312432a47  internal/store/epic_gates_test.go
c6811d2995f58376acdfec6da29de1d70fe3b6973f6acc7ea5753752e16fbc8d  internal/store/events.go
1af2c93771d4c2a22b4fe2a763a366b48800cd4628c728cb348d4ed6834ad45e  internal/store/events_test.go
0410e12d548766bfbfd34152184706c5fed81b7e26c91c4d926e4b086280121b  internal/store/evidence.go
716876bfcfccfc1083e6f587b9df2f5b5aa8a293584f0cf383e42c1f8f55763c  internal/store/files.go
bbb8cdca1de13291e485ce720b49d076462b94532ae8b0280e0a07ba9e46c521  internal/store/files_unix_test.go
6a0b04a96a8f6af6868d8b919abb77fe9696810c04783b5af0b3ef5bb8876c9a  internal/store/gates.go
36314a204102ff0a5cb31ea6243a0cd046a524081bf79a837a1eee7bc6a8ce29  internal/store/global_verification_test.go
cf4e09bd4089e9fa46906b336c3624c8e66d1878d39e445a62e80b0e1ce7713f  internal/store/helpers_test.go
1eddcf6006619d8baf99fe7a0384facd02c1ae726e717f43a703de880fd7ec16  internal/store/idempotency_test.go
303811e0b51d5a7fec1d8d12c4ab7857daf71cb75034de5d9627adb24dce0f1b  internal/store/leases_test.go
6a27fbf7bac4aab86f55dd647ebcdb4bde189dd34ab4f074f2c4e2668561ed9c  internal/store/mutation.go
7f0a16ea505dc274bceb67775c016fb791307ca8f41e6ab2385405d0feb2f937  internal/store/persistence.go
3ce94f5dc415040248e044558b83ece5d42363c14c89187fb797175fbea644c2  internal/store/reviews_test.go
87c29d4493bafcd3fe70a5d91f7b845dc4a643bc93a091ec265d2cf98772d360  internal/store/sessions.go
f05a9bbea274114cefbd24fe19eb6565a7faa0c35340780e2380c9f8a78bc8b3  internal/store/store.go
ccc12156e679522cfe2533e40b3f606a782def4623b54484f8aad4663b35b31b  internal/store/tasks.go
d646639e80b8b14b68c2385d9018d6710ef6ed76bbdb46ab8ce7b417f93ea1eb  internal/store/transitions.go
f1bf15239fd8bb6474b131e6852ef87459a28e97ea28115360379da518923ea0  internal/store/workflow_test.go
6a3689502c5df63fe1b397b37e35fea228ffeeecf868d2c6c92523a507bc6ba2  internal/transport/errors.go
c688b58349cf3ee03086cbd9aee07a2cc6c67599380feb7806c1dae13893dde1  internal/transport/http.go
380ce13112322172caeaf239d99e56eb24cafcce042f87fb3db3c171525f829b  internal/transport/http_test.go
d84b622842e31f7086ec16ab53168436b12f41e6e175695c706da2b4b79202b6  internal/transport/mcp.go
a00adb7a12e70a1743e5bd4ac4bcb5ac034167ec40e8459a61b050c0990dc10a  internal/transport/mcp_test.go
bbfe56b23c14e694829adb80e28cda5c4d6f7ea8e9d5719acc29b8bdbecf209c  internal/transport/operations.go
999c73691e8055d3fb94426b7bdac25c3686d6a7998d8b15a59cd59b2773bb93  internal/transport/workflow_test.go
```
