# WORK-167 — schema5 и capable поставка managed agents

22.10.2026. Root, область интеграции: `cmd/agent-board/main.go` и его capability
test, `infra/accord/projects/` guard/snapshot/validate/history/updater, связанные
unit/integration fixtures `tests/infra`. Backend163 отвечает за Go migration,
preflight и restore auth-reset; worker164 за `workerCommand`, web166 за панель.
Общая TASK — только root. Независимое ревью — WORK165.

TASK013 расширяет board до schema5 с managed_metadata/hosts/runs/commands.
Новый независимый permanent floor требует `managed_agents:1` вместе с прежними
registry/ACL/profiles/coordination/deduplication. Capability JSON и рабочая CLI
ветка worker включены в main integration; сами provider процессы здесь не пишутся.

Полный snapshot/restore принимает исторические board2/3/4 и новую5, сохраняет
schema4 migration snapshots с checksum. Schema5 требует managed tables и точную
managed_metadata(1,1); coordination metadata остаётся обязательной также на5.
Updater после stopped-writer/all-project snapshot и guard install требует5
во всех published boards, сравнивает logical hashes всех прежних таблиц и
исторические immutable bytes. Initial apply/DB rollback не используются.

Авторские tests проверяют preservation all-project managed state/history,
fail-closed missing table/metadata/coordination, checksum corruption, mixed4/5
и offline copy перед mocked auth-reset. Последняя проверка не объявляется
реальным reset managed credentials: это отдельная backend/process приёмка.
Guard tests сохраняют прежние обязательные floors и запрещают schema4 binary,
ложные bool/string capabilities и подмену authority marker.

Новые process/E2E ещё не создаются: это отдельный tester после source approve.
Production, реальные БД/credentials/GUI и SSH/Tandem не изменяются. Actual
schema4→5 capable runtime/restore и native Windows — отдельные критерии TASK013.

Авторский `tests/infra` unit/integration прогон: **PASS164/164**. Прежние
ResourceWarning о незакрытых sqlite connections в старых fixtures не меняют
result и не объявляются проверкой реальных host/systemd процессов. Новые
managed-storage cases7, guard/version regressions включены. Root main.go
capability/worker dispatch ждёт остальных author файлов перед Go build.

Root отдельная production сборка Next после author web typecheck/tests85:
**PASS** (Node24.19, Next16.4.2; compiled/static pages/typecheck успешны).
Она не является browser/E2E или утверждением source APPROVE backend/worker.

После исправлений browser clock skew и безопасного принятия нового run version
повторный root `make web-check` на Node24.19: **PASS88/88**, Prettier,
TypeScript и production Next build PASS. Независимый scoped source APPROVE
web/infra получен WORK165; backend/worker ещё проходят review.

Небольшая интеграция root после scoped web APPROVE: backend ограничивает историю
команд по фактическому JSON-размеру и сообщает `commands_truncated`; detail
показывает предупреждение о сокращённой истории, не выдавая её за отсутствие
команд. Полная история остаётся в хранилище. Текст forced stop больше не обещает
ожидание на Windows: там kernel Job Object может завершаться сразу. TypeScript
и Prettier проверяются повторно; scoped delta передан reviewer165.

В `.local/task013-delivery/` подготовлены публичные шаблоны Mac-arm64 и
Windows-x64 с инструкцией, без реальных origin/IDs/tokens/state. Итоговые binaries,
manifest/checksums и архивы создаются только после финальных проверок.

Первый root `make check`: FAIL на двух прежних acceptance ожиданиях schema4,
после успешных package checks (включая worker). Их ожидаемая текущая версия
изменена на5 в `tests/acceptance/backup_process_test.go` и
`migration_process_test.go`; checksum, historical rows и exact replay assertions
сохранены. Новых process сценариев этим изменением не добавлялось. Двухстрочный
fixture delta передан reviewer165; полный повтор требуется после последних
backend recovery fixes.

Полный повтор root `make check` после fixes: **PASS**, exit0. Включены fmt/vet,
все Go packages и существующие process acceptance, полный Go race, сборка,
legacy frontend16/16, infra164/164, web88/88, Prettier/TypeScript и production
Next build. Лог хранится в `.local/task013-delivery/check-final.log` (ignored).
После этого worker уточнил только комментарий про границу Unix process group;
поведение не менялось. Source backend/worker/web/infra и авторские tests получили
независимый APPROVE165 с явной PGID boundary. QA168 gate открыт; новые process/
browser/native проверки ещё выполняются отдельно.

Проверенный source checkpoint локально сохранён commit `65e1ba0`; public candidate
Mac-arm64/Windows-amd64/Linux-amd64 `-trimpath` cross-build PASS. Никаких push/release.
После него process QA обнаружил Darwin EPERM при сигнале группе только из zombie:
root reaper сначала проверяет WNOWAIT+census и не сигналит уже доказанно завершённой
группе. Для live members ошибки сигнала и обязательная проверка exit сохранены.
Reviewer165 отдельно APPROVE runtime delta; actual QA повтор ещё обязателен.

