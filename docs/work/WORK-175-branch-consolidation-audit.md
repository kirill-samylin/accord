# WORK-175 — аудит расходящихся веток перед публикацией

Автор: managed_worker. Основание: новое поручение пользователя — опубликовать
текущий рабочий код и всю Git-историю в новом репозитории, объединив ветки;
экспериментальные функции старых веток не переносить в текущий runtime.

Read-only аудит refs относительно `273ccbb15c99dfac91c37f34d2926abf01ca866f`.
Git refs/index/worktrees/remote не изменялись; native starts/config/service не
изменялись. Эта карточка — author inventory, независимое review выполняет WORK174.

## Топология и минимальное покрытие истории

Ниже `→` означает ancestor, а не рекомендацию применить старый tree:

- config-runtime → config-fixes → config-qa → performance-qa;
- config-fixes → integration-config;
- integration-libs → integration-config;
- config-fixes → performance и performance-fixes → performance-qa;
- backup-budget → backup-qa → backup-method-fixes;
- backup-qa → performance-qa.

Чтобы сделать все запрошенные refs предками объединённой ветки, достаточно
пяти tips: `codex/performance-qa` (`f94c95d`),
`codex/backup-method-fixes` (`c8ce77e`),
`codex/integration-config` (`d1d3fb8`), `codex/native-proxy` (`dc9c74c`) и
`origin/docs/agent-board-spec` (`25341ba`). Остальные перечисленные refs покрыты
их ancestry. Более поздние рабочие TASK009–013 и schema5 должны оставаться
действующим tree. После merge координатор отдельно проверяет ancestry каждого
исходного ref, точный текущий product tree и capabilities.

## Матрица disposition

«Не в273» — количество commits из `273..ref`, включая merge commits; counts
между строками пересекаются. «Файлы» — delta merge-base→ref. Это сравнение
истории, а не перечень новых изменений к применению поверх273.

| Ref | Tip | Не в273 | Файлы | Покрывающий tip | Действующий code disposition |
| --- | --- | ---: | ---: | --- | --- |
| codex/config-runtime | 932aa2a | 2 | 85 | performance-qa / integration-config | История сохранена, configurable D runtime не включать |
| codex/config-fixes | d5a3a12 | 3 | 91 | performance-qa / integration-config | История fix/invariants сохранена, D runtime не включать |
| codex/config-qa | 71d0ba7 | 4 | 107 | performance-qa | Исторические QA/FAIL сохранять, не объявлять current PASS |
| codex/backup-budget | d433718 | 2 | 7 | backup-method-fixes / performance-qa | История bounded retention сохранена, старый single-board wrapper не заменяет project backup |
| codex/backup-qa | 7200cc9 | 3 | 12 | backup-method-fixes / performance-qa | Исторические process fixtures не включать автоматически |
| codex/backup-method-fixes | c8ce77e | 4 | 14 | этот tip | UTC boundary correction сохраняется в истории |
| codex/integration-libs | 59ae1a1 | 1 | 20 | integration-config | Pure E libraries остаются в истории, current runtime не расширяется |
| codex/integration-config | d1d3fb8 | 8 | 122 | этот tip | Trust/portability validators и D prerequisite сохраняются в истории |
| codex/native-proxy | dc9c74c | 1 | 16 | этот tip | Основная security functionality уже есть в273, сохранить более поздний tree |
| codex/performance | b6076ee | 7 | 124 | performance-qa | Исторические implementation/review blockers сохраняются |
| codex/performance-fixes | 73554ca | 7 | 125 | performance-qa | История corrected COW/admin results сохраняется |
| codex/performance-qa | f94c95d | 18 | 163 | этот tip | Старый D/schema4 performance runtime не включать |
| origin/docs/agent-board-spec | 25341ba | 1 | 1 | этот tip | Точный текст уже хранится как docs/source/TZ-agent-board-service-v1.0.md |

## Какие commits и изменения сохраняет история

Config chain: `dad4c5f` persisted configurable roles/workflow; `932aa2a` merge
C catalog base; `d5a3a12` authorization/lease invariant corrections; `71d0ba7`
process QA с зафиксированным concurrent-startup defect. Runtime delta —
`cmd/agent-board/{config,epic,scopes,sweep}.go`, `internal/core/{config,rules}.go`,
`internal/store/{config_bootstrap,config_history,startup_config,runtime_roles,
claim_rules,token_scopes,handoff}.go`, `internal/transport/http_config.go`,
bridge role-expiry и миграции. В273 нет этих runtime CLI/DB APIs; существующий
`internal/config` — offline validator, profiles не равны configurable workflow.

