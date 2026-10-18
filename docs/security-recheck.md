# Security recheck — TASK-001, WORK-010

Дата: 18.10.2026. Вердикт: **APPROVE для локального пилота** в проверенной
области. SEC-01 и SEC-02 из [первичного заключения](security.md) закрыты
проверкой исправленной версии, новых блокирующих security findings нет.
Это отдельное заключение безопасности; оно не заменяет code review R1/R2,
process/E2E и общую приёмку TASK-001 и не объявляет готовой production/V1.

Проверяющий — отдельный агент `security_recheck`, не автор исходной реализации
или WORK-008. Именованные Claude-модели здесь недоступны; автор и проверяющий
используют одну доступную модель среды. Общий процесс, security-профиль,
AGENTS, TASK-001, requirements/architecture/API, README/development,
WORK-007/008 и исходное заключение прочитаны. Tandem отключён по прямому
указанию пользователя. Код приложения и чужие WORK не менялись.

## Проверенная версия

Base HEAD: `9537cb737ee3d1f1e3d63eeccc6a81e2fa681698`.
Исходники незакоммичены; 36 Go/source/test файлов и go.mod/go.sum
зафиксированы SHA-256 манифестом ниже. SHA-256 самого манифеста:
`0bc09f76873291c9ee5e107640d3e2b8d850ad4db6c20719b21e5acfa30ef1d8`.
Локальная копия: `.local/evidence/security-recheck/source-sha256.txt`.
Все 11 хешей frozen-дельты из WORK-008 совпали с фактическими файлами.
Смена проверенных файлов требует проверки затронутой области заново.

Сравнение с манифестом первичного security review подтвердило неизменность
`auth.go`, `sessions.go`, всех `cmd/agent-board` и `internal/transport` файлов,
`go.mod`/`go.sum`; результат —
`.local/evidence/security-recheck/initial-manifest-comparison.txt`.
Первичный отчёт сохраняется как история findings; его прежний отрицательный
вердикт по SEC-01/02 заменён этим заключением только для указанной версии.

## Граница доверия и закрытие findings

Активы и сетевые границы остались как в первичном заключении: данные доски,
целостность мутаций/аренд, исходный bearer и проверочные хеши; HTTP/MCP inputs
недоверены. Локальный администратор, владелец DB/data-dir и процессы с тем же
OS UID доверены. Защита от конкурентной подмены файла этим владельцем не
заявляется. Проверено поведение POSIX mode bits на macOS arm64; произвольные
ACL, Windows permissions и удалённое production-развёртывание не подтверждены.

**SEC-01 — закрыто.** `internal/store/files.go` вызывается до `sql.Open` и
первого SQLite IO. Новый DB создаётся `O_CREATE|O_EXCL`, mode0600. Existing DB,
`-wal`, `-shm`, `-journal` проверяются через Lstat: только regular0600, без
symlink. Родитель допускает0755, но не group/other write. Небезопасный
существующий0644 файл вызывает безопасный отказ; автоматического chmod нет.
Это принятая policy, а не обещание открыть любую историческую БД.

Независимый probe установил umask022 и existing parent0755. Новый DB и живые
WAL/SHM получили0600; повторное открытие при живом WAL и после полного close
сохранило0600 и возможность Authenticate. WAL действительно содержал
синтетический plaintext задачи, поэтому проверялись используемые sidecars.
Изменение одного уже существующего live DB/WAL/SHM на0644 приводило к отказу
нового Open; SHA-256 содержимого и режимы всех трёх компонентов остались
неизменны. На отдельной матрице DB/3 sidecars ×0644/symlink/dangling/directory/
FIFO отказ сохранял проверяемый объект и foreign target; invalid sidecar
не приводил к созданию отсутствующего DB. Родители0777/0770/0702 отвергнуты.

SQLite URI `file:` отвергается до IO: foreign targets сохранены. Обычное имя
с `?`, `#`, `%` открылось точно как проверенный filesystem path через
закодированный URI. Источник подтверждает отсутствие глобальной смены umask;
уменьшение umask делалось только в отдельном процессе синтетического probe.

**SEC-02 — закрыто.** `identifier()` использует32 байта `crypto/rand.Read`;
`CreateToken` возвращает `ab_` + hex этих байтов. Фактический секрет декодирован
и проверен на≥32 байт без вывода значения. Это256 случайных бит; prefix/hex
в энтропию не засчитываются. В SQLite хранится SHA-256 секретного значения;
raw bearer не найден в DB/WAL/SHM. Create/Authenticate/Revoke и отказ exact
`describe` replay после revoke проверены. Источник по-прежнему проверяет
revocation/expiry в транзакции перед replay; транспорт не принимает Principal
из клиентского JSON. Уже линеаризованные in-flight операции отзыв задним числом
не отменяет, как и было описано в первичном заключении.

## Фактические проверки

Все запуски здесь: macOS arm64, локальный Go1.27.2 с GOTOOLCHAIN=local,
GOCACHE/GOMODCACHE/GOTMPDIR из docs/development.md.

PASS: 8 top-level store tests затронутой области; полный вывод —
`.local/evidence/security-recheck/targeted-tests.txt`:

```sh
go test ./internal/store -run 'TestDatabase|TestTokenHashExpiryRevocationAndSessionBinding|TestLostResponsesReplayWithoutRenewalOrDuplicateEvents|TestRejectedMutationLeavesNoPartialTaskOrEvents|TestLinkRequiresLiveConsumerGeneration' -count=1 -v
```

PASS: 3 существующих CLI tests (`TokenFileAndRevoke`,
`TokenFilePrivacy`, `CallDescribeCreateGetAndEvents`), вывод — `cli-tests.txt`:

