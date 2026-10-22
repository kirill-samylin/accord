# WORK-153 — независимое ревью дедупликации и «Команды»

22.10.2026. Reviewer, та же доступная модель, отдельный от автора реализации.
Область записи: только этот WORK. Исходный кандидат `4ec5ccaeec929136d4aa900869d54b1f8a6c614a`
на `codex/dedup`, baseline `cebcbe5`. Managed launch не реализовывался и WORK
управляемого запуска не создавался: TASK012 занята отдельным поручением dedup/team.

Статус исходного кандидата: **REQUEST CHANGES**. Ниже перечислены findings,
переданные координатору и отдельному автору исправлений до source APPROVE.
Production не изменялся; `deploy-task012.command` не запускался.

## Источники и метод

Прочитаны AGENTS, TASK009/010/011/012, требования, архитектура, API,
multi-project, profiles, coordination, control-web-architecture. Проверены
фактические changed source/test chains core/store/projects/transport/bridge/web/
Next, а не только описание TASK012. Для React/Next применён
`/Users/shagi/.codex/skills/vercel-react-best-practices/SKILL.md`: границы
запросов/отмены, ключи project scope, входящие payload и mutation ownership.

## Findings исходного 4ec5cca

1. **P1 — нет capability floor новых durable semantics.**
   `cmd/agent-board/main.go:45`, `infra/accord/projects/guard.py:42–54` по-прежнему
   разрешают прежний schema4/coordination1 executable. Новые `duplicate_override`
   и creator fields сохраняются в JSON, а `duplicate_of` меняет смысл состояния.
   Старый executable не знает этих полей; `loadEntities` игнорирует unknown JSON,
   `saveBoard/saveEntities` переписывают все tasks даже при обычном transact/read.
   Поэтому допустимый старый binary теряет новые поля и semantics. Нужен явный
   постоянный `deduplication:1` floor до активации, backup/restore/update guards
   и проверка отрицания старого executable. DDL/schema bump сам по себе не нужен.

2. **P2 — «Это разные» для эпиков всегда отклоняется.**
   `internal/store/dependencies.go:48`,
   `web/src/features/duplicates/duplicate-pairs.tsx:99–110`:
   pairs включают epic, UI посылает `related`, а Store.Link разрешает связанные
   задачи только при Type=task. Минимальный fix: explicit epic→epic related с
   пустым condition/generation0 под прежним administrative gate, без расширения
   blocks/contract_of или межтиповых связей. Нужен regression фактического dismissal.

3. **P2 — восстановление эпика сохраняет duplicate_of.**
   `internal/store/transitions.go:21–22,79–86,154–166`:
   новая очистка связи находится только в task→ready ветке. Epic cancelled→open
   делегируется отдельно, получает open со старой duplicate link/баннером и
   перестаёт участвовать в поиске пары. Повторный mark может добавить вторую
   outgoing duplicate link. Нужна общая detach semantics при восстановлении
   задачи и эпика, с сохранением прочих links/history и regression.

4. **P2 — «Ждут исполнителя» включает заведомо невыдаваемые назначения.**
   `internal/store/team.go:107–112` проверяет ready/unleased/dependencies/open epic,
   но не `assignmentAvailable`. После edit/archive профиля Task011 запрещает
   любой новый claim. Team показывает её как готовую и увеличивает total.
   Использовать тот же predicate, что scheduler, не отдельную ослабленную копию.

5. **P2 — claim warnings игнорируют off и могут пропустить живого исполнителя.**
   `internal/store/duplicates.go:238–248`: mode отброшен, потому `next` выдаёт
   duplicate_check при off. Кроме того, similarTo сначала выбирает top5 всех
   candidates, и только затем фильтруется LiveLease. Пять более ранних/сильных
   ready/recent done скрывают шестую уже выполняемую похожую работу. Фильтр live
   должен идти до limit; area_overlap остаётся отдельным предупреждением.

6. **Generated cache в Git.**
   `web/.next/cache/_events_20421_9cc41a7240a4bfe6.json` — 216-byte JSON array,
   исследован только по shape/названиям ключей без вывода значений. Это generated
   telemetry artifact, а не source. Нужны удаление tracked cache и ignore `.next/`;
   наличие секрета или реальных пользовательских данных по этой проверке не заявляется.

## Поставка и достаточность проверок

`deploy-task012.command` в исходном виде не является approved delivery gate:
push идёт первым (строки14–15), отсутствуют полный make check/source APPROVE/
независимая QA/exact Linux update rehearsal до apply. Комментарий строки4 про
rollback противоречит принятому roll-forward контракту. `set -e` лишь прекращает
следующие команды: он не доказывает откат, сохранность, исправное hold/recovery
или возможность безопасного повторного apply. Удаление release dirs строкой27
не является допустимым восстановлением неизвестного частичного результата.
Сценарий не запускался и не предлагается пользователю для запуска. Использовать
reviewed capable `upgrade.py` stage/apply/resume/finalize после отдельных барьеров.

