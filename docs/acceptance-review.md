# Независимое ревью process/E2E — WORK-011

Дата: 18.10.2026. Вердикт: **APPROVE** нового набора
`tests/acceptance/` и достоверности [отчёта тестировщика](acceptance.md).
Блокирующих замечаний нет. Этот вердикт завершает ревью новых тестов,
а общая приёмка TASK-001 и полный make check остаются координатору.

Проверяющий `acceptance_review` — отдельный агент, не автор реализации и
не автор WORK-006. Прочитаны AGENTS, TASK, требования, архитектура, API,
development, актуальные code/security заключения, общий процесс и профиль
reviewer. Tandem отключён прямым указанием пользователя. Именованные модели
Claude недоступны; автор и reviewer используют одну доступную модель среды.
Source и тесты не исправлялись; соседний проект не менялся.

## Проверенная версия

Base HEAD: `9537cb737ee3d1f1e3d63eeccc6a81e2fa681698`; реализация и тесты
незакоммичены. Фактические SHA всех 46 файлов source manifest совпали с
актуальным [code APPROVE](review-recheck.md). Копии
`.local/evidence/review-recheck/manifest.sha256` и
`.local/evidence/acceptance/source-manifest.sha256` побайтово идентичны,
SHA-256 обеих: `9cc3437a3f55c5a46859d8610997fb7e14ceb0a0cbcfc3fcd3864d2b03d006b4`.

Test manifest `.local/evidence/acceptance/tests-manifest.sha256` содержит
ровно все четыре Go-файла нового пакета, SHA каждого совпал.
SHA-256 самого manifest:
`4e24eacc793aaa24ea44e1faf1efd621139b97f47e61419d91b6b40a4949a6f8`.

```text
8aee530a004addba47401fa79f88e0f56742bde77817dcdf32ca8e22b11a3a66  tests/acceptance/client_test.go
617181d760b24637da6658e600545d2f27f3f122b15b0f0f38c4b628e330c308  tests/acceptance/process_test.go
06abc9b4b5f65090dfe53238975163b5114178109368456f50f7e452426bbd3e  tests/acceptance/recovery_test.go
469817a0bc901cd8d6d0e5e4bf69d8f374e6027b064a9a64db6dce109bb19a13  tests/acceptance/workflow_test.go
```

Отчёт WORK-006 также сверен: `docs/acceptance.md` SHA-256
`2ff1c1b7b4a14c569614c5195aa7cfdbfb9753cb2dec30a9bee5b2404ca943c7`;
`docs/work/WORK-006-acceptance.md` SHA-256
`ef77b82e1fbbbdaf7d9f7bcb5b1623c86f4d95110afeef5bbb19e48101846283`.
Их mapping и ограничения соответствуют тестам и фактическому JSONL.

## Содержание проверки

Тесты собирают настоящий `cmd/agent-board`, запускают отдельный сервер,
выполняют каждую CLI-команду новым OS-процессом и используют настоящий
loopback HTTP и SQLite-файл. Store не вызывается напрямую; mocks, внедрённые
часы и подмена transport отсутствуют. Общие core DTO используются для
декодирования wire-ответов, бизнес-алгоритм в assertions не воспроизводится.

| Тест | Проверенное поведение |
| --- | --- |
| TestCLIConcurrentClaim | Два отдельных токена и server-generated session; общий старт двух CLI next; ровно один победитель. Последующий get подтверждает одного владельца и одну сохранённую попытку. |
| TestCLIArchitectureHandoff | Неверное generation не меняет link; живой developer создаёт dependency до release. Next блокируется до самостоятельного claim/done архитектора; evidence сохраняется, developer возвращается к той же задаче с новым generation и прежней branch. |
| TestCLILeaseExpiry | Реальный task TTL=1s. Ожидание HTTP Date за expires_at, отказ heartbeat до повторного claim, generation+1 у другого developer; stale comment/heartbeat отклоняются, комментарий не записан, попытка закрыта lease_expired. |
| TestCLIRetryAndFilteredPagination | Первый полный успешный ответ намеренно отбрасывается; одинаковые повторы возвращают сохранённый ответ, конфликт payload отвергается, create event один. Limit=1 с фильтром и добавлением события между страницами сохраняет полный набор без дублей; task filter и пустой scan проверены отдельно. |
| TestCLIRestartAndRevocation | SIGTERM и успешный exit, новый server на той же SQLite. Session/create replay, события, state, attempts, branch и generation сохраняются; следующий claim увеличивает generation. Реальный CLI revoke блокирует replay, новую мутацию и read. |

