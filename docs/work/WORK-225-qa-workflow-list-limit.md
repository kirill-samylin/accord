# WORK225 — QA-only workflow list page limit

24.10.2026. Root explicit ownership передал managed_qa только новые F6V8/B2V6/
C2V10/H1V7 dirs; старые authors/sourcefreezes/evidence неизменны. **SOURCE READY /
RUNTIME CLOSED**. Основание: actual203V13 достиг4roles completed, затем QA
`api.workflows` ошибочно запросил list limit100. Store.WorkflowList разрешает
максимум50 (`internal/store/automation_list.go:35`); reportlimit100 допустим.
Исторический203FAIL не переименован в PASS, продукт не изменялся.

Во всех4 новыхкопиях единственная behavioral строка API:
`/automation/list {limit:100}`→`{limit:50}`. Report100, pagination/duplicate cursor/
ID guards и все business/workerbe95/servera32/Next/foundation/consent/ACK/SHA/
fixture guards byte-exact предкам. Новые собственные versionedrootGO/source202
идентификаторы, plan note и immutable ancestor source pin; authorGO отсутствует.
Никаких миграций/production/GUI/Testproject/config изменений.

Небольшой одинаковый PURE `test-api-pagination.cjs` исполняет actualAPI source
через isolatedVM с deniedfetch/tokenread/realgate/request helper; constructor
не способен читать credential или проходить permission. Mockrequest фиксирует
route/limit/cursor, проверяет two-page accumulation, duplicate cursor/ID rejection,
emptylist, report100 и runtime-denied constructor. Все6cases толькоsource;
автор не исполнял тест. Независимый202 может отдельно просмотреть/выполнить pure
без browser/native/helper/HTTP. Syntax-only проверка не вызывает код модулей.

| Adapter | Source manifest | Pins |
|---|---|---|
| F6V8 | `540400c62e6761f6b14ea6952573817258bf884dbccc97daab3e625e76a9351f` | `da6923eb233e1465bcac94c091bf3178e5f11a9afcfa756882efe1972d933c57` |
| B2V6 | `f3ce2a4bd4724fd5f8af7013ad158b9b377fe056109691d5ff6182974ae095c5` | `82a62faeb8d71c51a1cf868c91cd6c9d3a8c2470c13b8d266a81c1cde5584986` |
| C2V10 | `d1622a8742f3436f001305dce8a84f8d268cdfdcc4558738a1cf4bb07338bb2e` | `a2ee22a6c4fc4f8801b31562603a4254ddfb1e37936a99087019c34f473eef3c` |
| H1V7 | `797adab4efcb64926db7b8b0ee31821700e5fdd0e5fb62086a5124f742b15d7f` | `2fb5ddb5ce619fd72abbc30cdd42d630217a6542f711257524ccbc52d66cad72` |

Private `.local/task014/work225-source-summary.json` содержит fullSHA/counts.
Static PASS196CJS syntax +12Python AST. Actualmetadata всех24fixturefiles:
regular/no-links, exactSHA и0644==pins420 PASS. Все ancestor maps повторноverified;
обратноеAPI/auth textcomparison доказывает только разрешённые изменения, остальная
sourcebusiness exact. H1V6 не содержал SHA256SUMS; историческийfullhash не выдуман,
новый H1V7fullmanifest создан. Во время draftcopy отсутствие старогоH1fullmanifest
остановило filesystemauthoring; исправлен только newcopy, никакихchild/runtime.

Helper.validate/configread/nativeCLI/browser/HTTP/MCP/models/server/compiler/SSH/
runtime **NOT-RUN**, source202review и отдельныеexactrootGO обязательны. Наличие
источников и mocktests не доказывает native pipeline/control/Windows/второйПК.
