# Локальная приёмка пилота 0A

Дата прогона: 18.10.2026. Исполнитель: отдельный тестировщик
`acceptance / WORK-006`, не автор реализации. Состояние: **5/5 process-тестов
PASS; новый набор передан на независимое ревью**. Это не завершение V1.
Основание начала новых E2E — актуальный [code APPROVE](review-recheck.md);
профильный [security APPROVE](security-recheck.md) выдан отдельно.

## Набор и соответствие требованиям

Кейсы ниже — инструкция; результаты конкретного запуска находятся отдельно
в таблице протокола. Все данные синтетические. Сервер, CLI и SQLite настоящие:
бинарник собирается один раз, каждая CLI-команда — новый процесс. Хранилище
напрямую тестами не вызывается. Один сервер на одну временную БД, адрес
выделяется ОС на loopback, готовность — строка адреса плюс health/ready.
Токены — новые приватные файлы 0600; временные каталоги 0700.

| TC | REQ | Автотест | Действия и ожидаемый результат |
| --- | --- | --- | --- |
| AC-01 | 02,04,05,08,09 | TestCLIConcurrentClaim и общий bootstrap | Собрать CLI; создать два токена в файлы; открыть разные board-сессии; создать epic/task, прочитать задачу. Файлы токенов и SQLite 0600, секрет отсутствует в stdout token create; epic open, task ready. |
| AC-02 | 06,09,18 | TestCLIConcurrentClaim | Одновременно запустить два независимых CLI next для одной задачи, с разными токенами/сессиями. Один task=null, один владелец; get показывает ровно одну попытку и того же владельца. |
| AC-03 | 11,12,13,18 | TestCLIArchitectureHandoff | Developer создаёт запрос architect; stale generation link отклоняется; текущий link contract_of проходит до release. После release next developer пуст. Architect сам захватывает запрос, завершает с artifact+summary. Developer захватывает исходную задачу с поколением +1 и сохранённой branch. |
| AC-04 | 18,19,20 | TestCLILeaseExpiry | Настоящий сервер task TTL=1s, session TTL=1m. Дождаться HTTP Date сервера за expires_at. Старый heartbeat → LEASE_LOST; другой developer получает generation+1; старые comment/heartbeat не меняют нового владельца. История содержит lease_expired, комментарий не записан. |
| AC-05 | 22,23 | TestCLIRetryAndFilteredPagination | Отбросить первый успешный ответ create, повторить те же байты request_id/payload. Два повтора идентичны, task_created один; другой payload → IDEMPOTENCY_CONFLICT. Читать events limit=1 с фильтром, между страницами добавить задачу; все подходящие события встречаются один раз. Task-фильтр и пустой scan корректны. |
| AC-06 | 05,11,18,22,23 | TestCLIRestartAndRevocation | Release с branch; сохранить session/create ответы и курсор; SIGTERM, успешный exit, новый процесс на той же SQLite. Describe/create replay идентичны, дополнительных событий нет; get сохраняет state/generation/attempt/branch; новый next увеличивает generation. |
| AC-07 | 04,22 | TestCLIRestartAndRevocation | CLI token revoke для ID координатора. Идентичный replay create, новая create и get → UNAUTHORIZED. |

AC-05 — **явная симуляция неопределённости клиента отбрасыванием полного
первого ответа**. TCP-разрыв, потеря сети и авария во время SQLite commit
этим не моделируются. Artifact архитектора — синтетическая ссылка: сервер
проверяет форму evidence, наличие файла по этой ссылке здесь не проверяется.
Два board-run/CLI процесса на этом Mac не являются двумя физическими ПК.

## Воспроизведение автоматического набора

Из корня checkout, с runtime/cache из [development.md](development.md):

```sh
export GOTOOLCHAIN=local
export GOCACHE="$PWD/.local/toolchains/cache/build"
export GOMODCACHE="$PWD/.local/toolchains/cache/mod"
export GOTMPDIR="$PWD/.local/toolchains/tmp"
.local/toolchains/go/bin/go test ./tests/acceptance -count=1 -v
```

