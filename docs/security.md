# Security review — локальный пилот TASK-001

Дата: 18.10.2026. Проверяющий — отдельный security-агент, не автор реализации.
Вердикт: **CHANGES REQUESTED** до устранения SEC-01 и SEC-02 и проверки
исправленной версии. Это заключение о локальном пилоте, не production audit.

## Проверенная версия

Base HEAD: `9537cb737ee3d1f1e3d63eeccc6a81e2fa681698` (initial remote MIT).
Приложение проверено как незакоммиченные новые файлы. SHA-256 манифеста
31 исходного/тестового файла и go.mod/go.sum:
`eaf6695b9578be4829ba2737943f1ece73cf2edd85a5ef13f508d15e63d78be2`.
Сам манифест приведён ниже; локальная копия —
`.local/evidence/security/source-sha256.txt`. Последующее изменение этих файлов
требует повторного заключения по затронутой области.

Прочитаны AGENTS.md, TASK-001, requirements.md, architecture.md, api.md,
README.md, development.md, общий процесс и security-профиль Давинчи.
Tandem не использовался по прямому указанию пользователя. Именованные модели
Claude здесь недоступны; отдельные авторы и проверяющий используют модель среды.

## Модель угроз

Активы: содержимое задач/evidence/попыток и сессий, целостность очереди/gates,
права владельцев аренд, исходные bearer-токены и их проверочные хеши.
Недоверенные входы: HTTP/MCP запросы, session/task/generation/request_id,
метаданные и ссылки участников, сетевые ответы CLI. Недоверенный токен не даёт
доступ к доске; действующий участник может ошибаться или пытаться изменить
чужую аренду. Каждый токен пилота вправе открыть любую разрешённую роль:
разграничение по scopes и отдельная human eligibility отложены в V1.

Локальный администратор CLI, владелец БД и процессы с тем же UID доверены:
их возможность отозвать/создать токен или редактировать БД не является обходом
сетевой авторизации. Соседний локальный UID не должен получать данные через
случайно публичный WAL. Разрешённый bearer нельзя отождествлять с доказательством
разных людей/моделей: независимость здесь только по server-generated session.

Сервис слушает literal loopback, не выполняет код, не читает artifact-ссылки и
не управляет агентами. Prompt injection в текстах остаётся недоверенным
содержимым для клиента; сам сервер не интерпретирует его как команды.
OAuth/TLS/reverse proxy/GitHub/UI и два физических ПК вне этого заключения.

## Обязательная единая дельта

### SEC-01 — P2: новый WAL доступен группе и другим пользователям

Место: `internal/store/store.go:59-92`. `sql.Open`/первый SQLite IO и
`journal_mode=WAL` выполняются до `os.Chmod(path,0600)`. При обычном umask022
новая БД сначала создаётся публичной; её WAL/SHM наследуют режим0644.
Последний chmod исправляет только основной DB, оставляя sidecars0644.
`MkdirAll(parent,0700)` не исправляет права уже существующего каталога.

Воспроизведение на fresh синтетической БД: DB0600, WAL0644, SHM0644;
WAL содержит синтетический plaintext заголовка задачи. Условие утечки:
пользователь выбрал существующий searchable0755 каталог `--db`, доступный
соседнему UID. Приватный0700 parent из README закрывает доступ как mitigation;
ошибка не даёт удалённому анонимному клиенту читать произвольные файлы.

Исправление: до первого SQLite IO создать новый regular DB с0600 безопасно,
либо отказать для небезопасного data-dir. Existing DB/sidecars должны быть
проверены/защищены до открытия; небезопасные symlink/non-regular paths нужно
отвергать без следования/повреждения чужих файлов. Не достаточно снова chmod
основного DB после инициализации. Не следует глобально менять process umask
при конкурентной работе.

Повторная проверка: свежая БД в существующем0755 каталоге при umask022 и restart
с existing DB/WAL дают все DB/WAL/SHM0600 либо безопасный отказ. Добавить regression
защиты существующего чужого файла/символической ссылки. Доказательство:
`.local/evidence/security/probe.go` и `probe.txt`; секреты не выводятся.