Root также исправляет reconnect после deadman: durable terminal recovery report
отправляется до нового ownership poll, который иначе возвращает expired lease409.
Только locally proven terminal превращается в recovered_stopped; unknown остаётся
unknown. Current generation выбирается до dedup, recovery request стабилен в одной
incarnation, новая incarnation получает новый ID. Recovery ACK не запускает
accepted plan и не продлевает ownership; transient сеть оставляет pending,4xx
останавливает исполнение. Human session refresh отдельный, heartbeat5s. Три новых
unit/HTTP regressions root с race PASS; source/tests APPROVE165 получен. Реальный
offline process сценарий выполняет QA168. Profile snapshot continuity исправляет
author163 отдельно; final candidate/manifest будут перестроены после этих fixes.

Checkpoint fixes `615ddf0`: второй полный root `make check` PASS, включая новые
binding/reconnect unit cases, полный Go race, acceptance и прежний web build.
Фактический QA подтвердил первые9 сценариев, затем нашёл startup UX409 при ещё
действующей прежней host lease. Server fence сохранён. Root добавил ограниченное
ожидание35s только до первого успешного poll и только при отсутствии active/
pending/неоднозначной local работы. Каждый запрос новый;401/403 и любой409 после
успешного получения authority остаются fail-stop. Семь HTTP unit cases проверяют
transient/fresh-owner/living-owner/post-acquire/auth/recovery paths; root и
независимый focused race PASS, source/test APPROVE165. Guide описывает ожидание.
Final QA candidate требуется пересобрать ещё раз; actual сценарии не заменены
одобрением source или промежуточными PASS.

Checkpoint initial acquisition `0b05610`: третий полный root `make check` PASS.
Actual QA v3 browser5/5 и8снимков, capable all-project migration/restore3/3 PASS.
Main process suite итог FAIL: после offline watchdog owned heartbeat прекратился,
но exit proof оказался unknown;8предшествующих PASS не заменяют весь suite.
Author164 добавил fixed allowlisted private diagnostics и параллельный StopAll:
восемь groups не должны последовательно расходовать по grace каждый. Source/
unit/race APPROVE165 получен, причина intermittent unknown ещё расследуется.

Actual native CLI `0.163.0-alpha.11.1` v3 пока FAIL до thread.started: частичный
override enabled=false для host-injected MCP создавал запись без transport.
QA config-only отдельно подтвердил full disabled stdio entries и all-disabled
list; author164 исправляет transport-aware adapter с дополнительной проверкой
получившегося MCP набора. Native permissions/auth не меняются и bypass не вводится.

Author163 после design/source APPROVE165 реализовал managed terminal handoff:
positive exit в той же транзакции закрывает только run/gen/host-owner bound
sessions и текущие task leases, сохраняет fencing/evidence и не выдаёт task done.
Unknown/ordinary EOF ничего не освобождают. Unit/race scope/replay/rollback и
немедленное продолжение singleton/task PASS; actual native-next continuation
требует новой приёмки. Guide дополнен этой семантикой.

В ходе native adapter review165 найден P1: два native config inspections под
общим e.mu блокировали local watchdog других active запусков. Доказано нарушение
локального monotonic deadline; успешный accepted report отдельно продлевает
server lease, поэтому истечение именно server lease этим наблюдением не доказано.
Author164 выносит blocking prep из mutex с повторным authority/state/path check
перед spawn и barrier regression. Combined source approval пока REQUEST CHANGES.

Последующая combined source/test review165: APPROVE. Blocking prep вынесен из
общего mutex; immutable snapshots и повторная проверка state/epoch/deadline/
consent/pins сохранены. Barrier unit доказывает Stop другого active запуска до
завершения blocked inspection и отказ просроченного нового launch. Actual
config-only probe обнаружил literal quoted names в CLI key-path parser;
whole-map TOML override исправлен, независимый probe подтвердил exact managed
connector и все5 остальных disabled, без обращения к модели.

Четвёртый полный root `make check` на итоговой product delta: **PASS**, exit0,
`.local/task013-delivery/check-v4-final.log`: весь Go/race/acceptance, infra164,
web88/format/TypeScript/production Next build. Предварительный v4 check был
намеренно остановлен exit143 при смене adapter source и не считается PASS.
QA fixture/case review и exact candidate runtime остаются отдельным gate.

Actual v5 `3ea1f8b`/Mac SHA7d2864...: три независимых full process **PASS16**,
browser **PASS5**/8PNG и capable updater **PASS3**, pins unchanged. Reviewer165
независимо проверил отчёты, private binaries/DB/journals и screenshots. Прежний
ownership unknown в этих трёх runtime не повторился, его причина не объявляется
доказанной. Native Codex **FAIL** с точным отказом MCP approval policy never:
реальная conversation создана, но Describe/Next не исполнялись, board sessions/
task attempts0. Completed turn не засчитывается как выполнение поручения.

