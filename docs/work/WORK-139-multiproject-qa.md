# WORK-139 — независимая process/browser QA мультипроектности

Owner: отдельный tester `multiproject_qa139`. Общая TASK, product source,
Go/frontend/infra, Git, SSH/VPS и Tandem не входят в область записи.

20.10.2026. Сейчас только read-only подготовка. Контракт прочитан:
`AGENTS.md`, `docs/TASK-009-multi-project.md`, `docs/multi-project.md`, связанные
requirements/architecture/API, существующие acceptance/browser harness и WIP
CLI/router/reset. Независимый product APPROVE136 ещё не получен; прямого GO
root на написание новых process/E2E и запуск runtime нет.

## Предлагаемые файлы после GO

- `tests/process/task009/**`: собственный synthetic harness, process ACL/native
  сценарии, migration/backup/restore lifecycle. Реальные CLI child, net/http,
  stdio MCP и SQLite; прежние тесты и product source не менять.
- `tests/browser/task009/**`: Chrome/Playwright acceptance актуального Next
  production build, отдельно сценарии переключения/вкладок и прав/настроек.
- `.local/task009-qa/**`: private fixtures, токены, БД, отчёты и снимки экрана.
  Файлы тестов стремятся к ≤300 строк, общие helpers разделяются по обязанности.

## Последовательность и обязательные проверки

1. Зафиксировать actual binary SHA, source revision/dirty status, test file SHA,
   Node/Go/Chrome versions и Next BuildID. Все state dirs0700/files0600;
   child output не пересылать, credentials не включать в args/отчёты/скриншоты.
   Останавливать только принадлежащие fixture PID, cleanup проверять явно.
2. Через прежний single-board режим/доступный verified old binary создать
   default с native token/session, browser cookie и synthetic данными. Offline
   web schema1 fixture соответствует каноническому прежнему DDL. До запуска
   multi-project сохранить точные логические task/event/replay/session строки,
   blob hashes и cookie value. После миграции сравнить сохранность, default
   binding/ID/физические пути и web1→2; GET не должен делать Describe/renew.
   Повторный restart не создаёт новый default и не меняет эти данные сам.
3. Создать проекты A/B с безопасными repository URLs. Проверить repo CRUD,
   повтор одного action ID (включая concurrent POST), payload/version conflict,
   неприемлемые protocols/credentials/query/fragment/local paths. Сохранённая
   связь не считается clone или проверкой удалённого Git-доступа.
4. Отдельные A-only, B-only и shared identities с owner/editor/viewer. B-only
   login/logout и пустой catalog не должны требовать default membership.
   Legacy aliases закреплены за default и проходят ACL. Last owner, archive
   с live lease, archived read/write и restore проверяются серверными запросами.
5. Через реальные scoped REST и MCP/stdio проверить одинаковые T-1/A/B,
   чужие keys/session IDs/cursors/assets/resources/action IDs/CSRF, отсутствие
   сведений о скрытом проекте и одинаковое404 для hidden/unknown ID. Viewer
   не получает write/singleton/heartbeat права. Core fencing/SHA gates остаются.
6. Revoke/downgrade до повторного/cached действия; затем regrant с тем же
   local principal: старые native sessions и выданные browser/catalog actions
   не оживают. Новый явный вход/Describe работает. Другой разрешённый проект
   не теряется. Global logout tombstone не даёт переимпортировать old cookie.
7. Chrome: catalog/create/open/deep409→explicit POST, reload/back, две вкладки
   A/B, sidebar/commands/files/forms/polling, catalog detail A→B, viewer/archive
   интерфейс. Фактический поздний ответ A удерживается/доставляется после switch
   B без подмены его тела; кэш/формы B не меняются. Captured mutation/retry не
   перенацеливается. Definitive validation400 не становится unknown-result lock.
   Проверить console/page errors, horizontal overflow, desktop/mobile и light/dark.
8. После остановки единственного fixture writer выполнить reviewed format2
   snapshot; проверить оба проекта, готовые blob bytes и unpublished raw pending
   state. Restore только в NEW offline staging через capable CLI reset. Проверить
   reset credentials/native/browser/actions/leases, pending quarantine без
   публикации и сохранность source snapshot; новые credentials/explicit owner
   recovery дают доступ к сохранённым данным. Неполный WAL/hot journal, tamper,
   symlink и неучтённые файлы должны отклоняться, last-good копия сохраняется.
9. Capability helper должен отвергнуть actual old binary и при наличии registry
   без marker. На fixture guard refusal не запускает server и не меняет данные.
   Старый Go не запускается в обход helper поверх реестра с новыми ACL.

## Границы evidence

Process/browser/migration/backup/restore/downgrade — **NOT-RUN** до GO.
Авторские unit/build PASS исполнителей не подменяют эти проверки.
Production HTTPS, Linux systemd/nginx/VPS и физический второй ПК здесь NOT-RUN;
это отдельные проверки владельца поставки. Paperclip feature parity не заявляется.
Новые тесты и evidence после реализации передаются независимому reviewer.

## Разрешение и подготовленные сценарии

Root выдал GO после независимого product source/author-test APPROVE136.
Созданы новые `tests/process/task009/**` и `tests/browser/task009/**`;
syntax всех12 CJS/Python файлов PASS. Runtime пока не запускался: последующий
узкий startup authority-loss P1 повторно открыт владельцами135/136, tester
ждёт исправленный binary и отдельный root freeze. Ни одного app PID не создано.

