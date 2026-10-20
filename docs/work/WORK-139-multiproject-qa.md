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