После этого author164 добавил отдельную **default-off** локальную опцию
native_mcp_approved_tools: строгие provider/12имён, scoped exact catalog/policies,
без global approval/sandbox изменений и без remote permission fields. Root
описал контракт в native-agent-permissions.md и guide. Source/tests/docs
APPROVE165; пятый полный root `make check` **PASS**, exit0,
`.local/task013-delivery/check-v6.log`, прежние Go/race/acceptance/infra164/web88/
format/TypeScript/Next build включены. QA flag для явно согласованного native
probe получил отдельный source APPROVE165 перед новым candidate.

Подготовлен конкретный private review artifact с grant только describe/next
для новой synthetic доски/read-only checkout. Человеку задан отдельный вопрос,
поскольку установленный native CLI явно отказал. Ответ ещё не получен;
никакой native grant после отказа не применялся. Windows/второй ПК NOT-RUN.

Exact v6 `798e689efc38761c5b4ec377e59b627edbe04d54`, Mac SHA256
`4177b6f4eefa5d27584e2e1b5ca56a73a1dcfb47014d603bf40275daa9b13334`:
actual default-off process **PASS16**, browser **PASS5**/8PNG, updater **PASS3**,
source/binary unchanged. Go1.27.2 trimpath CGO0 builds Mac-arm64/Windows-amd64/
Linux-amd64 PASS; Windows runtime этим не подтверждается. Own QA services stopped.
Настоящий probe с новым grant остаётся NOT-RUN без ответа человека; сохранённый
v5 native FAIL не заменён fixture результатом.

Curated packages `.local/task013-delivery/Accord-managed-{Mac-arm64,Windows-x64}-798e689.zip`
готовы: binaries точного candidate, инструкция/контракт/TASK/native permissions,
LICENSE, placeholder config без grant, SOURCE и SHA256 checksums. В manifest
явно отмечены candidate/incomplete, native FAIL/NOT-RUN и отсутствие production
поставки. Mac ZIP SHA256 `b2b2b018a4c093d9a84969d5afebf3fce91a1270f222ee3831c799fca4672d12`;
Windows ZIP `ba4084f93e0604a61a3760fcabdb77aae0c2ec8f4176f55e1d57aa45d91dd019`.
Root CRC/checksums/link/default-off/known-private-pattern checks PASS. Reviewer165
независимо сверил все archive bytes/9checksums каждого пакета/headers/templates,
relative links и source/evidence границы: delivery/evidence APPROVE. Это не
разрешение на production или native grant. На момент фиксации ответа человека
нет; все own QA services остановлены, source code frozen.

23.10.2026 получен явный ответ человека: разрешить только Mac synthetic
describe/next + read-only README + exact conversation resume. Global native
настройки, права файлов и реальный GUI/VPS этим не разрешены. Первый authorized
probe подтвердил actual Describe/Next/terminal handoff, но FAIL из-за QA profile
«Use only accord_managed»: модель отказалась применять builtin file read.
Этот FAIL сохранён. Narrow test-only fix разрешает builtin чтение README в
исходном read-only scope, не меняя permissions; review165 APPROVE, Node/diff
checks PASS, test commit cc4b175...; product diff пуст, binary798e/4177... прежний.

Следующий actual native probe **PASS4**:
`.local/task013-qa/run-1792706107727/native-result.json` и wrapper results PASS.
Codex CLI0.163.0-alpha.11.1 реально Describe/Next, прочитал random README marker,
handoff освободил task; exact provider UUID resume получил новую board session
и большую fencing generation той же task, затем второй proven handoff.
Native.cjs SHA0da1396e7bc87552dcdf3f3c9d21cbe69b24eb2936b42ba3e5ee738bd606646f;
product binary/source unchanged, cleanup PASS. Только agreed2 tools, без иных
MCP/native permission changes. Reviewer165 независимо сверил DB/receipts/marker/
sameUUID/clean checkout: evidence APPROVE.

Статус: Mac read-only native acceptance PASS. Windows/второй ПК NOT-RUN,
исторические FAIL сохранены, прежний flaky cause не объявлен доказанным.
Production поставки не было; TASK013 в целом остаётся открытой.

После native PASS подготовлена новая immutable документационная поставка r2;
исходные R1 ZIP не перезаписаны и их SHA повторно совпали. R2 использует те же
798e binaries, обновлённые TASK/SOURCE и сведения о test commit cc4b175.
Mac r2 SHA256 `84dedcf1529aec47e79a6784c3113d6d763c27eae5cef2a79d2eaf67f557f02d`;
Windows r2 `ce7779fadfdde57eacb4b2b0c719a54ec1ee52f89b2f8521c7f323a34c7bf8da`.
Manifest — `.local/task013-delivery/manifest-798e689-r2.json`; folders —
`.local/task013-delivery/r2/{Mac-arm64,Windows-x64}`. Default-off templates и
placeholder параметры сохранены: test grant не перенесён в пакеты/GUI/Windows.
Root проверки и независимое package/doc review165 APPROVE: CRC/все checksums/
archive bytes/headers/links/no private patterns, R1 неизменны. Только отчёты и
QA profile phrase изменились после product798e; broad tests не повторяются
без новой product delta. Все собственные тестовые службы остановлены.
