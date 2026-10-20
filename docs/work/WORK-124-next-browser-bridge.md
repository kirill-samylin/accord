# WORK-124 — browser JSON bridge

Владелец: исполнитель 124. Область записи: `internal/web/**` и эта карточка.
TASK-007 ведёт координатор. Дата: 20.10.2026.

## Контракт и границы

Явный `Accept: application/json` выбирает JSON на существующих `/web/*`
страницах и POST формах; URL и обработчики авторизации/мутаций общие с SSR.
`/web/data/*`, `/web/static/*`, binary download/preview и native API/MCP
не меняются. `Vary: Accept` и `Cache-Control: no-store` сохраняют границу
представлений. JSON не принимает произвольный upstream или новый bearer.
Next и Go работают через same-origin fixed upstream `/web/**`; Go `--web-origin`
равен публичному Origin Next, proxy сохраняет Origin и Set-Cookie. Cookie Path `/`.

GET успех: `{ok:true,page:{name,view,title,notice?,csrf?,actions?,principal:
{owner,machine},session,can_continue,can_edit,can_update,draft?,field_errors?,overview?,
agents?,agent?,activity?,stats?,observed_at,agents_observed_at,activity_observed_at,
task?,tasks?,epics?,questions?,my_questions?,legacy_decisions?,designs?,decisions?,
next_cursor?,epics_next_cursor?,query?,agent_state?,epic_key?,role?,state?,roles?,
states?,categories?,complexities?}}`. Вложенные core DTO уже имеют snake_case.
Это allowlist projection Page, не persisted browserSession/action rows.
Principal ID не включается; существующие публичные core session/task provenance
сохраняются. Raw bearer, browser hash, pending payload отсутствуют.

POST успех: `{ok:true,redirect:"/web/..."}` с HTTP 200 вместо 303; heartbeat:
`{ok:true,session,next_action_id}`. Ошибка сохраняет исходный HTTP status:
`{ok:false,error:{code,message},page,draft,field_errors,action_id,actions,
retry_same_action}`. Неизвестный исход требует тот же action_id и поля;
подтверждённая validation выдаёт новые operation-bound actions для исправления.
Форматы POST остаются URLencoded и существующий упорядоченный multipart.
Upload сохраняет 50 МиБ, deadline 120 секунд и streaming body.

## Выполнение

Реализация готова к независимому ревью. Новые process/E2E N/A для автора:
отдельный тестировщик после независимого approve.

## Дополнение координатора: редактирование задачи/эпика

Координатор поручил существующий `Store.Update` открыть для browser forms.
`POST /web/tasks/{T-N|E-N}/update`: `csrf,action_id,title,description,generation`.
Меняются только заголовок и описание. Остальные присланные metadata fields
не переходят в native UpdateInput. Новая operation `update` входит в тот же
durable journal/runAction/recoverActions. HTML POST сохраняет 303; JSON — 200.

`page.can_update` и `actions.update` выдаются на detail только живой human
board-сессии: requester незахваченной задачи/эпика либо держатель своей живой
human lease. Для списков capability имеет ключ `update:{key}`. Никакого
обхода native requester/ownership/fencing нет: даже намеренно переиспользованный
operation-bound ID на чужом объекте отклоняет Store.Update.

Заголовок непустой после trim и ≤200 Unicode code points; описание ≤16384
байт, включая допустимое native пустое описание. `generation=0` для незахваченного
объекта, текущий generation для своей lease. Для update raw URLencoded body
ограничен 64 КиБ: допустимый 16 КиБ текст после URL encoding занимает до 48 КиБ.
Остальные browser формы сохраняют прежние 32 КиБ; upload предел не изменён.

## Изменённые файлы

- `internal/web/browser_json.go`, `browser_page.go`: content negotiation,
  allowlist Page projection, error envelope, JSON redirect representation,
  response controller Unwrap для upload deadlines.
- `http.go`, `design_http.go`, `mutations.go`: подключение представления к
  существующим обработчикам; login/logout/upload/core rules переиспользованы.
- `contracts.go`, `pages.go`, `actions.go`, `browser_update.go`: native update
  capability, title/description form input и durable execution/recovery.
- `browser_json_test.go`, `browser_mutation_test.go`, `browser_upload_test.go`,
  `browser_update_test.go`: новые unit/integration regression suites.

## Авторские проверки

Go `1.27.2 darwin/arm64`, `.local/toolchains/go/bin` (системный Go отсутствует).

- PASS: исходный web suite до расширения update.
- PASS: negotiated page mapping всех 13 представлений; preauth/login/logout,
  cookies, SSR fallback, explicit Accept q, HEAD, generic raw errors/no secrets.
- PASS: Origin/CSRF/revocation, нет CORS allow; controlled draft projection.
- PASS: create validation/fresh IDs, exact/changed replay, claim/heartbeat/
  answer/release, immutable decision после lost-result replay и expiry,
  чужая lease/stale generation, explicit continuation/replay.
- PASS: uncertain core commit сохраняет draft и тот же ID, recovery replay.
- PASS: multipart upload metadata/replay/changed file/validation/CSRF,
  deadline forwarding 120 секунд, 50 МиБ invariant, binary download unaffected.
- PASS: update requester task+epic, foreign requester deny, own human live lease,
  stale generation/foreign live lease denial; version/evidence/state/role/priority
  не подменяются; validation/16 КиБ Cyrillic/пустое описание/SSR redirect;
  durable unknown update/recoverActions/exact replay после expiry.

- PASS: финальный `.local/toolchains/go/bin/go test ./internal/web` (2.339s).
- PASS: финальный `.local/toolchains/go/bin/go test -race ./internal/web` (16.431s).
- PASS: `.local/toolchains/go/bin/go vet ./internal/web`, `git diff --check`
  owned paths, gofmt. Все новые файлы <300 строк; прежний `design_http.go`
  после узкого подключения JSON ровно 300 строк.
- PASS: `.local/toolchains/go/bin/go test ./...`, включая существующий native
  acceptance (17.013s); новая Board.Update interface совместима со всеми packages.

Независимый review/внешний QA ожидаются; авторский PASS не заменяет их.
Новых dependencies, remote/deploy, Tandem, соседнего проекта не затрагивали.