Заявленные автором unit/race/tsc/esbuild и Chromium с заглушками next/* полезны,
но не доказывают standalone Next build/runtime и package provenance. Независимая
product/browser/update QA TASK012, exact Linux и реальная production поставка
пока NOT-RUN; прошлые TASK011 PASS не переименовываются в TASK012 PASS.

## Положительные свойства

Create check выполняется внутри прежней atomic mutation; ответ warnings
сохраняется в durable replay. Mark duplicate сохраняет live lease и проверяет
administrative права, состояние и цикл; восстановление не стирает audit.
Новые project service reads/writes идут через текущие ACL/epochs и session guards.
MCP остаётся12tools, ресурсы требуют собственную board session. React выводит
пользовательский текст через экранирование; project links/queries сохраняют scope.
QueryBoundary remount по routeScope защищает новое useSimilar placeholderData от
переноса между проектами. Team footnote честно отделяет session от OS process.

## Следующая проверка

Автор исправлений отдельно закрывает findings; reviewer перечитывает diff и
regressions, повторяет подходящие unit/integration. Только после source APPROVE
отдельный tester пишет/запускает process/E2E; новые тесты также проходят review.

## Повтор source — core и guard

WORK156 исправил пять core findings без ослабления scheduler/auth. Прочитан
полный diff и `internal/store/dedup_release_test.go`: epic related ограничен
epic↔epic/empty condition/generation0 и прежними правами; restore убирает только
собственную outgoing duplicate link; Team использует `assignmentAvailable`;
live candidates фильтруются до limit5, off сохраняет только bounded area advice.
`deduplication:1` обязателен для нового permanent guard и candidate capabilities.
Restore/backup/stage используют тот же guard, upgrade ставит helper под owned
startup hold до активации. Служебный authority marker bytes/UID/mode не меняются.
Tracked cache удалён из индекса, `.next/` добавлен в ignore.

Reviewer независимо повторил store/web/transport/bridge/similar — PASS;
projects/cmd — PASS; guard/ingress unit9/9 — PASS. История двух первых запусков
projects/cmd — FAIL в fixture setup `unsafe web data parent`: штатный Mac TMPDIR
идёт через symlink `/var`; один лишь umask077 не помог. Повтор с существующим
repo-private realpath `.local/toolchains/tmp` прошёл. Source storage guard
не менялся, этот результат не является FAIL продуктовой дедупликации.

Core/guard source slice **APPROVE**. Общий SOURCE APPROVE пока ожидает узкий
frontend fix восстановления strict create после authoritative concurrent
`DUPLICATE_SUSPECTED`, когда preview не показал candidates. Неизвестный transport
outcome должен по-прежнему замораживать исходную отправку; это отдельный случай
от известного отклонения сервером. Runtime QA/поставка по-прежнему NOT-RUN.

## Общий SOURCE APPROVE

Повторный frontend review: `useAction.errorCode` задаётся только для known
rejection с освобождённым pending, очищается при dispatch/success; unknown
outcome оставляет frozen retry/locked fields. `CreateTaskDialog` принимает
authoritative DUPLICATE_SUSPECTED независимо от preview, требует explicit consent
и причину, сбрасывает согласие при правке draft. После следующей validation
ошибки уже выбранный force сохраняет доступную форму причины и checkbox;
нет скрытого зависшего consent. Закрытие диалога unmounts весь draft.

Reviewer team/actionState10/10 PASS на Node24.19.0; existing module-type warning
не является FAIL, зависимости/настройки package ради его подавления не менялись.
Автор156 web72/types/format PASS. Реальный concurrent strict browser scenario,
UTF-8 reason validation и uncertain exact retry — обязательные отдельные QA
сценарии после этого source approval, не объявляются проверенными по чтению кода.

**SOURCE + AUTHOR TESTS APPROVE** после исправлений WORK156, mandatory source
findings закрыты. Product/author-test manifest:69files, SHA256
`7b55514bb25f43a13aec24c4d71664d89f771ee272cd55613f347e0eec756290`.
Baseline `cebcbe5b1955ffc95fb74aabc90c2233d6c83ae3`; набор — union
`git diff --name-only BASELINE` и `git ls-files --others --exclude-standard`,
отфильтрованный на `.gitignore`, `cmd/`, `internal/`, `infra/`, `tests/infra/`,
`web/src/`. Manifest — sorted строки `sha256 + two spaces + relative path + LF`,
для отсутствующего пути `DELETED + two spaces + path + LF`. Docs и пользовательский
deploy script в этот product manifest не входят. Cache deletion и ignore
проверены отдельно. UI file SHA256: useAction
`0b6c5fff335c9dfd5469e8aefae2e042830e1b6ec5102aa2cd15a0ca0990ba64`,
CreateTaskDialog
`09ad8ab921851b16fbaaad948751d3fee44de78a92b5abd32faf5ac937e25187`.

Разрешён следующий барьер: отдельный tester process/browser TASK012 и review
новых тестов. Root полный final make check, immutable Linux/update/restore,
actual stage/guard replacement, production full backup/HTTPS/preservation и
итоговый evidence review остаются отдельными обязательными барьерами.
Это не FINAL APPROVE и не разрешение запускать исходный deploy-task012.command.