Дополнительно по прямому поручению root подготовлен generic
`production-readonly.cjs`. Он получает только пути приватных config/token/output
через env; не запускается tester на VPS и не получает реальные credentials.
Разрешены лишь собственные login/logout/explicit project-open, GET и MCP
inspection без Describe/lease/task/repository/ACL/upload. Неожиданные browser
мутации блокируются до сети. Отчёт агрегированный, screenshots private.

WIP review136 уже исправил корректность harness: preflight3100/3101 до HTTP,
readiness по stdout своего child, private root ancestry/UID, HTTP deadlines,
native merged headers и явное различение MCP server denial от timeout/parser
failure. Negative проверки не засчитывают transport errors как ACL PASS.
Fixture SSH URLs используют explicit `ssh://git@host/path` по принятому контракту.
Все новые файлы менее300строк. Полный test/evidence APPROVE ещё NOT-RUN.

## Итог actual local QA — freeze

Полный run **PASS25/25**, exit0: private
`.local/task009-qa/run-1792500343500/results.json`, SHA256
`9452466964fdbb014480f503db0de0165757434cdeaf6a381e78690f9c18adc1`.
Фактическая среда: darwin/arm64, Go1.27.2 (`go version -m`), Node24.19.0,
Chrome155.0.8059.85, Next BuildID `_BtTvE7AHru0IDFWVC3t0` с upstream3101.
Binary SHA256 `3946080cacf567a8dc2456cfe1f240e05d42846f4cac72993766c4b77cabd396`;
реально запущенный legacy binary из490bf55 SHA256
`151929bea531faf0ef3b7b5da9ee1c0fb605714f08b06ba92829290dfe55dd64`.

Root подтвердил product commit `d9030ab`, неизменённый approved Go215 aggregate
`20b896df6eaee22e39410b2ca54eec35a5e9ffe1f720afb38145ee0904662cf5`.
На момент run dirty только QA/docs; full revision/build source manifest
фиксирует root отдельно. Binary собран до этого commit; фактический embedded
buildinfo сохранён в private `go-buildinfo.txt` SHA256
`4be76574b91ba987b8218d1edc812df001cac856f8fbbeec02ad3b8d541957b6`.
Tester Git не выполнял и не заменяет root provenance догадками.

Проверены реальные legacy web1→2, token/cookie/native human session/live lease,
точные task/event/replay/session строки и старый default blob/upload/download;
новые isolated projects, repoCRUD/URL validation/version/conflicting payload и
concurrent same-action replay; A-only/B-only/shared/viewer, last owner,
глобальный logout/tombstone, Origin/CSRF; REST и stdio MCP, collisions task keys,
чужие session/cursor/resource/file; downgrade/revoke/regrant для native и
unused/cached browser/catalog capabilities; незавершённый настоящий upload не
держит ACL guard, viewer downgrade блокирует final publish и убирает reservation.

Chrome покрывает вход/deep explicit POST/two tabs/reload/switch/back/cache,
создание проекта с репозиторием, archive/read-only owner/restore, catalog drafts
A→B, validation400→correction, фактически committed lost response→exact retry,
actual held A dashboard read с явным network completion/cancellation до assert B,
scoped Next upload1MiB с native size/SHA, viewer UI, desktop/mobile/light/dark.
Шесть финальных screenshots сняты с `animations: 'disabled'`; overflow, page
errors и unsolicited prefetch отсутствуют. Historical первый dark screenshot
попал в100ms transition; readonly computed-style diagnostic подтвердил enabled
buttons opacity1 и правильные settled цвета. Продуктовый CSS не изменялся.

Actual process отказался стартовать при marker+missing/empty registry, сохранив
default. Полный format2 snapshot восстановлен только в новый offline staging:
registry+4published boards+unpublished raw pending; credentials/browser/native
sessions/actions/leases reset; old cookie/token rejected, explicit fresh owner
recovery читает старые tasks/blob bytes. Pending board.db+journal hashes
сохранены в restored и ordinary reject quarantine; partial rename не оживляет
старую HMAC create capability. Настоящий legacy executable отвергнут capability
guard и при потерянном marker. Live production paths не затрагивались.

Process-only override отдельно **PASS16/16** на macOS без Next/Chrome:
`.local/task009-qa/run-1792500278083/results.json`. Linux/VPS/HTTPS остаются
**NOT-RUN у tester**; root выполнит точный process-only код на Linux.
Инструкция в `tests/process/task009/README.md`, включая OLD_BINARY override,
необязательный Go tool и явно помеченный operator-build-provenance fallback.

Два ранних запуска завершились harness/fixture FAIL: NextCLI штатный143 на
SIGTERM и asynchronous deep-open button race; затем неверное ожидание сохранения
stale journal после запроса SQLite recovery. Исправлены в tests; это не скрытые
product PASS. Subsequent полные25 и process-only16 прошли.

После полного run единственная CJS delta — platform-aware текст limitations
для Linux вместо hardcoded macOS. Продуктовые сценарии не меняются; root/136
согласовали metadata-only review без повторного Chrome. Final source manifest
в `.local/task009-qa/final-freeze.json`, aggregate всех15 code files+README:
`e9df458b0733cc430b1b021d7d3cf5748f1d822d3826cb08c35a31a3b8307590`.
На окончательном freeze оба owned ports3100/3101 **FREE**, Go/Next/MCP остановлены,
Chrome закрыт. Root может использовать binary/dist для выпуска. Независимое
окончательное test/evidence review136 запрошено; product source tester не менял.