Для ревью можно выбрать 2–3 сценария без нового полного прогона:

```sh
.local/toolchains/go/bin/go test ./tests/acceptance -count=1 -v \
  -run 'TestCLI(ConcurrentClaim|LeaseExpiry|RestartAndRevocation)$'
```

По умолчанию временные бинарник/БД/токены удаляются после тестов. Каждый
сервер получает SIGTERM только по собственному PID, затем ожидание до 7s;
принудительный kill при превышении срока считается ошибкой очистки. CLI
ограничены 12s, сборка — 2m. Ожидание TTL опрашивает HTTP Date сервера до 6s,
не подставляет часы клиента и не использует произвольный sleep.
Опциональный `ACCEPTANCE_EVIDENCE_DIR` сохраняет только собранный бинарник;
токены и БД туда не копируются. Повторное использование каталога заменит
бинарник, поэтому для нового зачётного RUN выбирайте новый каталог.

## Шаги ручного воспроизведения

Этот ручной протокол отдельно **NOT-RUN**; автоматический прогон выше PASS.
Используйте чистый каталог и только синтетические имена. Команды ниже
выполняются из корня checkout. Значения `SESSION`, `EPIC`, `TASK`, `GEN`
заменяйте данными предыдущих ответов, без угаданных номеров T-N/E-N.

1. Выполните `make build`, затем в первом терминале:

   ```sh
   umask 077
   mkdir .local/manual-acceptance
   bin/agent-board token create --db .local/manual-acceptance/board.db \
     --owner manual-a --machine manual-a --output .local/manual-acceptance/a.token
   bin/agent-board token create --db .local/manual-acceptance/board.db \
     --owner manual-b --machine manual-b --output .local/manual-acceptance/b.token
   bin/agent-board serve --db .local/manual-acceptance/board.db \
     --listen 127.0.0.1:8087 --task-ttl 15m --session-ttl 1h
   ```

   Сохраните ID токенов из stdout. Секреты не читайте в терминал. В другом
   терминале `curl --fail http://127.0.0.1:8087/readyz` должен завершиться успешно.

2. В рабочем терминале создайте короткие функции вызова:

   ```sh
   a() { bin/agent-board call "$1" --server http://127.0.0.1:8087 --token-file .local/manual-acceptance/a.token; }
   b() { bin/agent-board call "$1" --server http://127.0.0.1:8087 --token-file .local/manual-acceptance/b.token; }
   ```

   Например, `a describe <<'JSON'`, затем строка JSON из таблицы, затем
   отдельная строка `JSON`. Для остальных операций используйте тот же способ.
   Для каждого нового изменения берите новый request_id. Чтения не требуют его.

   | Вызов | JSON, где подставляются только полученные идентификаторы |
   | --- | --- |
   | a describe | `{"request_id":"open-coord","role":"coordinator","run_id":"coord","machine":"manual-a"}` |
   | a describe | `{"request_id":"open-a","role":"developer","run_id":"dev-a","machine":"manual-a"}` |
   | b describe | `{"request_id":"open-b","role":"developer","run_id":"dev-b","machine":"manual-b"}` |
   | a create | `{"session_id":"COORD","request_id":"epic","type":"epic","title":"Manual pilot","description":"Synthetic acceptance"}` |
   | a create | `{"session_id":"COORD","request_id":"task","type":"task","epic_key":"EPIC","role":"developer","title":"Slice","description":"Synthetic task"}` |
   | a get | `{"session_id":"COORD","task_key":"TASK"}` |

   Сохраните COORD, DEV_A, DEV_B и EPIC/TASK. Ожидается open/ready.

