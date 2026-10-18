# Конфигурация Accord: локальное управление

Реализация D использует YAML version1 и SQLite schema3. Полный контракт —
[config-contract](config-contract.md). Ни один YAML не применяется автоматически
из рабочего каталога. Пользователь SSH/ОС — локальный администратор; агентский
bearer не получает операций apply, mint, revoke, scopes или handoff-admin.
OAuth, GitHub events, уведомления и импорт E ещё не реализованы. Конфигурация с
их включением отвергается CONFIG_UNSUPPORTED, а не игнорируется.

## Первый запуск

`token create --db DB --owner NAME --machine MACHINE --output PRIVATE` при
создании новой БД явно использует NAME как единственного bootstrap local user.
Каталог PRIVATE должен иметь mode0700; новый файл будет0600. Существующий
allowlist эта команда не расширяет. Новый токен по умолчанию получает только
настроенные agent roles. Для human/design-owner укажите `--roles` явно, например
`--roles human`; owner должен присутствовать в auth.local.users.

Пустой проект можно сначала запустить командой `serve --db DB
--bootstrap-owner NAME`. Профиль bootstrap_compat содержит семь пилотных ролей,
прежние права и reopen; реальные начальные project/task/session TTL сохраняются.
`--project`, `--task-ttl`, `--session-ttl`, заданные явно после появления active
config, должны совпадать с ней. Неявные defaults не переопределяют настройки.

## Явное применение Davinci или собственного workflow

1. `agent-board config init --profile davinci > workflow.yaml`.
2. Замените OWNER разрешённым локальным login; настройте роли/права/области.
3. `agent-board config validate workflow.yaml` возвращает JSON valid/hash/errors.
   При invalid команда завершится с ошибкой. Шаблон с OWNER ещё не валиден.
4. `agent-board config get --db DB` возвращает active config_version и hash.
5. `agent-board config apply --db DB --file workflow.yaml --expected-version N
   --actor operator` применяет только поверх этой версии. Два конкурентных apply
   с одинаковым expected имеют одного победителя. Stale CAS всегда конфликтует.

Изначальный apply к новой пустой DB использует expected-version0 и дополнительно
требует `--bootstrap-owner NAME --project NAME --task-ttl 15m --session-ttl 15m`.
Bootstrap owner должен входить в новый auth.local.users. При offline миграции
schema1/2 эти же аргументы задают именно реальные прежние настройки процесса;
не угадывайте их по defaults. Миграция создаёт compatibility version1, поэтому
последующее применение целевого YAML использует expected-version1. Изменённый
кандидат не применяется частично. Миграция и явный apply — два разных события.

`config history --db DB --after-version N --limit 100` читает версии по числовому
порядку. Повтор того же hash с правильным expected — аудируемый no-op, version
не увеличивается; применение исторического YAML создаёт новую версию.
Raw YAML сохраняется только в private БД. HTTP GET `/api/v1/config` и
`/api/v1/config/history?after_version=N&limit=100` требуют bearer и возвращают
canonical config без raw YAML/комментариев. HTTP записи конфигурации нет.

## Живые сессии, scopes и безопасные правила

CONFIG_IN_USE содержит конкретные session/task/note keys. Изменение TTL, ролей,
прав/evidence/областей/блокирующих правил при затронутой живой работе отклоняется;
удаление используемой роли и уменьшение note limits ниже текущих данных также.
Ни аренда, ни сессия не освобождаются командой apply. Дождитесь окончания работы
и expiry; затем прочитайте новую версию и повторите CAS. Следующая транзакция
работающего сервера видит успешно применённый config без перезапуска.

`token list --db DB` показывает roles и scopes_version без секретов. Локальное
изменение: `token scopes set --db DB --id ID --roles role1,role2
--expected-version N --actor operator`. Удаление роли живой сессии отклоняется.
Отзыв токена остаётся немедленным и не освобождает task lease.
Legacy tokens мигрируют ровно на семь прежних ролей, включая human; новые роли
не появляются в scopes автоматически. Human claim и done оба проверяют owner,
members, category answer_by и complexity. Agent identity не становится OAuth.

Отдельные role/task/session TTL сохраняются. Singleton role lease может истечь
раньше session: старый SID теряет право записи, его task lease материализуется
как expired, новая сессия получает новую role generation. Bridge ограничивает
heartbeat/deadline минимальной TTL и role_expires_at; длинная session не продлевает
утраченную role lease. Idempotent replay не продлевает никакую аренду.

Warn/off для дополнительных evidence полей не снимают built-in floor. Неполный
результат имеет evidence_complete=false и не открывает completed gate. Независимые
серверные author sessions, exact SHA/generation, последние verifier assignments,
DAG и одна task на сессию обязательны при любом mode. Singleton warn/off при
singleton=true и one_task/cycle warn/off отклоняются при validate.

## Задачи, области и передача эпика

Create ready:false создаёт draft без description; переход в ready проверяет
обязательные поля. Priority по умолчанию3 у эпика, задача наследует priority и
areas эпика; явный [] у areas остаётся пустым. Области хранят canonical paths,
exact aliases разрешаются перед canonical prefix children. Single editor —
обязательная авторизация; overlap warn содержит task/session/owner/areas.

Дубликаты ищутся Unicode casefold, collapse whitespace и Dice по множествам
символьных триграмм. Block допускает force только coordinator с reason и audit.
`link type:duplicate_of` требует права отмены и reason: source отменяется атомарно,
target и попытки не объединяются. Task/epic зависимости образуют общий DAG с
неявными child→epic edges; epic→собственный child запрещено.

Coordinator передаёт открытый эпик transition state:open, owner,
expected_version и reason. Локальный администратор может выполнить
`epic handoff --db DB --key E-N --owner NAME --expected-version N
--reason TEXT --actor LABEL`. Новый owner должен существовать. Передача при
живых task leases отклоняется; задача/role lease не передаются другому SID.

## Миграция, backup и откат

До изменения schema1/2 создаётся проверенный private snapshot с checksum в
`DB.migrations/schemaN-<random>.db`. DDL, scopes, compatibility config и
user_version3 фиксируются одной SQLite write-транзакцией. Неуспех оставляет
старую схему. Исторические completion/statistics не перепроверяются новой схемой.

Backup/restore/recovery поддерживают schema1/2/3. Binary со старым поддерживаемым
schema максимумом обязан отвергнуть schema3. Откат binary через изменённую схему
без restore запрещён. Для возврата на schema1/2 восстановите соответствующий
pre-migration snapshot и выполните offline recovery reset перед выдачей новых
токенов; изменения после snapshot будут потеряны. Обычная активация/неудачный
старт не делают скрытый downgrade или fallback.