### SEC-02 — P2, контракт: 192 бита вместо заявленных минимум256

Место: `internal/store/mutation.go:18-23`, используется `CreateToken`.
`make([]byte,24)` даёт192 случайных бита; ADR-003 требует≥256.
`ab_` и hex-представление энтропию не увеличивают. Текущие192 бита не являются
практически подбираемым токеном; это конкретное расхождение security-контракта.

Исправление: токену нужно≥32 байт из `crypto/rand`; допустимо увеличить общий
identifier либо ввести отдельный генератор токена. Regression проверяет
минимальную длину decode hex, отсутствие raw secret в БД/обычном выводе,
работоспособность token create/auth/revoke. SHA-256 подходит для случайного
высокоэнтропийного bearer: password KDF здесь не требуется.

## Фактические проверки

PASS: targeted existing unit/integration тесты (13 top-level тестов) на
Go1.27.2/macOS arm64, список и полный вывод в
`.local/evidence/security/targeted-tests.txt`. Проверены token hash/expiry/revoke,
чужая session/machine, replay без повторного продления, rollback отвергнутой
мутации, lease expiry/restart/fencing, атомарность singleton heartbeat,
право на живого consumer при link, REST unknown principal/fields/trailing/null/
body cap/query/method, cross-origin отказ и loopback bind, SDK workflow/revoke,
CLI mode0600/O_EXCL и отказ публичного token file.

PASS: build проверенного CLI в `.local/evidence/security/agent-board`.
Отдельные adversarial probes с собственными синтетическими данными:

- CLI не следует307 redirect; второй endpoint не получил запрос.
- CLI отвергает plain HTTP к нелокальному IP до отправки секрета.
- token output не перезаписывает existing regular, symlink и dangling symlink;
  existing synthetic data и исходная цель ссылки сохранены.
- revocation после первоначальной аутентификации запрещает exact describe replay.
- SDK localhost protection возвращает403 на чужой Host даже с валидным bearer.
- Fresh SQLite permissions выявили SEC-01, token length выявил SEC-02 (FAIL).

Исходники подтверждают: transport создаёт Principal только из Authenticate,
не принимает его из JSON; core повторно проверяет revocation/expiry внутри
write-транзакции до replay. Захват SQLite write lock предшествует проверке:
отзыв, завершённый до получения lock, запрещает операцию; уже линеаризованная
операция может завершиться перед отзывом. Это не обещание отменять in-flight
запросы задним числом. Все сетевые мутации идут через единый Store.

SQL values параметризованы; конкатенируемые table names статические внутренние.
Domain candidate snapshot и транзакция сохраняют мутации/события/replay вместе.
В транспортных внутренних ошибках и panic recovery нет caller-секрета;
request/body/bearer не журналируются. Health/readiness не раскрывают данные.

REST использует CrossOriginProtection, но отдельный Host allowlist есть только
в SDK на `/mcp`: REST с совпавшими чужими Host/Origin проходит к аутентификации
и валидации. Без bearer он всё равно401; защищённые операции остаются
недоступны DNS-rebinding странице без секрета. Единый Host policy — рекомендация
к будущему proxy/browser deployment, а не доказанный auth bypass пилота.

Проверены локальные официальные исходники закреплённого MCP SDKv1.8.0
`mcp/streamable.go:174-195,317-340` и Go1.27.2 `net/http/csrf.go:134-174`:
body cap, localhost protection и поведение Origin API не предполагаются по имени.
Зависимости pinned/checksummed в go.mod/go.sum. Отдельный vulnerability database
scan/Govulncheck **NOT-RUN**; отсутствие известных CVE этим отчётом не заявляется.

NOT-RUN: полный test/vet/race (координатор выполняет один общий прогон),
process E2E отдельного тестировщика, реальные Claude Code/Codex, Windows/Linux,
удалённый TLS/proxy, нагрузка, отказ диска/kill во время commit и production.
N/A: execution sandbox/SSRF артефактов/webhook signatures/OAuth — таких путей
в пилоте нет. Проверка полноты workflow/gates — отдельное code review.

## SHA-256 проверенных файлов

