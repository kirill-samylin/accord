# TASK-009 — мультипроектность Accord

20.10.2026. Прямое поручение: подключение репозиториев, переключение проектов,
изоляция данных и прав, сохранение текущего проекта. Ветка `codex/multi-project`.
Статус: COMPLETE. Реализация, независимые review, QA и VPS поставка приняты.
Это завершение TASK009, а не полный V1 или проверка физического второго ПК.
Контракт: [multi-project.md](multi-project.md).

## Реализованный объём

- Каталог доступных проектов, создание с репозиторием, переключение, отдельные
  параметры, архивирование/восстановление без удаления данных.
- Несколько Git HTTPS/SSH ссылок на проект: добавление, изменение ветки/URL,
  удаление связи, версии и безопасный повтор. Credentials в URL не принимаются.
- Участники по стабильным идентификаторам; владелец/редактор/наблюдатель,
  изменение/отзыв доступа, защита последнего владельца и основного проекта.
- Отдельные tasks/sessions/leases/events/actions/notes/design/config и storage.
  Совпадающие T-1, имена участников или репозиториев не объединяют права/данные.
- Явные scoped Next/web/REST/MCP/CLI адреса. Две вкладки независимы: cache,
  drafts, polling, files, команды и retries сохраняют исходный project scope.
- Серверные ACL до чтения/записи/replay; revoke→regrant epochs не оживляют
  старые sessions/actions. Viewer не пишет даже через прямой API или cached action.
- Global secure cookie/logout, явное открытие deep link, Origin/CSRF, fencing,
  leases, independent SHA/generation gates и durable replay сохранены.
- Publish-last provisioning, quarantine/reject recovery, quotas; полный backup
  всех проектов и offline restore/reset только в новое закрытое staging.

Привязка Git означает сохранённую связь, а не clone/проверенные Git credentials
или выполнение кода. Worker execution относится к «Управляемому запуску».
GitHub/OAuth-вход не добавлен: сохраняется ранее выбранная схема токенов.

## Сохранность текущего проекта

Основной board.db, web.db и blobs остались на прежних физических путях.
Registry1 добавлен отдельно; board schema2 сохранена, web1→2 мигрирован.
Старые токены, browser/native sessions, lease generations, history и durable
records не сброшены. Legacy адреса закреплены за default и проверяют его ACL.
Начальный owner назначен явно выбранному прежнему credential ID, не первому
посетителю и не всем прежним токенам.

На VPS сравнение с verified pre-upgrade snapshot выполнено повторно после
финального полного backup: все старые строки девяти board tables, design rows
и blob bytes сохранены. Новые тестовые A/B в production не создавались.
Реальные БД не скачивались, не переносились из fixtures и не откатывались.

## Проверенная версия и поставка

Приложение: `5454eb5c98ac088e5ead3daaaad637423112ddad`.
Go1.27.2 Linux/amd64, clean buildvcs; binary SHA256
`6950b220cd60b9abf3d295f114c60512ae40334480302c8fcf742ab47d61e0a8`.
Go source215 aggregate
`20b896df6eaee22e39410b2ca54eec35a5e9ffe1f720afb38145ee0904662cf5`.
Next BuildID `YYrHpN-gOyr560HLMZaUM`, upstream127.0.0.1:7892;
Next manifest SHA256
`9f3e3e3d3e00cf713b6aa7d58cbb04f1287cfd7950e5e04a880e25fe6554124f`.
Pinned Linux Node24.19.0 и lock/SRI sharp; сборок на VPS не выполнялось.

Исправленные операторы: `865f0a448f99dc1ae4c619184ef396d631f35706`.
Их immutable manifest SHA256
`02a93cd32f6f22d5ebf2242408f0c4397eff6d3b70a9e0c2f6afa7145c49c0ae`.
Приложение5454 не переписывалось. Отдельный reviewed offline amendment изменил
только семь helper entries и audit; исходный journal сохранён O_EXCL0600/fsync,
поля app revision/Go/Next/baseline/owner и реальные БД не менялись.

Рабочая панель открывается по индивидуальной ссылке подключения; адрес не хранится в текущей документации.
Go7892/Next7893/nginx active, NRestarts0; прежние backup/web-backup/renew timers
active. После фактического full backup повторно проверены effective config,
permanent guard, owned PID, /proc argv и ELF SHA. Operator journal phase complete.
Tandem и соседний проект не изменялись, к их API/доске не подключались.

## Фактические проверки

