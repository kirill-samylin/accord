# agent-board

Общайся с пользователем по-русски. Это самостоятельный сервис координации,
не часть приложения Давинчи. Исходный документ сохранён неизменным в
`docs/source/TZ-agent-board-service-v1.0.md`; актуальные требования —
`docs/requirements.md`, технический контракт — `docs/architecture.md` и
`docs/api.md`, состояние работы — `docs/TASK-001-pilot.md` и последующая
проверка/публикация `docs/TASK-002-publication-readiness.md`; текущее продолжение
до рабочего сервиса — `docs/TASK-003-working-service.md`; веб-версия —
`docs/TASK-004-human-web.md`, текущая панель — `docs/TASK-005-control-web.md`;
предыдущая доработка — `docs/TASK-006-control-refinement.md`, архитектура веба и
масштабирования — `docs/control-web-architecture.md`. Прежний локальный этап после
уточнения пользователя — `docs/TASK-007-next-paperclip.md`: полноценный Next.js
клиент и фактический перенос Paperclip UI, поверх прежнего Go backend.
Развёртывание Next — `docs/TASK-008-next-vps.md`; мультипроектность —
`docs/TASK-009-multi-project.md`, контракт —
`docs/multi-project.md`. Очередь следующих отдельных чатов прямо поручена:
постоянные профили агентов, затем координация, затем управляемый запуск.
После TASK009 запрещён старый Go/rollback БД: сохраняй capable guards/ACL и
полный backup всех проектов; initial TASK009 apply не является updater новых релизов.
Актуальный checkpoint постоянных профилей — `docs/TASK-010-agent-profiles.md`, контракт —
`docs/agent-profiles.md`. Профиль не identity/session/присутствие/worker; immutable revisions/snapshots
и ACL сохраняются. Историческая schema3 TASK010 расширена до schema4 TASK011. Следующие capable поставки —
`infra/accord/projects/upgrade.py`, с отдельным review/QA до production. Текущий этап координации —
`docs/TASK-011-coordination.md`, контракт `docs/coordination.md`; pinned assignment
не lease/identity, schema4/coordination floor обязательны. Управляемый запуск
остаётся следующим отдельным чатом после реального closure TASK011.
TASK012 (дедупликация и экран «Команда») — `docs/TASK-012-dedup-team.md`;
поставлен на VPS release `64ed870b749289b545d29cd2ec7813c6266de54d`, ветка
`codex/dedup` отправлена в origin. Board schema4 сохраняется; новый permanent
floor `deduplication:1` обязателен вместе с coordination/profiles/registry/ACL.
Прежний schema4 binary не понимает новые JSON semantics и запрещён. Следующие
обновления только capable `projects/upgrade.py` с all-project backup/roll-forward.

Текущий явно порученный этап — TASK013 управляемые агенты:
`docs/TASK-013-managed-agents.md`, контракт `docs/managed-agents.md`.
Пользователь выбрал управление обоими ПК обоими участниками после local owner
enrollment/controller allowlist; роль editor сама не даёт OS control.
Рабочая реализация расширяет board до schema5 и floor `managed_agents:1`,
сохраняя registry/ACL/profiles/coordination/deduplication. По прямому поручению
23.10.2026 рабочий VPS обновлён capable updater до clean release
`273ccbb15c99dfac91c37f34d2926abf01ca866f`, schema5/managed_agents1; Mac GUI
enrolled с отдельным user LaunchAgent. Текущая production приёмка — WORK169–172
и TASK013; позднейший WORK195 realMacPASS5 подтверждён169. WORK196 finalized273, journalcomplete и полный backup4проектов/9runtimeDB проверены. Windows/физический второйПК NOT-RUN. Source branch сама по себе не является доказательством установки.
Только capable all-project upgrade/roll-forward, не старый binary/DB rollback.
Native CLI resume использует точный conversation ID; уточнения первой версии
доставляются на следующем turn. Не выдавай queued/offline за applied/stopped,
SDK/stub/cross-build за real native Windows или второй физический ПК.

23.10.2026 пользователь расширил поручение до полного цикла выполнения
требования: из Accord либо Codex/Claude → аналитик → задачи → разработка →
независимое review → QA → исправления/результат, с управлением и вмешательством
человека. Канонический checkpoint — `docs/TASK-014-automation.md`, frozen/APPROVE183
контракт — `docs/automation.md`. Исторический ручной scope TASK011 не запрещает
это прямо порученное расширение. До независимого architecture/security approve
новый workflow не реализуется; registry epochs, host-local consent, leases,
SHA/generation gates, isolation и capable all-project roll-forward сохраняются.
Наличие контракта не означает, что автоматизация или native GUI запуск работают.

