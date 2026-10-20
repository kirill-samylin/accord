# TASK-011 — координация

20.10.2026. Ветка `codex/coordination`, старт clean checkpoint
`343b93c71297c58c972f73fed4d1d4ca3a349740`. Прямое продолжение порученной
пользователем очереди profiles → coordination → managed execution.
Статус: IMPLEMENTATION IN PROGRESS. WORK147 architecture APPROVE;
контракт [coordination.md](coordination.md) принят до реализации. Source review,
QA и поставка NOT-RUN.

## Владение и последовательность

- Root: канонический контракт/TASK, operations/infra, интеграция, release/VPS.
- WORK147: отдельный независимый архитектор, до реализации.
- WORK148: отдельный backend автор, core/store/projects/transports/web/CLI и
  авторские unit/integration, без process/E2E.
- WORK149: отдельный frontend автор, Next и клиентские unit.
- WORK150: независимый reviewer source/tests/operations.
- WORK151: root operations и интеграция.
- Независимый reviewer: source/author tests/operations и повтор после исправлений.
- После source APPROVE отдельный tester: process/E2E, затем review новых тестов.
- Synthetic Linux migration/update/recovery/restore → full production backup →
  reviewed capable update → HTTPS/preservation evidence → final review/closure.

## Согласование объёма

Пользователь20.10.2026 явно подтвердил «Да, этот контур» на вопрос о назначении
задач постоянным профилям, управлении зависимостями, причинах блокировки и цикле
review → QA в панели, при самостоятельном подключении работников. Contract
реализует именно этот ручной контур; managed execution остаётся следующим этапом.

## Исходные границы

Ядро уже имеет role-based atomic next, leases/fencing, зависимости и SHA/generation
review gates. TASK011 использует их как единственный источник правил. Новая
плоскость координации связывает задачи с постоянными профилями и даёт человеку
рабочие действия/объяснения в панели. Конкретные wire/data решения фиксируются
в `coordination.md` после независимой архитектуры; черновой план не приёмка.

Provider execution, автоматический dispatch, процессы работников, clone/GitHub,
credentials и полный V1 вне этапа. Browser human session не подменяет отдельного
developer/reviewer/tester и не завершает их работу от их имени. Профиль не
credential/presence/worker. Неизменный исходный документ не редактируется.

## Обязательная сохранность

Старые проекты, данные, ACL/epochs, cookie/logout, session snapshots,
leases/fencing, независимость сессий и SHA gates сохраняются. Любая новая
persisted semantics требует capability floor; schema migration — полного
all-project snapshot до DDL, additive preservation и только roll-forward.
Initial TASK009 apply и DB rollback запрещены. Tandem/соседний davinci-gui-
не используются. Credentials/реальные данные только private ignored .local/VPS.

## Исходная поставка

TASK010 COMPLETE/FINAL APPROVE: product `cbf202086431335d4657c722a0f75328a09dc4ea`,
closing checkpoint343b93c. Подробные hashes/runtime/backup/evidence — TASK010.
Это переданный исторический checkpoint; актуальность VPS проверяется до поставки.
Физический второй ПК, provider workers и полный V1 остаются NOT-RUN.

## Промежуточные результаты реализации

WORK149 Next source ready: client67/67, types/format/build PASS. Независимый150
web source slice APPROVE, включая explicit draft rebase, successful QA gate и
отключение действий при mixed task/coordination versions. WORK151 infrastructure
155/155 PASS с независимым повтором и slice APPROVE. Backend148 author tests ещё
в работе; общего source APPROVE нет. Новый tester152 пока только read-only план,
process/E2E ещё не пишет и не запускает. Production не переключён.

## Source APPROVE и общий check

WORK150 общий SOURCE APPROVE; обязательные findings закрыты: stale generation
не становится admin fallback, current/positive различаются, bounded explanations
детерминированы и не теряют reviewer/tester причины, schema4 exact singleton.
Binding74source/author-test files SHA256
`5cd77400d904fe72783b3beabda63b1fa3a4137ddfc914e651a3347548cc672a`; root
независимо повторил вычисление и получил тот же hash. Backend148 полныйGo/race/vet
и отдельные finalreader regressions PASS, Next149 client67/types/format/build PASS.
Root полный make check PASS: Go/all/race/vet/build, прежнийembeddedweb suite,
infra155, Next67/types/format/build. Private log `.local/task011/full-check-source.log`.
После этого tester152 разрешены новые process/E2E. Это ещё не closure: localQA,
Linux/update/restore/production/HTTPS/evidence-review остаются NOT-RUN.
