# Архитектура пилота

> TASK012 добавляет project-local similarity и `duplicate_of`, не изменяя
> scheduler/lease/generation/SHA gates. Durable поля задач требуют нового
> `deduplication:1` permanent floor при той же board schema4; прежний binary
> не допускается поверх новых записей. Контракт/приёмка —
> [TASK012](TASK-012-dedup-team.md), capable update — projects/upgrade.py.

> TASK011: назначения задач pinned профилям, объяснимые gates и ручной цикл
> review → QA описаны в [coordination.md](coordination.md). Schema4/coordination1
> floor защищает новые task semantics; фактическая готовность — TASK011.

> TASK010 добавляет board schema3 (profiles/revisions), immutable snapshot
> при явном Describe и existing ACL/session guards. Выбранный после WORK140
> контракт — [agent-profiles.md](agent-profiles.md). Старые unbound sessions не
> объединяются с профилями. Capable update требует schema3 floor и полного
> all-project snapshot; TASK009 initial apply повторно не применяется.

> TASK009 по прямому поручению пользователя расширяет single-board ADR-001:
> Go registry с явными project ID/ACL и изолированными board/web stores,
> сохранение прежнего default state. Актуальные выбранные границы и wire —
> [multi-project.md](multi-project.md); состояние — [TASK009](TASK-009-multi-project.md).
> Старые ADR ниже сохраняют инварианты core; не служат запретом согласованного расширения.

Дата: 18.10.2026. Источник полномочий — поручение пользователя переработать
ТЗ и начать отдельный сервис, TASK-001. Контракт — [requirements](requirements.md)
и [API](api.md). Документ описывает целевую реализацию, не выдаёт проектирование
за runtime-проверку. Фактический статус и результаты — TASK и WORK.

Веб-клиент TASK-007 выделен в `web/`: Next.js/React/TypeScript получает
JSON-представление прежних browser handlers. Go остаётся единственным владельцем
авторизации, транзакций и доменных правил. Это дополнение к пилотному ядру ниже;
контракт клиента, topology и ограничения загрузки — [next-web.md](next-web.md),
приёмка — [TASK-007](TASK-007-next-paperclip.md).

## ADR-001. Отдельный Go/SQLite сервис

Принятое техническое допущение по поручению переработать: Go + SQLite,
один процесс на один проект, Go module в корне нового репозитория. Это
вариант 1 открытого вопроса исходника, а не якобы полученное отдельное
согласование пользователя. Альтернативы Rust и TypeScript не дают пилоту
существенного преимущества, но добавляют иной toolchain/поставку. Для проверки
установлен Go 1.27.2; минимальная поддерживаемая версия проекта — Go 1.26.1 в go.mod.
Pure-Go `modernc.org/sqlite` v1.60.1 и official MCP Go SDK v1.8.0 фиксируются
в go.mod/go.sum; обновление — отдельное обоснованное изменение.

Лицензия MIT уже установлена владельцем в remote
`https://github.com/kirill-samylin/accord.git` (origin/main `9537cb7`, LICENSE
©2026 Кирилл Самылин), принятом после передачи ссылки пользователем.
Предварительный Apache-2.0 из исходника заменён этим фактическим решением.
Remote подключён; TASK-002 разрешает push проверенной рабочей ветки.
Merge/main и выпуск не поручены. Working service остаётся
agent-board; переименование во время параллельной реализации не выполняется.

```text
cmd/agent-board       CLI serve/token/API client
internal/core        DTO, роль, lease/gate/transition rules, errors
internal/store       SQLite, transactions, tokens, events, idempotency
internal/transport   REST + MCP → тот же core/store
docs                 контракты, roadmap, evidence
.local               игнорируемые runtime, DB, бинарники, секреты
```

Бизнес-правила не дублируются между REST и MCP. Transport декодирует вход,
аутентифицирует, привязывает actor и отображает ошибки. Store атомарно читает
состояние и применяет правила. CLI не имеет отдельного пути обхода правил.
Файлы ориентировочно ≤300 строк; разделение по понятной ответственности.

## ADR-002. Транзакции и сохранность

SQLite WAL с busy_timeout и сериализованными write-транзакциями.
Разные DB-подключения/процессы обязаны конкурировать за SQLite write lock;
один Go mutex/MaxOpenConns=1 не доказывает межпроцессную атомарность.
Схема версиями; миграции встроены и выполняются до readiness. Неизвестная
более новая схема отклоняется без попытки «исправить» данные.