3. AC-02: сохраните следующие два JSON в `.local/manual-acceptance/a.json`
   и `b.json`, заменив session_id на DEV_A/DEV_B соответственно:
   `{"session_id":"SESSION","request_id":"race","task_key":"TASK"}`.
   Запустите в одном терминале два отдельных процесса:

   ```sh
   a next < .local/manual-acceptance/a.json > .local/manual-acceptance/a-result.json &
   b next < .local/manual-acceptance/b.json > .local/manual-acceptance/b-result.json &
   wait
   cat .local/manual-acceptance/a-result.json .local/manual-acceptance/b-result.json
   ```

   Ровно один task непустой. Назовите его сессию WIN, поколение GEN;
   используйте токен победителя в дальнейших developer вызовах.

4. AC-03: победитель создаёт architect task в том же EPIC (как create выше,
   но session_id=WIN, request_id=question, role=architect). Сохраните QUESTION.
   Победитель `link`: `{"session_id":"WIN","request_id":"link",
   "from":"QUESTION","to":"TASK","type":"contract_of","generation":GEN}`.
   Затем `release`: `{"session_id":"WIN","request_id":"release",
   "task_key":"TASK","generation":GEN,"reason":"Need architecture",
   "branch":"synthetic/manual"}`. Новый next на TASK должен вернуть task=null.
   Откройте через a describe architect с machine=manual-a и отдельными
   request_id/run_id. Architect сам вызывает next на QUESTION и transition
   `{"session_id":"ARCH","request_id":"decision","task_key":"QUESTION",
   "generation":ARCH_GEN,"state":"done","evidence":{"artifact":["docs/manual.md"],
   "summary":"Synthetic decision"}}`. Developer next с новым request_id
   получает исходный TASK, generation=GEN+1, attempt с указанной branch.

5. AC-05: сохраните create нового developer task в отдельный JSON с
   session_id=COORD, request_id=retry и полученным EPIC. Выполните
   `a create < файл > /dev/null`; дважды повторите тот же файл, ответы
   должны совпасть. Events через a с
   `{"session_id":"COORD","since_cursor":0,"limit":1,"types":["task_created"]}`:
   продолжайте с возвращённым next_cursor до has_more=false, не меняя фильтр.
   Создайте ещё задачу между страницами, проверьте отсутствие пропусков/дублей.
   В отдельном проходе добавьте task_key созданной retry-задачи: событие одно.
   Измените title в исходном retry JSON — ожидается IDEMPOTENCY_CONFLICT.

6. AC-06: освободите живую developer аренду с новым request_id и branch.
   Сохраните get/events, остановите сервер Ctrl-C в его терминале, дождитесь
   выхода и запустите ту же команду serve на той же БД. Старые сессии остаются
   применимы до их срока; describe/create replay возвращают сохранённые
   ответы. Get сохраняет state/generation/attempt/branch; следующий claim
   увеличивает generation. Сверяйте курсор events: replay не добавляет событие.

7. AC-04: остановите сервер и запустите на той же БД с `--task-ttl 1s`.
   Освободите любую ещё живую задачу и получите новый claim. Сверяйте HTTP
   Date из `curl -sI http://127.0.0.1:8087/healthz` с lease.expires_at,
   пока серверное время не станет больше. Старый heartbeat с task_key/GEN
   получает LEASE_LOST. Другой developer получает generation+1; старый
   comment с text и GEN также LEASE_LOST, get не содержит этот комментарий.

8. AC-07: `bin/agent-board token revoke --db .local/manual-acceptance/board.db
   --id ID_ТОКЕНА_A`. Через a повторите исходный create/retry и новый create,
   затем get: все получают UNAUTHORIZED. Остановите сервер Ctrl-C. Удалите
   только созданный здесь `.local/manual-acceptance`, убедившись в пути.

## Протокол RUN-20261018-acceptance