Это assertions внешнего контракта: ослабление fencing, дублирование событий,
потеря сохранённого ответа/истории или пропуск события приводят к FAIL.
Process-набор дополняет существующие store/transport проверки и не заявляет
собственного process-покрытия всех вариантов gates и MCP.

Приватные синтетические fixtures: отдельный `t.TempDir` 0700 на сценарий,
token/DB 0600, исходные bearer не выводятся и не передаются аргументом CLI.
Вывод дочернего сервера не пересылается; сообщения ошибок не печатают bodies.
Cleanup посылает SIGTERM только сохранённому собственному PID; превышение 7s
и вынужденный kill дают FAIL. `pkill` и воздействий на чужие процессы нет.

ОС выделяет порт через `127.0.0.1:0`; адрес читается из server stdout,
затем проверяются health/ready. Сборка ограничена 2m, CLI — 12s, получение
адреса — 10s, HTTP readiness — 3s, ожидание TTL — 6s. Polling TTL опирается
на серверное время и deadline; произвольных sleep нет. Пагинация ограничена
счётчиком страниц. Разделение helpers/process/workflow/recovery осмысленное,
новых дублирующих реализацию helpers нет. Файлы: 176/174/122/128 строк,
все менее 300; `gofmt -l tests/acceptance` не выдал файлов. UI — N/A.

## Фактические доказательства

Зачётный RUN тестировщика: `.local/evidence/acceptance/results.jsonl`,
SHA-256 `945bab53bedfe193a570a555281f1f35c36804511d8ea412d50a2e71914aebcf`.
Проверены события JSONL: ровно перечисленные пять top-level тестов PASS,
пакет PASS 3.553s, FAIL/SKIP отсутствуют. Preliminary `targeted.txt` относится
к предыдущему прогону до небольшой правки tests и не используется вместо
финального JSONL.

Reviewer воспроизвёл только три сценария, с окружением development.md:

```sh
ACCEPTANCE_EVIDENCE_DIR="$PWD/.local/evidence/acceptance-review" \
  .local/toolchains/go/bin/go test ./tests/acceptance \
  -run '^(TestCLIConcurrentClaim|TestCLILeaseExpiry|TestCLIRestartAndRevocation)$' \
  -count=1 -json > .local/evidence/acceptance-review/selected-results.jsonl
```

PASS: `TestCLIRestartAndRevocation` 0.65s, `TestCLIConcurrentClaim` 0.16s,
`TestCLILeaseExpiry` 2.18s; пакет 3.907s, exit=0, FAIL/SKIP нет.
Время: 18.10.2026 13:06:19…13:06:23 +03:00, macOS arm64, Go1.27.2.
SHA-256 `selected-results.jsonl`:
`999646264d3c5769d3063f545ecfb149330b020c5734fccfab26c89d4112118f`.
Повторно собранный binary reviewer побайтово совпал с сохранённым binary
tester: `af7dacd5b6ee86d5bf4712985de793e90055d4f61f9db5c3104dc1ab28e1aa2d`.
Сводная проверка manifests/JSONL/binaries —
`.local/evidence/acceptance-review/verification.json`.

## Ограничения и передача

Проверены локальные CLI-процессы на одном Mac. Две физические машины,
реальные Claude Code/Codex и MCP process/E2E этим набором не проверены.
Discarded first response — честная симуляция неопределённости клиента;
TCP-разрыв и crash во время SQLite commit не моделируются. Graceful restart
не выдаётся за аварийную сохранность при отказе диска. Evidence архитектора —
синтетическая ссылка, доступность файла не заявлена.

NOT-RUN reviewer: весь новый набор повторно, общий `./...`/make check/race,
другие ОС, remote TLS/proxy, OAuth, нагрузка, production и ручной протокол.
Это не блокирует approve данного локального testset и не закрывает внешние
этапы пилота/V1. Push/PR/merge/release не выполнялись. Координатор выполняет
единый make check и сводит общую приёмку; после изменения frozen source/tests
нужно заново проверить затронутую область. Следующий этап — новый агент.
