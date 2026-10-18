# WORK-006 — Независимая локальная приёмка

- Родитель: TASK-001-pilot. Роль: тестировщик, экземпляр `acceptance`.
- Статус: на проверке; этап: новые process/E2E готовы к отдельному ревью.
- Область: `tests/acceptance/`, `docs/acceptance.md`, эта WORK.
- Прочитаны AGENTS, TASK, requirements/architecture/API/README/development,
  общий процесс и tester-профиль соседнего Давинчи. Соседний проект не менялся.
- Условие начала новых тестов: актуальный APPROVE `docs/review-recheck.md`;
  security recheck выдан другим независимым агентом. Тестировщик не автор
  реализации и не выдаёт своему новому набору независимый approve.
- Tandem отключён прямым указанием пользователя. Именованные Claude-модели
  недоступны; роль передана профилем, используется модель текущей среды.

## Результат

Добавлены четыре Go test-файла (каждый менее 300 строк), пять независимых
сценариев с реальными subprocess CLI/server и persistent SQLite. Нет прямых
вызовов Store, транспорт не подменён. Покрыты AC-01…07 из
[приёмки](../acceptance.md): bootstrap приватного токена/БД, describe/create/get,
два concurrent next с одним победителем; developer→architect dependency/link
с live generation/release/evidence/возвратом к той же задаче; настоящее
истечение TTL=1s по HTTP Date сервера и fencing; retry с discarded first
response и ровно одним create event; filtered pagination с append;
SIGTERM/restart той же БД с сохранением sessions/state/generation/replay;
local CLI revoke блокирует replay, новую мутацию и read.

Набор строит бинарник через `runtime.GOROOT()/bin/go` из root модуля,
хранит runtime-данные в приватных временных каталогах, ограничивает CLI
таймаутами. Cleanup сигнализирует только собственному PID сервера; forced
kill — FAIL. Нет `pkill`, чужие процессы не затрагиваются, токены/ответы
ошибок не попадают в test-логи. Наличие артефакта архитектора не выдумано:
проверяется только фактическое сохранение синтетической evidence-ссылки.

## Проверки и версия

PASS: финальный targeted `go test ./tests/acceptance -count=1 -json`,
5/5, пакет 3.553s, exit 0, 18.10.2026 13:04:49…13:04:53 +03:00.
Окружение Go 1.27.2 / macOS arm64; env из development.md.
PASS: gofmt; все 46 SHA frozen source reviewer совпадают до запуска.
Исходный base HEAD `9537cb737ee3d1f1e3d63eeccc6a81e2fa681698`, source manifest
digest `9cc3437a3f55c5a46859d8610997fb7e14ceb0a0cbcfc3fcd3864d2b03d006b4`.
Точная версия новых тестов, SHA каждого evidence и фактический RUN —
в `docs/acceptance.md`; сохранён проверенный бинарник SHA
`af7dacd5b6ee86d5bf4712985de793e90055d4f61f9db5c3104dc1ab28e1aa2d`.

NOT-RUN: общий make check (координатор), process MCP, реальные Claude/Codex,
два ПК, Linux/Windows, TLS/OAuth/load/production, crash/network/disk fault
injection, отдельный ручной прогон. UI N/A. Обнаруженных дефектов реализации
в этом наборе 0; это ограниченное evidence локального пилота, не V1 acceptance.
Source, go.mod/sum, README и общая TASK не менялись. Push/PR/merge/release нет.

## Журнал и передача

- 2026-10-18T13:03:50+03:00: observed checkpoint — четыре новых test-файла
  созданы после актуального code APPROVE; source manifest сверен 46/46,
  начат preliminary targeted прогон. Фактическое начало работы было раньше
  этой зафиксированной отметки; время задним числом не назначается.
- 2026-10-18T13:04:58+03:00: completed, в работе → на проверке, исполнитель
  `acceptance`: финальный targeted PASS 5/5, cleanup без ошибок, бинарник/
  JSONL/manifest сохранены `.local/evidence/acceptance/`; coordinator получил
  frozen набор для отдельного test-review. Доработки кода не потребовались.

Следующий исполнитель: отдельный reviewer новых tests/acceptance и отчёта.
После approve координатор выполняет один общий make check и решает общую
приёмку TASK. Без этого независимое ревью нового E2E-набора не завершено.