Окружение: macOS/darwin arm64, Go 1.27.2. База HEAD
`9537cb737ee3d1f1e3d63eeccc6a81e2fa681698`, незакоммиченная реализация.
Проверенная source-версия — все 46 файлов [manifest reviewer](review-recheck.md),
digest текста `9cc3437a3f55c5a46859d8610997fb7e14ceb0a0cbcfc3fcd3864d2b03d006b4`;
перед запуском каждый SHA сверен успешно. Сохранён именно проверенный бинарник.

Финальный запуск: 2026-10-18T13:04:49.539961+03:00 —
13:04:53.093035+03:00, exit=0. Команда после переменных окружения выше:

```sh
ACCEPTANCE_EVIDENCE_DIR="$PWD/.local/evidence/acceptance" \
  .local/toolchains/go/bin/go test ./tests/acceptance -count=1 -json \
  > .local/evidence/acceptance/results.jsonl
```

| Проверка | Результат | Время |
| --- | --- | --- |
| RetryAndFilteredPagination / AC-05 | PASS | 0.51s |
| RestartAndRevocation / AC-06,07 | PASS | 0.25s |
| ConcurrentClaim / AC-01,02 | PASS | 0.16s |
| ArchitectureHandoff / AC-03 | PASS | 0.24s |
| LeaseExpiry / AC-04 | PASS | 1.39s |
| Пакет, включая build TestMain | PASS | 3.553s |

Открытых обнаруженных дефектов реализации в этом наборе: 0. Cleanup PASS:
все тесты завершились без принудительного kill и ошибок shutdown. До финального
запуска preliminary `-v` тоже PASS (3.626s); после него добавлена проверка
приватности SQLite-файла и убран дубль декодирования ошибки в самих тестах.

Идентичность нового набора (frozen для независимого test-review):

```text
8aee530a004addba47401fa79f88e0f56742bde77817dcdf32ca8e22b11a3a66  tests/acceptance/client_test.go
617181d760b24637da6658e600545d2f27f3f122b15b0f0f38c4b628e330c308  tests/acceptance/process_test.go
06abc9b4b5f65090dfe53238975163b5114178109368456f50f7e452426bbd3e  tests/acceptance/recovery_test.go
469817a0bc901cd8d6d0e5e4bf69d8f374e6027b064a9a64db6dce109bb19a13  tests/acceptance/workflow_test.go
```

Все доказательства локальные, игнорируются Git, префикс `.local/evidence/acceptance/`:

| Файл | SHA-256 |
| --- | --- |
| agent-board — проверенный бинарник | af7dacd5b6ee86d5bf4712985de793e90055d4f61f9db5c3104dc1ab28e1aa2d |
| source-manifest.sha256 | 9cc3437a3f55c5a46859d8610997fb7e14ceb0a0cbcfc3fcd3864d2b03d006b4 |
| tests-manifest.sha256 | 4e24eacc793aaa24ea44e1faf1efd621139b97f47e61419d91b6b40a4949a6f8 |
| results.jsonl — финальные фактические события Go test | 945bab53bedfe193a570a555281f1f35c36804511d8ea412d50a2e71914aebcf |
| targeted.txt — preliminary вывод | a3023a7ad13fbca338279723f5b6e59a63345dbf450757c77ee62288b54161ef |

## Ограничения и следующий шаг

NOT-RUN этим исполнителем: полный `make check` (владелец координатор),
MCP process/E2E, настоящие Claude Code/Codex, два физических ПК, Linux/Windows,
remote TLS/proxy, OAuth, нагрузка, production, TCP/crash/disk fault injection,
отдельный ручной прогон. Существующие MCP SDK/store regression подтверждены
другими WORK и не выдаются за собственный E2E этого тестировщика. UI — N/A,
в пилоте нет интерфейса. Push/PR/merge/release не выполнялись.

Следующий шаг — независимое ревью этих тестов с 2–3 целевыми воспроизведениями,
затем один общий check координатора и сведение критериев TASK. До этого
WORK-006 на проверке; общий пилот/V1 завершёнными не объявляются.
