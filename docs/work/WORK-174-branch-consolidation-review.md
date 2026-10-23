# WORK174 — независимое ревью объединения веток и публикации

23.10.2026. Reviewer; root выполняет Git consolidation/publication и WORK173.
Reviewer не меняет remote, refs, source или production; область записи — эта
карточка. Пользователь явно поручил новый репозиторий DenisShagi/accord, одну
рабочую ветку и уточнил вариант **«Текущий код + вся история»**. Продолжение
native fix отложено до публикации по новому приоритету пользователя.

## Независимая инвентаризация

База current code — `273ccbb15c99dfac91c37f34d2926abf01ca866f`, плюс честные
текущие docs/checkpoints. До consolidation найдены24 local branches;12 не
являются ancestors273. Ещё old-origin docs/agent-board-spec — отдельный tip.
Remote-tracking HEAD является alias старого main, не дополнительной веткой.

Всю divergent history покрывают пять максимальных tips:

| Tip | SHA | Disposition |
| --- | --- | --- |
| codex/performance-qa | f94c95d19ab7516511e32533f99b1c6430c705b3 | Изолированные D/runtime/performance, config QA, backup QA |
| codex/backup-method-fixes | c8ce77e039c841e0cb0d67b8924950154c3ea495 | Более новый отдельный backup test-method073 |
| codex/integration-config | d1d3fb87d4c6d55895a9b0701dd6ae1112121e75 | E pure libraries/config067, не runtime integration |
| codex/native-proxy | dc9c74cd692e287173def9fb5743dbf18ffdef21 | Host guards фактически перенесены в current tree |
| origin/docs/agent-board-spec | 25341ba5d3890a0273f80ba9cfb66526c1097e3b | Исходный неизменный документ уже сохранён другим путём |

Двенадцать divergent local refs: backup-budget/backup-method-fixes/backup-qa,
config-fixes/config-qa/config-runtime, integration-config/integration-libs,
native-proxy, performance/performance-fixes/performance-qa. Остальные local и
old remote heads уже входят273. Первоначальный быстрый подсчёт11 исправлен на12
после полной программной проверки ancestry.

## Реальное отличие источников

`git cherry` не нашёл patch-equivalent commits divergent branches; это само по
себе не означает отсутствие функциональности. Дополнительно сверены branch
diff от merge-base, tree blobs, текущие код/контракты и исторические WORK/TASK003.

- Native proxy: десять изменённых файлов byte-identical current tree, включая
  http_host.go, host_config.py, unit, host/security/recovery tests. Остальные
  изменённые entry/routing/docs files развиты последующими этапами; host protection
  не потеряна. http_host.go SHA `aceb5f9cf290a2b1846cc0b44605375dd94ff81d6a94683a00abd338b96ec9ca`.
- Старый root `TZ-agent-board-service.md` byte-identical current
  docs/source/TZ-agent-board-service-v1.0.md, SHA
  `ec852912088618e18e5f30284a7085018fb32671f5568f323ba496dd0419cd8f`.
  Дублировать файл в root или подменять актуальные requirements не нужно.
- Config D **не перенесён**: dynamic roles/token scopes, configurable workflow/
  evidence gates, config apply/history и CLI/admin routes, десятки отсутствующих
  current source/test files. Историческая schema3 не совпадает с нынешней
  profiles/schema3 lineage; автоматический импорт недопустим.
- Performance D4 **не перенесён**: COW snapshots, dirty/indexed events/projections,
  decimal statistics и runtime migrations. Историческая schema4 имеет другую
  семантику, чем нынешняя coordination/dedup lineage. Review069/QA072 PASS имели
  свои границы, annual/target VPS/runtime closure не были поставлены/приняты.
- Bounded legacy backup retention и method073 **не перенесены**. В current
  legacy backup.py другая retention policy, а активный full format2 project
  backup — отдельный современный контракт. Старый helper нельзя накладывать
  вместо current capable backup без отдельной адаптации и приёмки.
- E libraries/config067 **не перенесены**: GitHub signature/order primitives,
  JSONL interchange, notifications encoding/policy/sender и explicit pure trust
  config. Исторический WORK067 прямо исключает SQL/schema5/runtime/outbox/live.
  Runtime E не был реализован; отсутствие нельзя объявить регрессией готовой
  runtime integration или утверждать, что она уже работает в current code.

## План

**CONSOLIDATION PLAN APPROVE** для явно выбранного текущего кода со всей историей.
Сохранить current product tree, объединить пять covering tips как history
parents с явным объяснением resolution. Такая операция сохраняет каждую старую
версию через ancestry, но не является feature integration старых экспериментов.
Public disposition manifest обязан прямо перечислить эти границы и SHA.

После consolidation independently проверить: каждый из24 local heads и все
old remote refs — ancestor новой main; product tree cmd/internal/infra/web/schema/
go.mod/go.sum совпадает с reviewed current tree, кроме отдельно review-approved
документальных изменений; MIT LICENSE/начальная история сохранены. Новый target
получает только одну main без force/history rewrite. Старые refs/worktrees и
несогласованные local changes не удалять. Не использовать push --all/--mirror,
которые создали бы несколько рабочих веток или затронули лишние refs.

## Проверка публичности

Read-only scan2758 reachable objects,1899 blobs: bounded Accord-token, GitHub PAT,
OpenAI key, AWS access key и private-key-header patterns —0 findings. Изначальные
неограниченные sk-pattern matches оказались частями слов task-... в тестах;
boundary-corrected scan устранил false positives. Это конкретный pattern scan,
не абсолютная гарантия отсутствия любых секретов.

Pending docs GUI onboarding/WORK158–161/169–172 прочитаны/проверены: точных
значений текущих GUI token files нет; literal remote-IP origins и project IDs
нет. Loopback backend origin в WORK169 — публичный технический пример.
Локальные owner paths в исторических onboarding/WORK158/159 не credential.
Нулевая проверка remote-origin literals здесь относится только к перечисленным
новым/pending docs, а не ко всей истории. Root отдельно подтвердил четыре
исторических committed/public HTTPS ссылки в TASK009–012: это существующие
публичные адреса сервиса, не credentials/private config. Их старые bytes
сохраняются в history; current docs могут получить generic guide references.
Internal refs/codex/app snapshots не являются запрошенными рабочими branches,
могут содержать untracked operational script и исключаются из published ancestry.
WORK169–172 различают deployment activation, actual GUI native preparation
FAIL и pending finalize; Windows/Claude/physical second PC остаются NOT-RUN.

`deploy-task012.command` имеет private operational values/stale workflow и
**не должен публиковаться**; сохранить локально untracked. `.local/`, private
configs/state/DB/archives/tokens/SSH/runtime reports не включать. Старые GUI
onboarding schema4 guidance следует пометить историческим и дать ссылку на
нынешний managed-worker guide/schema5; они не должны стать current capability
инструкцией после publication.

Final consolidation manifest, exact tree/ancestry, выбранный whitelist staged
docs и push ещё **PENDING**. Эта карточка не разрешает silent product changes,
обход native prep failure, finalize или публикацию private runtime evidence.