Отдельные таблицы tokens, sessions, tasks, role_generations, events, idempotency
и metadata. Эпик и задача — отдельные JSON-строки tasks; аренда, попытки,
связи и evidence вложены в строку задачи. Ссылочную целостность проверяет core
в транзакции; SQL foreign keys для вложенных ссылок пилот не обещает.
Events имеют SQLite integer primary key и append-only; idempotency читается
точечно. Полный журнал не загружается и не переписывается каждым heartbeat.
Задачи/сессии пока загружаются целиком и сохраняются по строкам с проверкой
изменений: это ограничение пилота, не доказательство нагрузочных целей V1.
Связь, состояние, попытка, события и сохранённый ответ фиксируются вместе.
Частичный успех мутации невозможен; при rollback нет частично снятых gates.

Серверные времена хранятся UTC с достаточной точностью. Clock инъектируется
в unit/integration tests, не доступен клиенту. Restart сохраняет generation,
привязку owner/session, request_id и cursor. Подтверждённая транзакция
не откатывается пользовательским retry после сетевого обрыва.

## ADR-003. Сессии и leases

Токен — случайный секрет ≥256 бит, проверочный hash вместо исходного значения.
Сравнение не раскрывает токен; metadata token owner/machine/expiry хранится
отдельно. `session_id` выдаёт сервер и связывает с principal ID, ролью, run_id
и machine токена. Право на human-роль в локальном пилоте у локального владельца;
полный human OAuth/eligibility не считается реализованным.

Role singleton в пилоте только architect. Coordinator non-singleton: каждая
машина может координировать независимо. Без фиктивного coordinator → executor
handoff: executor сам describe и next. One-task-per-session проверяется атомарно.
Role lease и task lease — отдельные subjects со своими generation и expiry.

Алгоритм мутации:

1. Проверить живой token и принадлежность session, начать write transaction.
2. Найти idempotency запись scope+request_id: exact replay вернуть исходный
   результат; конфликт canonical payload отклонить. Replay ничего не продлевает.
3. Привести истёкшие leases затрагиваемых subjects к закрытым попыткам; их
   недействительность определяется now, а не запуском sweep.
4. Проверить role/task ownership, generation, now<expires_at, разрешённый
   переход и текущие gates/evidence. Ошибка не продлевает аренду.
5. Записать изменение, попытку/lease, события и сохранённый результат; commit.
   Новая успешная мутация продлевает session/singleton, а task — только если
   мутация проверила её живую аренду и generation (update/comment/link) либо
   явный heartbeat. Создание другой задачи не продлевает удерживаемую.

`describe` новой singleton-сессии не оживляет старую роль. При потере роли
старой session новый захват роли только новой session. Heartbeat принимает
ожидаемые generations; stale generation/expiry отвергает весь запрос.
Старая task generation никогда не получает права нового поколения.
SessionTTL — отдельный ограничитель сессии, не синоним TaskTTL; его expiry
также проверяется без фонового sweep. Читающий get не держит процесс живым.

Sweep допустим для материализации lease_expired, но не нужен для правильности.
Событие истечения и закрытие попытки ровно однократны независимо от того,
кто обнаружил expiry: next, mutation, sweep или restart. После потери связи
клиент прекращает локальные записи; сервер не способен обеспечить это сам.

## ADR-004. Результаты проверок и provenance

Review/tester — самостоятельные задачи. Они содержат target developer key
и immutable target SHA/generation. Generation фиксируется при создании для
done target либо при первом claim проверки после target done; до того 0 —
непривязанное значение, не wildcard для готового approve.
Evidence завершённой реализации содержит SHA, попытки
сохраняют server-generated session IDs авторов результата. Проверка не может
использовать сессию, участвовавшую в создании этого результата. Один владелец
токена может запустить автора и независимого reviewer разными сессиями;
сервер не превращает это в доказательство разных людей/моделей.

Gate — predicate, а не просто parent.state==done:
completed для обычной работы; approved для reviewer; passed для tester.
Отрицательный completion сохраняется, но gate false. Даже явное condition
completed не ослабляет positive requirement проверки. Review успешен только
при target done и совпадении target/evidence/verified SHA и target generation.
Переоткрытие target даже с тем же SHA
немедленно делает gate false. Новый цикл проверки создаётся отдельной задачей
на новый SHA; старый approve остаётся историческим.

Проверка gates выполняется при claim и завершении потребителя. Потребитель,
который уже работает при invalidation, не получает право успешно завершиться
на старой проверке. Уже завершённые результаты не стираются; родительский
эпик снова open. Закрытие эпика отдельно проверяет положительные обязательные
проверки, а не только terminal status дочерних задач. Исторические проверки
заменяются новым циклом: для каждой когда-либо назначенной target роли
reviewer/tester нужна актуальная положительная проверка. Удаление старой edge
через audited unlink не стирает эту обязанность, но позволяет заменить edge.
Проверки target учитываются глобально, независимо от эпика reviewer/tester.
В текущей generation/SHA обязательна последняя назначенная проверка каждой
роли по серверному порядку создания. Старый positive не перекрывает новый
pending/negative; новый positive заменяет предыдущий negative. Изменение
gates сверяет ранее закрытые эпики общим predicate и атомарно переоткрывает
потерявшие допустимость, включая межэпиковые и транзитивные зависимости.

