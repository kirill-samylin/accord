# WORK-166 — Панель управления агентами TASK013

22.10.2026. Автор managed web. Область: web/src managed UI и интеграции.
Контракт: docs/managed-agents.md, source contract APPROVE WORK165.
Применён React skill vercel-react-best-practices: scoped query cancellation,
параллельное получение данных сервером, отдельные компоненты, выводимые состояния.

В работе: компьютеры, run list/detail, start/stop/resume/clarify, серверные
capabilities/action IDs, uncertain exact retry, pinned profiles и черновики.
Browser enrollment не выполняется: локальный владелец разрешает папку/агента/
контроллеров через native worker. Online — heartbeat, не доказательство процесса.
Тесты автора: unit/integration; новые browser/process E2E — отдельный тестировщик
после независимого source APPROVE. Production/реальный Windows пока NOT-RUN.

## Готовность к независимому source review

Авторский UI завершён поверх baseline HEAD
`d9f9eba6fdecb131e36f49907d421a2df43cccbd` + текущие незакоммиченные TASK013
изменения. Commit/push/release не выполнялись.

Реализовано:
- `/app/p/{id}/managed`, отдельная навигация «Управление», вкладка рядом с
  профилями/сессиями; project-scoped список компьютеров и работ, run detail.
- Fresh worker heartbeat + ownership lease отделены от последнего process state
  и timestamp. Раздельные команды/delivery phases, truthful forced stop,
  queued clarification на границе turn, normalized bounded output/truncation.
- Start выбирает только advertised host/workspace/provider и active exact profile
  version; UTF-8 quotas, provider compatibility, busy workspace, независимый host
  CanControl и readonly project checks. Сервер повторяет authoritative проверки.
- Stop/resume/clarify используют существующий useAction/issued action_id/CSRF;
  uncertain outcome удерживает exact body/request scope, никаких новых действий
  до retry. Конфликт обновляет server capabilities, сохраняя текст и требуя
  явного принятия актуального состояния. Stop сохраняет неотправленное уточнение.
- Поллинг 5s с project route scope, AbortSignal, stop-on-auth-error и bounded
  backoff; background выключен; на время неопределённой мутации scope locked.
- Browser не выдаёт secret и не регистрирует hosts/controllers. Раскрываемая
  инструкция объясняет local enrollment, согласие владельца и IDE limitations.

Файлы: `web/src/features/managed/*`, `web/src/lib/managed-agents.ts`,
`managed-agents.test.ts`, `managed-actions.test.ts`, `managed-fixtures.test-support.ts`,
`web/src/app/styles/managed.css`; интеграции в globals/types/api/project-routes,
control-app/board-screens/navigation и profiles navigation/detail.
Новые production source файлы ≤300 строк. Тесты разделены по domain/action scopes.
Существующий control-app до изменения уже превышал этот ориентир; сохранена его
роль общей query boundary без unrelated рефакторинга.

PASS: TypeScript; 85 Node unit/integration tests (13 новых managed regressions);
Prettier всех затронутых web файлов; scoped `git diff --check`.
Лог тестов: игнорируемый `.local/managed-web-tests.txt`.
NOT-RUN: build (координатор выполнит после integration), независимое source review,
реальный browser/process/native CLI E2E, Windows/второй физический компьютер,
production/capable upgrade. Новые process/browser E2E автор не писал.
Backend author согласовал точные routes/DTO/actions; detail обязан включать
`managed.hosts` и `managed_run`, profiles на list для pinned start choice.

Дополнено по уточнению координатора: `ManagedPage.actor_principal_id` выводится
только внутри раскрываемой инструкции как «Ваш код управления для подключения
компьютера» с кнопкой копирования и явным объяснением local consent. Это server-
derived project-local principal ID; UI не подменяет его registry identity ID
и не отправляет remote controller grants. Повторный TypeScript PASS.

## Исправления partial review165 (22.10.2026)

P2 clock skew исправлен: backend добавил server `ManagedPage.observed_at`.
`hostOnline` теперь сравнивает server heartbeat/ownership lease только с этим
server observation timestamp; браузерный `Date.now()` не участвует в start/resume
freshness gates. Все host cards, detail и command reasons используют тот же
snapshot. В UI явно указаны «по последним данным» и время серверных данных;
скрытая/необновлённая вкладка не выдаёт старый snapshot за текущее наблюдение.

Защита409 дополнена на UI: конфликт требует явного принятия только actual newer
same-run detail version. Ошибка с list-only/отсутствующим detail или старой версией
не может подтвердить stale состояние кнопкой «Использовать актуальное состояние»;
предлагается сначала получить актуальное detail. Draft текст сохраняется. Backend
163 отдельно исправил `managedError` для возврата свежего command detail.

PASS: TypeScript; Node unit/integration88/88 (добавлены3 meaningful regressions:
browser clock ±2s/±24h с неизменными server snapshots и доступным start/resume,
реальный stale heartbeat/expired lease, newer exact detail vs missing/foreign
conflict acknowledgement); Prettier; scoped diff check. Native/browser/process
E2E автор не выполнял. Build ранее выполнен координатором, после этого изменения
новый build пока NOT-RUN. Независимый повтор source review остаётся PENDING.
Baseline HEAD по-прежнему `d9f9eba6fdecb131e36f49907d421a2df43cccbd`.
Ready fingerprint (SHA256 sorted path+NUL+bytes+NUL всех managed feature source и
`web/src/lib/managed*.ts`, включая тесты):
`f201667acca6d5a7cdbfef3fdd2e12413bfd6731cec0a17ecc84d5c50137a901`.
