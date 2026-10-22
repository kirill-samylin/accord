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

Статус: implementation готова к source review165; actual process migration и
полная интеграция пока NOT-RUN. Никакой production поставки не было.