24.10.2026 текущий TASK014 server/Next release —
`4ae529272e1542e8473902de26d68349af266d1d`, schema6/floor `automation:1`.
Capable follow-up WORK268 после первой поставки a32: HTTPS270V3 independently
PASS, canonical finalize/final-audit завершены, independent final20228afab67.
Journal complete, новый format2 backup всех4проектов/9runtimeDB/9historicalDB,
точные logical rows и история сохранены. Прежний HTTPS220V6/a32 принят отдельно.
Прежний273 — историческая версия, откат старого binary/БД запрещён.
Mac worker `dfcef4b88f695942d2b0b8dbf59e4e587f45f1d9` установлен отдельно:
Test_internet online/epoch4, четыре профиля и owner-only policy созданы. Реальное
Python-требование завершено и независимо принято WORK261/202: аналитик,
разработчик, отдельные reviewer/tester, один local commit, два JUnit по2passed,
все5terminal ACK/pending0. Первоначальный analyst был forced stopped; после
истечения deadline выполнен явный replan. Это НЕ same-UUID resume proof.
WORK260 exact project trust и WORK263 prompt не расширили другие права/settings.
WORK264 timeout/stable-block установлен в4ae; сроки, бюджеты и права не расширены.
Mac standalone WORK269 установлен Codex-only; catalog/guard/install проверены,
interactive intake NOT-RUN. Windows4ae x64 crossbuild есть, native Windows/Claude/
второй физический ПК NOT-RUN. Mac UI production-контур PASS, полный V1 не заявлен.
Подробности и receipts — TASK014/WORK249/251/258/261/268/270.
По поручению пользователя текущий код и история всех26локальных веток включены
в `main` репозитория `https://github.com/DenisShagi/accord.git`; прежняя история
и MIT LICENSE сохранены. Это поручение разрешает соответствующую публикацию.

## Работа

Продолжение24.10.2026 — `docs/TASK-015-client-intake.md`: настоящий Mac Codex
интерактивно отправил требование через MCP в Test_internet. Цикл completed32,
один local commit2a2a530, две реальные проверки по5passed; UI pause/clarify/resume
того же native UUID подтверждены. Claude Desktop3.6556.2 затем отправил через
accord_test второй workflow wf_fb2f6400fad6a5c326b178f6a357eb8a: completed25,
commitbf77b6d, две реальные проверки по8passed, четыре Codex-роли. Add-only
Desktop MCP применён с backup/readback, прежние настройки/UID/GID0600 сохранены;
обычный Desktop login использован без CLI OAuth. Final QA/review второго пакета
фиксируются в TASK015. Это не новый Windows/Claude execution/второйПК PASS.
Найденный Claude Code2.1.308 CLI не авторизован; provider gate остаётся false.
Native TUI Codex добавил два служебных
поля global config; после закрытия own Test процессов выполнено отдельно
проверенное восстановление точного baseline. Не утверждай bytes-unchanged за
весь запуск. Прямой local5ACK snapshot первого analyst generation не архивирован
до resume; последующий gen2/exit evidence не выдавай за такой snapshot.

- Пользователь 18.10.2026 поручил переработать ТЗ и начать реализацию в отдельном
  каталоге. Переданный затем origin — `https://github.com/kirill-samylin/accord.git`;
  его начальная история и MIT LICENSE сохранены.
- Поручение TASK-002 разрешает отправить проверенную рабочую ветку в origin;
  merge/main, выпуск и переключение сервера отдельно не поручены.
- Последующий TASK-003 разрешает завершение рабочего сервиса и безопасное
  развёртывание Accord на существующем VPS с сохранением прежнего Tandem.
  Пользователь выбрал сначала отдельные токены, GitHub-вход позже;
  физический второй ПК он проверит позже. Не выдавай это за полный V1 PASS.
- Tandem временно не используется по прямому указанию пользователя. Не
  подключайся к доске, не создавай на ней задания и не переноси её настройки.
- Координатор распределяет непересекающиеся области записи; каждый исполнитель
  ведёт свою карточку в `docs/work/`. Общую TASK редактирует только координатор.
- Используй отдельных исполнителей для реализации и независимого ревью.
  Профили ролей в текущей среде передаются путями; модели Claude за ними здесь
  не закреплены. Проверяющий использует ту же доступную модель, что автор.
- До реализации контракта прочитай требования и архитектуру. Не назначай
  несогласованные продуктовые компромиссы молча; перечисли допущения и границы.
- Изменяющие операции транзакционны. Не ослабляй аренды, авторизацию,
  идемпотентность или проверку результата ради прохождения тестов.
- Проверяй поддерживаемые API по официальным источникам. Не обновляй
  зависимости без связи с задачей. Стремись к файлам до 300 строк; исключения
  объясняй в WORK. Комментарии объясняют контракты и ограничения.
- Unit/integration тесты пишет разработчик; новые process/E2E тесты — отдельный
  тестировщик после независимого approve. Новые тесты также проходят ревью.
- Не сохраняй токены и реальные пользовательские данные в код, отчёты или Git.
  Локальные runtime/БД/артефакты держи в игнорируемой `.local/`.
- Сохраняй чужие изменения. Не изменяй соседний проект `davinci-gui-`.
- Git локальный, ветки `codex/…`. Remote, push, PR, merge и выпуск — только
  после соответствующего поручения. Не выдумывай адрес репозитория.
- Различай PASS, FAIL, NOT-RUN и N/A. Два запуска на одном ПК не являются
  проверкой двух физических машин. SDK-клиент не является Claude Code/Codex.

## Завершение

Укажи реализованный объём, изменённые файлы, проверенную версию, фактические
проверки и ограничения. Непройденная обязательная проверка оставляет этап
незавершённым; незавершённый пилот нельзя объявлять готовой первой версией.