Performance: `b9fdc15` COW snapshots/dirty persistence/indexed events/stats;
`cb5388b`, `b6076ee` independent blocker reports; `96f3bfd` исправляет epic reopen
alias mutation и outward administrative results; `73554ca` source approval;
`f94c95d` combined process QA. Merge commits `30abe16`, `544bd78`, `3620733`,
`4f62533`, `c1ef9ae` соединяют prerequisites. Отдельные файлы snapshots/indexes/
projection/startup_retry в273 отсутствуют; текущий Store не является их
семантическим эквивалентом. Это историческая ветка развития, не текущий PASS.

Backup: `aeb085c` вводит verified-copy inventory, 24h/7d buckets, newest-two floor,
5GiB budget и disk floor; `d433718` review; `7200cc9` real CLI wrapper QA;
`c8ce77e` корректирует UTC-hour/day test oracle. Current legacy single-board
`infra/accord/backup.py` не содержит budget policy. Новые project-wide capable
backup/upgrade guards в273 нельзя заменять старым wrapper из этой ветки.

Integration: `59ae1a1` добавляет pure GitHub HMAC/repository matching/order,
interchange JSONL parse/export и notifications URL/DNS delivery policy libraries.
`d1d3fb8` добавляет explicit repository allowlist, canonical endpoint identities,
NUL identity checks и config/interchange tests. Пакеты `internal/integrations`,
`internal/interchange`, `internal/notifications` и integration config helpers
в273 отсутствуют. Ни историческая pure-library проверка, ни ancestry merge не
включают webhook ingress, OAuth, durable outbox или SQL import/export runtime.

## Уже имеющийся эквивалент

Native proxy `dc9c74c`: 10 из16 изменённых branch файлов совпадают с273 побайтно,
включая `http_host.go`/tests, Host configuration helper/tests, configured-host
CLI tests и service/install baseline. Остальные6 — позднее расширенные main,
HTTP/MCP/browser routing и docs. В273 сохраняются external-host flag, exact
Host authority perimeter и SDK localhost exception только внутри этого perimeter;
новые project/profile/coordination/dedup resources не откатываются. История
branch ещё не ancestor, хотя ключевой код уже перенесён в текущую линию.

Spec `25341ba`: исходный `TZ-agent-board-service.md` имеет SHA256
`ec852912088618e18e5f30284a7085018fb32671f5568f323ba496dd0419cd8f`, точно
совпадающий с273 `docs/source/TZ-agent-board-service-v1.0.md`. Содержимое уже
сохранено, ancestry исходной doc branch требуется объединить отдельно.

## Старые worktrees и незакоммиченные изменения

Read-only `git worktree list --porcelain` + status проверили20 registrations.
Старые config/runtime/fixes/QA, backup/budget/fixes/QA, integration/libs/config,
performance/fixes/QA, web control/service, second-PC source и TASK012 checkout
не содержат tracked изменений. Два `/private/tmp` registrations отсутствуют на
диске: status недоступен; prune/remove не выполнялись.

Единственный untracked source file в старых ветках:
`.local/worker-source-003/tests/acceptance/proxy_process_test.go`.
Он уже есть в273 с теми же четырьмя Test-функциями; единственная diff — ожидаемое
число MCP tools11 в старой копии и12 в текущей. Уникальной незакоммиченной
product functionality здесь не обнаружено; старую копию не stage/overwrite.

Основной checkout содержит текущие doc/operational notes и untracked handoff
files, их сохранение/commit принадлежит координатору. Чистый live release
worktree273 без изменений. Полные path-level branch deltas и worktree inventory
сохранены в игнорируемой `.local/task013-publication-audit/{branches175,
worktrees175}.json`; секреты/credential values для аудита не читались.

## Результат

PASS read-only topology/source inventory, а не merge/build/runtime acceptance.
User-authorized ours/history consolidation допустимо только с явным описанием:
вся история refs сохраняется, действующий code tree остаётся текущим; старые
D/performance/E функции не объявляются интегрированными или проверенными.
Git mutations/push/merge и итоговая публикационная проверка — у координатора.