| Проверка | Результат |
| --- | --- |
| Go fmt/vet/all tests/race/build; продуктовый source после amendment не менялся | PASS |
| Frontend format/typecheck,53 unit, production build | PASS |
| Полный make check до операционного amendment | PASS |
| Финальные infra125 unit; из них41 project units, независимый повтор reviewer | PASS |
| Независимые product/author-test/new-test reviews136 | APPROVE |
| QA139 actual process+Chrome, migration/ACL/files/epochs/replay/tabs/late response | 25/25 PASS |
| QA139 дополнительный local process-only | 16/16 PASS |
| VPS Linux actual old490/new5454 synthetic process-only/full restore/reject/guard | 16/16 PASS |
| Hardened Linux Node/sharp/nginx stage; exact owned listeners | PASS |
| Permanent guard как User accord, effective executable/argv/backup, /proc PID/SHA | PASS |
| Внешний HTTPS:11 экранов ×1440/390 ×light/dark,44 screenshots | PASS |
| HTTPS689 responses200, errors/failed assets/prefetch0; auth/cookie/CSP/logout | PASS |
| Scoped native reads/Origin denial, MCP12 tools и scoped resources | PASS |
| Actual full format2 backup, hashes/integrity; post-backup services/guards/data | PASS |
| Физический второй ПК, полноценные workers/providers/load acceptance | NOT-RUN |

Synthetic QA использовал реальный старый490bf55 binary: web1→2, cookie,
нативная сессия, live lease, старый blob и строки сохранены. Проверены A-only,
B-only, shared, viewer; чужие keys/cursors/session/action/assets; concurrent
replay; настоящий незавершённый upload с downgrade; архив с live lease;
startup missing/empty authority failclosed; full snapshot/restore/reset,
quarantine+partial rename; старый executable отвергнут с marker и без него.
Chrome проверил реально committed lost response/exact retry, поздний ответ A,
две вкладки, исправление validation400 и upload1MiB через Next с SHA.

Local full report SHA256
`9452466964fdbb014480f503db0de0165757434cdeaf6a381e78690f9c18adc1`;
Node24.19.0/Chrome155.0.8059.85/Go1.27.2, QA Next `_BtTvE7AHru0IDFWVC3t0`.
Final HTTPS report SHA256
`5aacfc92edba90fce12ca8474f48dc2bc7819890ebe0c4722dacbcbfc73fea2b`.
Full format2 manifest SHA256
`2735e200aa7662f0bb6b4f95e8ee77cc455d0b4b8a5b07898f601909eb8e46fd`.
Все runtime, screenshots, secrets и evidence — ignored `.local/` либо private VPS;
в Git отсутствуют реальные токены и содержимое пользовательских данных.

## Исправленный дефект поставки

Первый apply проверял binary/PID/HTTP200, но прежний web.conf сортировался после
90-projects.conf и возвращал ExecStart без projects-data. Внешняя проверка
обнаружила отсутствие scoped redirect: этот ранний runtime — FAIL, не ACL PASS.
Go/Next остановили, durable START_REQUIRES_INSPECTION закрыл cold boot.

Исправление source-reviewed: zz override; exact resolved systemd executable
и argv для Go/guard/backup, проверка guard path против @argv подмены; actual
owned /proc argv+ELF SHA; permanent ExecStartPre повторяет проверку. Guard
сохраняется во всём install/recovery окне, manual marker не удаляется. Старый90
удаляется только при точном совпадении прежних trusted bytes; web.conf сохранён.
После approved helper amendment выполнен guarded resume и полный HTTPS PASS.

Проверочный сценарий также исправлен: исходная ошибка больше не скрывается
необработанным cleanup wait; report/Chrome cleanup guaranteed, diagnostics
не содержат чувствительных данных. Его10 failure-path checks PASS и независимый review APPROVE.
Однократный generated Python cache в inactive operators удалён только по
строгому reviewed списку, source inventory/hashes восстановлены; дальнейший
root запуск использовал -B. Никакие DB не заменялись ради прохождения проверок.

## Изменённые области и владение

- Backend135: `internal/projects/**`, store/web/transport/CLI scope и migration;
  авторские unit/integration. WORK135.
- Frontend137: `web/src/**`, каталог/участники/репозитории, project scope/cache/
  mutation boundaries, client unit. WORK137.
- Root: `infra/accord/projects/**`, защита legacy backup/restore, release package,
  infra unit, канонические multi-project/requirements/API/architecture/TASK docs.
  WORK138 и Git/SSH/VPS/integration.
- Reviewer136: независимое source/test/evidence review, только WORK136.
- Tester139: `tests/process/task009/**`, `tests/browser/task009/**`, WORK139;
  новые process/E2E только после product APPROVE. Реальные credentials/VPS — root.

## Границы и передача

Репозитории пока через Git URL, без clone/provider credentials/execution.
Изоляция одной службы не является отдельными контейнерами или доказательством
безопасности запуска недоверенного кода. Physical second PC и полный V1
NOT-RUN; Paperclip feature parity не заявляется. Backup остаётся на VPS;
внешнее backup-хранилище не настраивалось.

Старый Go после активации registry запускать нельзя. TASK008 auto-rollback и
повтор initial TASK009 apply не подходят для следующих релизов. Следующему
этапу нужен capable update/roll-forward с полным snapshot, проверкой guards и
сохранением registry/ACL; не откатывать старую БД поверх новых записей.

Пользователь прямо поручил последовательную очередь новых чатов после TASK009:
постоянные профили агентов → координация → управляемый запуск. Новый чат получает
этот checkpoint, контракты и проверенные версии; детальные продуктовые решения
следующих этапов формируются до реализации, не объявляются готовыми заранее.
