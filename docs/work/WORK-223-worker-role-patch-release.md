# WORK223 — отдельная поставка исправленного worker

Координатор: root. Статус: CONTRACT_REVIEW_PENDING, исполнение не разрешено
этим документом. Основание — реальный FAIL WORK203V11: аналитик и разработчик
завершили работу, ревьюер получил отказ проверки и задал вопрос человеку.

## Узкий продуктовый дефект

`Engine.prompt` в `internal/worker/execution.go` предписывает локальную проверку
всем automation-ролям. `automationCheckRequest` разрешает её только developer
и tester. Исправление WORK222 согласует инструкции с существующим распределением
прав. Reviewer остаётся read-only и проверяет точный SHA, diff и доступные
подтверждения; analyst составляет план. ACL, sandbox, checks, attestations,
аренды, schema/floors и API не расширяются.

Автор WORK222 меняет только prompt и его регрессионные unit-тесты. Независимый
review202 и тесты обязательны до нового process/E2E. Старый V11 FAIL сохраняется.

## Компоненты поставляются с отдельными версиями

Проверяемый сервер, supervisor и Next остаются точным release
`a32db22973d5b43d4fb227a89337bf5b82a4969f`. Серверные handlers, migration/backup,
Go supervisor, Next и все их загруженные артефакты не меняются исправлением prompt.
В production пока установлен прежний release273; обновление всё ещё закрыто
обязательными runtime gates и capable all-project upgrade.

Для Mac собирается отдельный worker из commit с WORK222. Его настоящий revision,
SHA бинарного файла и base revision указываются отдельно. Нельзя выдавать этот
worker за исходный a32 binary или подменять старые native-evidence receipts.
Windows/Linux worker можно cross-build с тем же исправлением; это не native QA.

## Новые QA adapters

В descriptors добавляется отдельный pinned worker executable/revision. Старый
`candidate` остаётся точным a32 server/supervisor; его foundation guards A209,
C216 и N218 не ослабляются и продолжают проверять старый точный executable.
Все команды `worker identity`, `automation-preview`, `enroll`, `consent`, `run`
исполняются новым worker, supervised старым проверенным supervisor. Для B2 оба
воплощения worker используют новые уникальные имена и тот же pinned worker.

Обязательна независимая проверка diff base→worker commit: изменения production
ограничены ролью prompt, остальной server/supervisor/permissions/check/IPC код
побайтно совпадает с a32. Tests/docs могут отличаться и перечисляются явно.
Новые descriptors pin этот diff/provenance, worker binary и source approval.

Foundation receipts не объявляются фактическим запуском нового worker. Реальное
подтверждение нового worker — новый четырёхролевой native цикл, точный computed
permissions proof каждого запуска, настоящий developer commit и проверки,
независимые reviewer/tester и завершение с ACK/lock/pending proofs. Затем F6,
C2, H1, B2 выполняются тем же новым worker. В отчётах отдельно указаны сервер
a32 и worker revision/SHA; native Windows/Claude/второй физический ПК — NOT-RUN.

Нельзя менять prompts в QA-профилях только ради обхода продуктового дефекта,
разрешать reviewer запрещённый check или объявлять refused проверку пройденной.
Повтор требует нового source freeze, source202 receipt и одноразового root GO.
Все старые runtime/source/claims остаются неизменными. Fixture inventory проверяется
до GO целиком: SHA, файлы, owner, отсутствие ссылок, 0644 внутри private0700.

## Выпуск и подключение

212 deployment продолжает ставить a32 server/Next, с сохранением всех capable
guards, all-project backup и roll-forward. Его gates должны ссылаться на
независимые фактические отчёты тестов a32-сервера с отдельным исправленным worker;
receipt не должен скрывать различие версий. На Mac устанавливается именно новый
worker, который прошёл эти тесты. Текущий GUI checkout и его worker без отдельной
необходимости не меняются; Test_internet получает отдельное подключение/state.

До независимого CONTRACT APPROVE и source APPROVE WORK222 этот документ не
разрешает менять descriptors, собирать release или запускать новые модели.
