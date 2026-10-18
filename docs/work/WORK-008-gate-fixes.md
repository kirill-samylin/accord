# WORK-008 — Исправления gates и приватности SQLite

- Родитель: TASK-001-pilot. Роль: разработчик, экземпляр `gate_fixes`.
- Статус: на проверке; авторская дельта заморожена 18.10.2026.
- Вход: AGENTS, TASK-001, требования/архитектура/API, docs/review.md,
  WORK-005; профиль developer и общий процесс соседнего Давинчи прочитаны.
- Основание: R1/R2 независимого ревью, SEC01 и энтропия токенов WORK-007;
  решения координатора о cross-epic и latest-assignment semantics.
- Область: internal/store и эта WORK; DTO/core, transport/CLI, go.mod/sum,
  requirements/architecture/API и общая TASK не менялись этим исполнителем.
- Tandem выключен по указанию пользователя. Push/remote/PR не выполнялись.
- Именованные Claude-модели в среде не закреплены; отдельный разработчик
  использует модель текущей среды. Независимый approve себе не выдаёт.

## Реализация

R1: единый `epicClosable` используется явным close и reconciliation ранее
done эпиков после успешной мутации. Проверка выполняется до сохранения
идемпотентного ответа и коммита той же SQLite-транзакции. Несогласованные
эпики открываются с `epic_reopened`; child done, evidence и attempts остаются
историческими. Рекурсивные gates проходят через завершённых промежуточных
потребителей и cross-epic связи; implicit review-target циклы fail closed.

R2: обязательства reviewer/tester ищутся глобально по developer target,
включая проверку из другого эпика. Создание проверки явно открывает также
эпик target. Для target+role+актуальные SHA/generation засчитывается последняя
назначенная подходящая проверка по числовому серверному T-N; preplanned
generation=0 допускается при выборе до первого claim/binding. Отсутствие
актуального цикла не стирает ранее назначенную обязанность. Новый pending
или negative supersedes прежний approve/pass; новый positive заменяет
historical negative/stale. Старый positive source больше не открывает
explicit dependency: требуется явный unlink/relink на новый positive.

SEC01: DB precreate O_CREATE|O_EXCL 0600 до первого SQLite I/O. Existing
DB/-wal/-shm/-journal разрешены только как regular 0600, без chmod чужих
файлов. Symlink/dangling link и publicly writable parent отклоняются;
parent 0755 допустим. SQLite URI `file:` на входе отклоняется; обычный путь
становится абсолютным и кодируется для точного открытия проверенного файла,
в том числе при `?`/`#` в имени. `:memory:` остаётся test-исключением.
Владелец каталога и процессы под тем же OS UID доверенные: это не обещание
sandbox против конкурентной подмены владельцем. POSIX права на Windows
этой реализацией и текущими тестами не подтверждены.

Случайный identifier увеличен 24 → 32 bytes. Регрессия декодирует реальный
secret после `ab_` и проверяет минимум 256 случайных бит, не печатая значение.

## Фактические проверки

PASS на финальной исходной дельте:

```sh
GOTOOLCHAIN=local GOCACHE=$PWD/.local/toolchains/cache/build GOMODCACHE=$PWD/.local/toolchains/cache/mod GOTMPDIR=$PWD/.local/toolchains/tmp .local/toolchains/go/bin/go test ./internal/store -count=1
GOTOOLCHAIN=local GOCACHE=$PWD/.local/toolchains/cache/build GOMODCACHE=$PWD/.local/toolchains/cache/mod GOTMPDIR=$PWD/.local/toolchains/tmp .local/toolchains/go/bin/go test -race ./internal/store -count=1
```

Результаты финальных запусков: unit/integration 0.689s, race 4.310s.
Затронутые файлы форматированы gofmt, production/test Go-файлы области
проверены по ограничению ≤300 строк. Новые standard-library API сверены
с документацией установленного Go: os.OpenFile, os.Lstat, net/url.URL.String.

Новые regression сценарии: link на done consumer; SQL audit-trigger failure
с rollback link+epic+event+idempotency и subsequent retry/replay exactly once;
reopen источника с каскадом через три эпика без переписывания child audit;
implicit review cycle; cross-epic reviewer/tester до и после закрытия target
epic; negative → replacement positive; same-SHA stale → fresh generation;
T-9 approve → T-10 pending/negative → fresh positive; downstream Next после
superseded reviewer/tester source и явного relink; umask022/public parent0755,
fresh/live WAL/restart; DB и 3 sidecars × public/symlink/dangling;
publicly writable parent; SQLite URI foreign target unchanged; actual secret
entropy. Всё — synthetic store unit/integration, не process/E2E.

NOT-RUN этим исполнителем: полный make check, transport/CLI повторный прогон,
новые process/E2E, реальные Claude Code/Codex, две физические машины,
Linux/Windows, remote TLS и production acceptance. Владельцы — координатор,
отдельные reviewer/security и затем tester после актуального approve.

## Журнал

- 2026-10-18T12:55:28+0300: исходная R1/R2+SEC01 дельта и targeted store
  regression/race готовы; начальная версия ещё не frozen, ожидается единое
  уточнение координатора о latest applicable assignment.
- 2026-10-18T12:57:46+0300: latest-assignment и downstream invalidation
  включены по уточнению координатора, SQLite URI отвергается до I/O;
  финальный targeted race PASS. Результат передаётся независимым проверяющим.

## Frozen manifest SHA-256

Изменения незакоммичены; это идентичность 11 файлов авторской исходной дельты,
не approve и не manifest всей TASK. Документация контракта — область координатора.

```text
d85db430b4cc84173e35a09d1d6baf156cdd819a5f68b37d0a47f5667afc4aeb  internal/store/dependencies.go
d1ab7291bb546a3c94aa0911332c24974964688cc26a8e76cd85635312432a47  internal/store/epic_gates_test.go
716876bfcfccfc1083e6f587b9df2f5b5aa8a293584f0cf383e42c1f8f55763c  internal/store/files.go
bbb8cdca1de13291e485ce720b49d076462b94532ae8b0280e0a07ba9e46c521  internal/store/files_unix_test.go
6a0b04a96a8f6af6868d8b919abb77fe9696810c04783b5af0b3ef5bb8876c9a  internal/store/gates.go
36314a204102ff0a5cb31ea6243a0cd046a524081bf79a837a1eee7bc6a8ce29  internal/store/global_verification_test.go
1eddcf6006619d8baf99fe7a0384facd02c1ae726e717f43a703de880fd7ec16  internal/store/idempotency_test.go
6a27fbf7bac4aab86f55dd647ebcdb4bde189dd34ab4f074f2c4e2668561ed9c  internal/store/mutation.go
f05a9bbea274114cefbd24fe19eb6565a7faa0c35340780e2380c9f8a78bc8b3  internal/store/store.go
ccc12156e679522cfe2533e40b3f606a782def4623b54484f8aad4663b35b31b  internal/store/tasks.go
d646639e80b8b14b68c2385d9018d6710ef6ed76bbdb46ab8ce7b417f93ea1eb  internal/store/transitions.go
```