```text
ae9702f6beea7a112b510b4bc6f52e6da52f42ba82bbbfdea6f5c6b8dafa63ca  cmd/agent-board/client.go
54fd07eb51bc1c2c669f6ab5a22dd9fdac80bfa0d100b6735d6b4eccc4c6eaa6  cmd/agent-board/client_test.go
f13e4a9d4668ee7559a254795d028ba50d26bc3c3c3552e7227fdd3c686e23cb  cmd/agent-board/main.go
5555bf7e57e26017072f5c967384d09e212a066c01fd604ba4c744928bf31026  cmd/agent-board/token.go
f1ee024ee7e93d6c54dabaf7a60e523c733a5e9a4855ed4b10ad7f9541fcc6c0  go.mod
83418331dd831bcf5abbd9646375e96e00bdfb8427450da8fc089c8ae6c41222  go.sum
8ffe8ba1ba6898ae6b3e2934014f04065e7590ee906de3b213d0f47d261baed6  internal/core/types.go
a8ee5a490f93d9219a0f3b304a1b2de04c94e41a00e16313be4ff7c30f8b8b86  internal/store/attempts.go
31e991211961509490208f2abbe4a20a46631df3ebdf1fa491d44f9e546859a3  internal/store/auth.go
1339af28bdb7ee196aafc2435d8385c077303a8c424f6bebc6d25b3033b52147  internal/store/dependencies.go
c6811d2995f58376acdfec6da29de1d70fe3b6973f6acc7ea5753752e16fbc8d  internal/store/events.go
1af2c93771d4c2a22b4fe2a763a366b48800cd4628c728cb348d4ed6834ad45e  internal/store/events_test.go
0410e12d548766bfbfd34152184706c5fed81b7e26c91c4d926e4b086280121b  internal/store/evidence.go
cf4e09bd4089e9fa46906b336c3624c8e66d1878d39e445a62e80b0e1ce7713f  internal/store/helpers_test.go
673ca0177999f3f416607ba5b667cca0f50de77e01499f074a2f76b183a03b99  internal/store/idempotency_test.go
303811e0b51d5a7fec1d8d12c4ab7857daf71cb75034de5d9627adb24dce0f1b  internal/store/leases_test.go
ae4938eb08957469a5a37c2685144a547618f9a46e25c852947ef67c7e8b2554  internal/store/mutation.go
7f0a16ea505dc274bceb67775c016fb791307ca8f41e6ab2385405d0feb2f937  internal/store/persistence.go
3ce94f5dc415040248e044558b83ece5d42363c14c89187fb797175fbea644c2  internal/store/reviews_test.go
87c29d4493bafcd3fe70a5d91f7b845dc4a643bc93a091ec265d2cf98772d360  internal/store/sessions.go
97f4dc1ca3d25e42c124385e223534cfc1c5f97db3a39d30eb965053e502a173  internal/store/store.go
27582bd443bbc48278bd02489493451dda51d4516de17a3641fe107c1a6d8962  internal/store/tasks.go
76a098adbb31f43f73d4625dfd42bc73910d3f1b5af20c19140f5c53d6771006  internal/store/transitions.go
f1bf15239fd8bb6474b131e6852ef87459a28e97ea28115360379da518923ea0  internal/store/workflow_test.go
6a3689502c5df63fe1b397b37e35fea228ffeeecf868d2c6c92523a507bc6ba2  internal/transport/errors.go
c688b58349cf3ee03086cbd9aee07a2cc6c67599380feb7806c1dae13893dde1  internal/transport/http.go
380ce13112322172caeaf239d99e56eb24cafcce042f87fb3db3c171525f829b  internal/transport/http_test.go
d84b622842e31f7086ec16ab53168436b12f41e6e175695c706da2b4b79202b6  internal/transport/mcp.go
a00adb7a12e70a1743e5bd4ac4bcb5ac034167ec40e8459a61b050c0990dc10a  internal/transport/mcp_test.go
bbfe56b23c14e694829adb80e28cda5c4d6f7ea8e9d5719acc29b8bdbecf209c  internal/transport/operations.go
999c73691e8055d3fb94426b7bdac25c3686d6a7998d8b15a59cd59b2773bb93  internal/transport/workflow_test.go
```
