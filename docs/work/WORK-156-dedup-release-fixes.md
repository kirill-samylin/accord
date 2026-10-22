# WORK-156 — исправления перед поставкой TASK012

22.10.2026. Автор исправлений, отдельно от reviewer WORK153 и координатора
WORK154. База `4ec5ccaeec929136d4aa900869d54b1f8a6c614a`, ветка `codex/dedup`.
Владение согласовано с координатором: `.gitignore`, CLI capabilities в
`cmd/agent-board/main.go` и `projects_test.go`, `infra/accord/projects/guard.py`,
`tests/infra/test_projects_guard.py`, store `dependencies.go`, `team.go`,
`transitions.go`, `duplicates.go` и новый `dedup_release_test.go`; только этот
WORK. Общую TASK и README ведёт координатор. Чужой `deploy-task012.command`
не изменён. Commit/remote/VPS этим исполнителем не выполнялись.

Позднее координатор расширил владение на
`web/src/features/mutations/{useAction.ts,CreateTaskDialog.tsx}` для strict409
recovery. Прочитан и применён React skill
`/Users/shagi/.codex/skills/vercel-react-best-practices/SKILL.md`:
consent сбрасывается в обработчике изменения формы, без синхронизирующего effect.

## Изменения

- Добавлен capability `deduplication:1`; permanent guard требует точное целое1
  вместе с прежними registry/ACL/profiles/coordination/schema4 floor. Старый
  schema4 binary не понимает persistent duplicate relations/override/provenance
  и при повторной сериализации может потерять поля, поэтому до запуска нужен
  именно semantic floor. DDL/schema bump не нужен: board остаётся4. Stage,
  apply, backup, restore и systemd уже используют общий `require_capable`.
  Guard не меняет bytes/UID/mode authority marker. Установка нового helper до
  storage mutations остаётся обязанностью reviewed capable updater; rollback
  БД и прежний binary не применялись.
- `/web/.next/` игнорируется. Единственный tracked generated cache
  `_events_20421_9cc41a7240a4bfe6.json` удалён из индекса, локальный файл сохранён;
  содержимое не печаталось.
- «Это разные» поддерживает `related` между двумя epic с пустой condition и
  generation0 через прежний administrative gate. Прочие epic dependency types,
  mixed task/epic, conditions и generation по-прежнему запрещены. UI не менялся.
- Восстановление cancelled epic→open удаляет только исходящие `duplicate_of`,
  как task→ready; прочие связи сохраняются. Неподтверждённое восстановление с
  отсутствующей причиной не меняет связи. Повторный mark/restore проверен.
- Свободные задачи команды используют общий `assignmentAvailable`: stale или
  archived profile не попадает в список до явного reassign/clear.
- Claim advice соблюдает duplicate mode off; отдельный advisory area_overlap
  сохраняется. Фильтр живых leases применяется до ranking/top5, чтобы пять
  свободных задач не скрывали настоящего holder. Оба списка ограничены5.
- Подтверждённый сервером `DUPLICATE_SUSPECTED` открывает explicit checkbox и
  причину в том же диалоге, даже если preview был пустым/устарел/не загрузился.
  `useAction` передаёт код ошибки явно, без разбора message. Любая правка draft
  снимает consent/причину/server code; успешное создание тоже снимает consent.
  Unknown result сохраняет frozen pending action и разрешает только exact retry.

Store/CLI/infra source/test файлы меньше300 строк; зависимости не менялись.
Существующий `CreateTaskDialog.tsx` уже превышал300 строк. Он получил узкие
изменения recovery/consent без перестройки всей формы перед release; поэтому
выделение компонентов оставлено вне исправления race strict409.

## Авторские проверки

- `make test`: **PASS**, включая прежний acceptance suite и новые unit/store
  integration regressions. Новые process/E2E этим автором не писались.
- Infrastructure suite: **PASS157/157**, включая старый schema4/no-capability,
  invalid floor types/versions и сохранение bytes/UID/mode/inode/timestamps marker.
- `git diff --check`, `.next` ignore: **PASS**.
- `make race vet`: **PASS**, все Go packages с race detector и vet.
- Next TypeScript `--noEmit`, клиентские тесты72/72, prettier для двух изменённых
  файлов: **PASS**. Полный `next build` выполняет координатор в release checks.

Две первые итерации `make test` завершились **FAIL** только из-за неверных
ожидаемых wire codes в новом тесте: внутренние forbidden/invalid_input
отображаются в FORBIDDEN_ROLE/VALIDATION. Assertion исправлены по core.Fail;
продуктовые проверки и ACL ради теста не ослаблялись.

Независимое review, новый process/browser QA, Linux и production: **NOT-RUN**
этим автором; их выполняют отдельные роли до разрешённой поставки.