```sh
go test ./cmd/agent-board -run 'TestTokenFileAndRevoke|TestTokenFilePrivacy|TestCallDescribeCreateGetAndEvents' -count=1 -v
go build -o .local/evidence/security-recheck/agent-board ./cmd/agent-board
go run .local/evidence/security-recheck/probe.go
python3 .local/evidence/security-recheck/cli-probe.py
```

PASS: отдельные adversarial probes из этого recheck; исходники и результаты
в `.local/evidence/security-recheck/probe.go`, `probe.txt`, `cli-probe.py`,
`cli-probe.txt`. Это временные проверки проверяющего на синтетических данных,
не maintained process/E2E suite отдельного тестировщика.

CLI повторно подтвердил фактические256 бит токена, новый output0600, отсутствие
сырого токена в stdout/stderr, точный SHA-256 в SQLite и revoke. Existing regular,
symlink и dangling output отклоняются без изменения целей.307 redirect не
выполняется; нелокальный plain HTTP отвергается до отправки bearer. Исходники
CLI/transport/auth и первоначальные ограничения вывода/логов не изменились.
Повтор полного первичного сетевого аудита не выполнялся: дельта не меняла эту
область, а неизменность подтверждена побайтовыми хешами.

NOT-RUN этим проверяющим: полный `./...`/vet/race/make check (единый прогон
владельца общей приёмки), новые process/E2E отдельного tester, реальные
Claude Code/Codex, два физических ПК, Linux/Windows, custom ACL, нагрузка,
отказ диска/kill во время commit, remote TLS/proxy, production deployment,
Govulncheck/vulnerability database scan. Отсутствие известных CVE не заявляется.
OAuth/scopes, human eligibility, единый REST Host policy и production hardening
остаются зафиксированными границами/рекомендациями первичного review/V1.
R1/R2 проверяет отдельный code reviewer; этот отчёт не выдаёт verdict за него.

## SHA-256 проверенных исходников

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
d85db430b4cc84173e35a09d1d6baf156cdd819a5f68b37d0a47f5667afc4aeb  internal/store/dependencies.go
d1ab7291bb546a3c94aa0911332c24974964688cc26a8e76cd85635312432a47  internal/store/epic_gates_test.go
c6811d2995f58376acdfec6da29de1d70fe3b6973f6acc7ea5753752e16fbc8d  internal/store/events.go
1af2c93771d4c2a22b4fe2a763a366b48800cd4628c728cb348d4ed6834ad45e  internal/store/events_test.go
0410e12d548766bfbfd34152184706c5fed81b7e26c91c4d926e4b086280121b  internal/store/evidence.go
716876bfcfccfc1083e6f587b9df2f5b5aa8a293584f0cf383e42c1f8f55763c  internal/store/files.go
bbb8cdca1de13291e485ce720b49d076462b94532ae8b0280e0a07ba9e46c521  internal/store/files_unix_test.go
6a0b04a96a8f6af6868d8b919abb77fe9696810c04783b5af0b3ef5bb8876c9a  internal/store/gates.go
36314a204102ff0a5cb31ea6243a0cd046a524081bf79a837a1eee7bc6a8ce29  internal/store/global_verification_test.go
cf4e09bd4089e9fa46906b336c3624c8e66d1878d39e445a62e80b0e1ce7713f  internal/store/helpers_test.go
1eddcf6006619d8baf99fe7a0384facd02c1ae726e717f43a703de880fd7ec16  internal/store/idempotency_test.go
303811e0b51d5a7fec1d8d12c4ab7857daf71cb75034de5d9627adb24dce0f1b  internal/store/leases_test.go
6a27fbf7bac4aab86f55dd647ebcdb4bde189dd34ab4f074f2c4e2668561ed9c  internal/store/mutation.go
7f0a16ea505dc274bceb67775c016fb791307ca8f41e6ab2385405d0feb2f937  internal/store/persistence.go
3ce94f5dc415040248e044558b83ece5d42363c14c89187fb797175fbea644c2  internal/store/reviews_test.go
87c29d4493bafcd3fe70a5d91f7b845dc4a643bc93a091ec265d2cf98772d360  internal/store/sessions.go
f05a9bbea274114cefbd24fe19eb6565a7faa0c35340780e2380c9f8a78bc8b3  internal/store/store.go
ccc12156e679522cfe2533e40b3f606a782def4623b54484f8aad4663b35b31b  internal/store/tasks.go
d646639e80b8b14b68c2385d9018d6710ef6ed76bbdb46ab8ce7b417f93ea1eb  internal/store/transitions.go
f1bf15239fd8bb6474b131e6852ef87459a28e97ea28115360379da518923ea0  internal/store/workflow_test.go
6a3689502c5df63fe1b397b37e35fea228ffeeecf868d2c6c92523a507bc6ba2  internal/transport/errors.go
c688b58349cf3ee03086cbd9aee07a2cc6c67599380feb7806c1dae13893dde1  internal/transport/http.go
380ce13112322172caeaf239d99e56eb24cafcce042f87fb3db3c171525f829b  internal/transport/http_test.go
d84b622842e31f7086ec16ab53168436b12f41e6e175695c706da2b4b79202b6  internal/transport/mcp.go
a00adb7a12e70a1743e5bd4ac4bcb5ac034167ec40e8459a61b050c0990dc10a  internal/transport/mcp_test.go
bbfe56b23c14e694829adb80e28cda5c4d6f7ea8e9d5719acc29b8bdbecf209c  internal/transport/operations.go
999c73691e8055d3fb94426b7bdac25c3686d6a7998d8b15a59cd59b2773bb93  internal/transport/workflow_test.go
```
