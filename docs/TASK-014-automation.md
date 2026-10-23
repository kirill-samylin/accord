# TASK-014 — выполнение требования командой агентов

23.10.2026. Координатор: root. Исходная опубликованная версия — `26d833f`;
рабочая ветка — `codex/native-preflight`. Это продолжение прямого поручения
пользователя: ввод требования из Accord либо Codex/Claude, автоматическое
создание и выполнение задач аналитиком, разработчиком, независимым ревьюером
и тестировщиком, управление ходом работы обоими участниками.

## Результат и границы

Одна отправка требования запускает наблюдаемый durable workflow:
анализ → план задач → реализация → review точного результата → QA →
исправление с повторными проверками либо итоговый отчёт. Человек может
приостановить, отменить, продолжить, уточнить требование и ответить на вопрос.
Вход из приложения и native MCP использует одинаковые правила и сущности.
Отдельные ручные managed runs остаются доступными.

Первый режим исполнения — последовательно в разрешённом checkout. Это явный
default, предотвращающий конкурирующих писателей; параллельные изменения
требуют отдельных workspaces и отдельного контракта интеграции результатов.
Workflow не получает права на ОС из project editor/owner: используются
локальное enrollment, controller allowlist и отдельное ограниченное разрешение
автоматизации. Подключение третьего участника к сайту не подключает его ПК.
Секреты native provider остаются на компьютере владельца.

Контракт — [automation.md](automation.md), после независимого review.
Исторические ограничения TASK011 о ручном цикле расширены этим поручением;
lease, access epoch, idempotency, независимость проверок, SHA/generation gates
и capable all-project обновления сохраняются. История и данные проектов не
подменяются демонстрационной базой. Изменения соседнего GUI не входят в этап.

## Состояние

**IN PROGRESS / NOT READY.** Контракт181 frozen и получил APPROVE183; автоматический
координатор в установленной версии отсутствует. В последнем проверенном snapshot Test_internet
вход и membership PASS, hosts/runs/profiles — 0; новый worker ещё не enrolled. На Mac GUI enrolled
worker; первый реальный background native запуск завершился до conversation
на isolated MCP preparation. Это был открытый prerequisite; позднейший195 real native GUI PASS5 и169 EVIDENCE APPROVE закрыли Mac gate. WORK196 finalized273: journalcomplete и all-project backup4проектов/9runtimeDB подтверждены. Новая автоматизация ещё не установлена.

| Работа | Владелец | Состояние |
| --- | --- | --- |
| Диагностика native preparation, авторские unit | managed_worker / WORK176 | SOURCE APPROVE; установлена62e в WORK182 |
| Реальный config-only Go process probe | managed_qa / WORK177 | 12 фаз PASS; independent evidence APPROVE |
| Установка diagnostic worker Mac | root / WORK182 | PASS; native retry FAIL: discovery timeout |
| Полный automation контракт | managed_worker / WORK181 | frozen, architecture/security APPROVE183 |
| Независимое architecture/security review | managed_review / WORK183 | APPROVE; не source/runtime приёмка |
| Core/store реализация и авторские tests | managed_worker / WORK185 | SOURCE APPROVE65; bounded PolicyGet/32checks/CheckAborted/terminal epoch/revoke bound |
| Registry/transport/bridge интеграция | root / WORK186 | SOURCE APPROVE31; actual Store→HTTP terminal ACK/lock/resume regression PASS |
| Browser/Next UI и author tests | automation_web / WORK187 | SOURCE APPROVE45; selected-scope/CheckAborted UI;103frontend/tsc/Nextbuild PASS |
| Capable schema6 upgrade/restore | automation_infra / WORK191 | SOURCE APPROVE191;54 infra units и targeted race PASS; Linux process QA впереди |
| Local worker/checks/Git proof/IPC | automation_worker_impl / WORK193 | SOURCE APPROVE97;4P1+regressions, worker/IPC/cmd race и crossbuild PASS |
| Native foundation retry | managed_qa / WORK195 | realMacPASS5 +169EVIDENCE APPROVE; root196 finalize273 complete |
| Существующие schema6 acceptance | managed_qa / WORK197 | SOURCE APPROVE189; оба actual process tests PASS8.649s |
| Новая source/QA/production приёмка | отдельные owners после кода | composite SOURCE APPROVE189; final checks и новые runtime QA впереди |

Root единолично редактирует эту TASK. Исполнители ведут собственные WORK и
получают непересекающиеся области записи. Process/E2E пишет отдельный QA
после независимого approve соответствующего кода и самих новых проб.

## Приёмка

1. Рабочий native start/stop/exact conversation resume/clarify на Mac;
   interrupted/offline/ownership uncertainty не выдаются за остановку.
2. Требование из Accord и scoped MCP создаёт один workflow при replay;
   подмена проекта/policy/host/profile, устаревшие и отозванные grants запрещены.
3. Реальный аналитик формирует валидированный план; задачи получают leases.
   CLI exit 0 без предметного результата не переводит этап в успех.
4. Реальная разработка проверяется отдельными review и QA с актуальными
   commit/generation и доказательствами выполненных проверок.
5. Отрицательная проверка вызывает ограниченный цикл исправлений; вопросы,
   исчерпание лимитов и неопределённость требуют видимого вмешательства.
6. Pause/cancel/resume/clarify сохраняют историю, переживают restart и
   потерянные ответы, не создают конкурирующих writers/дубли процессов.
7. UI показывает требование, этап, задачи, исполнителей, вопросы, действия
   управления и итоговые доказательства; queued не отображается как applied.
8. Миграции, старые binary guards, ACL/leases и all-project snapshot/restore
   проверены отдельно перед capable production roll-forward.
9. Native Windows/Claude, Codex и второй/третий физический ПК отмечаются
   отдельно. Cross-build, SDK и два запуска на Mac их не заменяют.

Ни один source commit, UI макет или synthetic fixture не является полным PASS
этого этапа. Версии, фактические проверки, ограничения и оставшиеся действия
фиксируются здесь по мере получения доказательств.

## Composite source checkpoint

WORK189: independent SOURCE APPROVE получен для18565/18631/18745/19120/19397;
все manifests независимо проверены без drift. Это не production/native PASS.
WORK197 два existing schema6 acceptance после отдельного review прошли actual
process tests8.649s. Root frontend103/tsc/Nextbuild и infra54 PASS; полный Go race
повторяется после последних frozen corrections. Новый native preflight WORK198
и Linux fixture WORK199 готовят отдельные testers; до source review и root GO
они не исполняются. Production по-прежнему273/schema5, Test Mac enrolment
и автоматическая policy ещё не созданы.

Final frozen composite checks: `go test -race ./...` PASS, включая весь
acceptance process package31.564s, store100.737s и worker12.526s. Frontend103
unit/TypeScript/Next production build PASS; infra54 PASS. Эти результаты
предшествуют отдельным actual native pipeline и Linux restore QA; production
273 не менялся. Подготовленные source/docs проверены на known local tokens,
provider-key/private-key patterns: findings0, private runtime не staged.