## ADR-005. Идемпотентность и события

Idempotency key = principal ID + session ID + request_id; describe без session.
Hash канонического DTO включает operation и разрешённые поля; JSON whitespace/
порядок ключей не создаёт новое действие. Неверные/неизвестные поля отклоняются
до изменения. Успешный ответ сохраняется в той же транзакции с business state.
DB crash между mutation и response не создаёт дубликат при повторе. Не удалять
idempotency в пилоте; retention policy полной V1 требует отдельного решения.

Events page фиксирует high_water на начало чтения и проходит id>since до
него по возрастанию. Limit ограничивает возвращённые совпадения. next_cursor
ставится на последний безопасно просмотренный id; следующий подходящий id
не перескакивается. Если совпадений больше нет, можно продвинуться до high_water.
Новые события выше high_water будут на следующей странице. Фильтры одинаковы
в одном проходе; сохранение high_water из произвольного ответа запрещено.

## ADR-006. MCP и авторизация

Official Go SDK v1.8.0, Streamable HTTP stateless handler; board-session в
tool input. Основная версия: 2026-07-28; compatibility 2025-11-25 добавлена
в TASK-002 после выявленного сбоя реального Codex. Отдельная регрессия проверяет
legacy initialize/initialized/list/call, существующий SDK-тест — основной протокол.
Фактический результат повторной приёмки фиксируется в TASK-002. Версии перечисляются
через ServerOptions.SupportedProtocolVersions, не самописный JSON-RPC handshake.
SDK 1.8.0 поддерживает 2026-07-28 через stateless HTTP; stateful handler на
новой версии имеет иной контракт, поэтому его не включаем случайно.

Список 11 tools стабилен при авторизованном запросе и не меняется после describe.
Роль — состояние board, не скрытый переключатель списка инструментов соединения.
Только контекст авторизации запроса может фильтровать список. Пилот использует
общий список и серверные role checks. OAuth discovery, protected resource
metadata, audience и PKCE — отдельный этап V1; bearer-пилот не объявляется
реализацией полного OAuth или поддержкой любого клиентского инструмента.

Локально loopback HTTP; remote deployment только с TLS и проверкой proxy trust/
Origin/limits. HTTP size limits, JSON unknown-field rejection, body close,
timeouts и нормальные ошибки обязательны. В логах нет bearer/query secrets.
Новый DB безопасно создаётся regular 0600 до первого SQLite IO. Существующие
DB и sidecars проверяются до открытия: regular 0600, без symlink; каталог
не group/other-writable. Небезопасный путь вызывает отказ без исправления
чужих файлов. Подмена владельцем каталога/тем же OS UID вне модели угроз.
Core не доверяет присланным participant/author_session/generation без проверки.

## ADR-007. Области и будущие интеграции

Области пилота — теги. Будущая V1 нормализует aliases к canonical root проекта,
различает `scope` (роль/эпик заметки) и файловый `area`, отклоняет traversal и
двусмысленные roots; дочерний путь пересекается с родителем. Сервис не следует
symlink клиента и не блокирует фактические файлы. Путь сервиса/БД на сервере
не является root рабочего дерева участников.

GitHub webhooks отложены: delivery ID dedup, signature, SHA и события, но
отсутствие доставки не доказывает полноту. При ограничении «без GitHub API»
unknown completeness восстанавливается ручной redelivery. Полнота/CI статус
не заменяются догадкой. Проектный мигратор не встроен в сервер.

## Первичные источники (проверены 18.10.2026)

- [MCP SDK v1.8.0](https://github.com/modelcontextprotocol/go-sdk/releases/tag/v1.8.0):
  stateless protocol и supported versions.
- [MCP tools 2026-07-28](https://modelcontextprotocol.io/specification/2026-07-28/server/tools):
  список зависит от авторизации запроса, не побочного эффекта соединения.
- [MCP authorization](https://modelcontextprotocol.io/specification/2026-07-28/basic/authorization):
  OAuth требует полноценного отдельного контракта.
- [SQLite transactions](https://www.sqlite.org/lang_transaction.html) и
  [WAL](https://www.sqlite.org/wal.html): single writer и конкурентное чтение.
- [GitHub failed webhooks](https://docs.github.com/en/webhooks/using-webhooks/handling-failed-webhook-deliveries):
  неуспешные доставки не повторяются автоматически.
