# WORK-091 — независимое security review веб-Accord

- Роль: security, отдельный от авторов 087/088/089/root агент `/root/websecurity091`.
- Родитель: TASK-004. Статус: выполнено. Вердикт: **APPROVE реализации**.
- Проверена замороженная незакоммиченная версия в `.local/web-service-084`,
  ветка `codex/web-service`, HEAD/base `f06338089bec3b8c7f17f541c463131f5c98f96c`.
- Контроль версии: 2026-10-19T18:15:43+00:00; получен FREEZE-088 и freeze остальных source от координатора.
- Запись только эта WORK. Source, Git, Tandem, соседний Давинчи и live не изменялись.
- Профиль security и общий/parallel процесс Давинчи прочитаны как инструкции роли;
  AGENTS отдельного Accord отменяет Tandem для этой задачи. Именованные Claude
  agents/model bindings недоступны; использована унаследованная модель, без смены.

## Объём и модель угроз

Прочитаны актуальные web requirements, ADR-W1..4, SEC-W01..06, actual contracts.go,
изменения auth/store/history/transport/MCP bridge, browser backend, file storage,
SSR/JS, публичный nginx renderer, offline backup helper и tests. Входы недоверенные:
HTTP headers/cookie/form/multipart, имена и содержимое дизайна, текст пользователя
и агента. Активы: bearer и cookie, board leases/generation/replay, история ответов,
private web DB/blobs, доступность существующего сервиса и согласованность backup.
TLS proxy, OS identity/root и единственный writer остаются границами доверия.

**Новых блокирующих находок нет.** Approve относится к source ниже и не заменяет
процессную/браузерную приёмку или проверку live-выпуска.

## Результат проверки условий SEC-086

- SEC-W01: Extra стоит после admission/Host/native-Origin guards; native API/MCP/
  metrics сохраняют bearer и запрет любого Origin. Нет расширения 12 MCP tools;
  новые resources возвращают metadata/authorized URL и проверяют current actor.
  Публичный renderer оставляет plaintext без proxy, metrics private, native64KiB,
  фиксированный upstream Host; web upload получает отдельный лимит и120s.
- SEC-W02: случайная cookie256bit, только hash/tokenID в webDB, Secure/HttpOnly/
  Strict/Path=/ и без Domain; login rotation, preauth consumption, absolute expiry,
  logout и current AuthenticateID проверены. Native mutations сохраняют
  authenticatePrincipal внутри board-транзакции до idempotent replay.
- SEC-W03: exact single Origin и CSRF до mutation; multipart upload проверяет
  bounded первые csrf/action поля до storage bytes. Login body16KiB/token4096,
  ровно один источник, секрет не возвращается в Page. Атомарный limiter5/мин,
  глобальный fallback без доверия forwarded headers, bounded preauth/buckets.
  SSR autoescape без template.HTML/JS/URL; JS использует textContent и безопасные
  локальные ссылки. CSP без unsafe-inline/eval, no-store/nosniff/no-referrer.
- SEC-W04: server-generated human session, адресный Next, native Transition/
  Release/Heartbeat и generation; cached/uncertain retry сохраняет исходный ID,
  GET не делает claim/renew. Human answer/provenance сохраняется immutable event
  в той же board transaction и exact replay не добавляет второе событие.
- SEC-W05: bounded streaming50MiB+64KiB, два upload, preview decoder один,
  атомарный полный резерв, quota1GiB и disk floor512MiB. O_EXCL/O_NOFOLLOW,
  descriptor roots, private modes/owner, случайные storage names. fsync/publish/
  metadata порядок, fail-closed ready integrity и pending/orphan recovery.
  PNG/JPEG проверяются config+bounds до полного Decode, preview перекодируется;
  WebP attachment-only согласно согласованному ADR, без новой зависимости.
  HTML/SVG/PDF/ZIP всегда attachment/octet-stream+sandbox/nosniff.
- SEC-W06: offline snapshot останавливает writer под maintenance lock, копирует
  board+web+blobs/staging, проверяет refs/checksums/schema, fsync и atomically
  publishes до retention. Restore guide требует web-auth reset и существующий
  native revoke-all recovery до сети. Rollback не восстанавливает старую boardDB.

## Фактические проверки

PASS — независимый запуск после интеграции:

```text
go test ./internal/web ./internal/webui ./internal/store ./internal/transport ./internal/bridge
web 1.900s; webui cached; store 2.015s; transport/bridge cached
```

PASS — независимый принудительный прогон после финального FREEZE:

```text
go test -race ./internal/web ./internal/store ./internal/transport ./internal/bridge -count=1
web 8.844s; store 9.933s; transport 3.222s; bridge 13.789s
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tests/infra -p 'test_web_*.py' -v
9 tests / 0.140s / OK
```

Использован существующий Go toolchain основного Accord:
`.local/toolchains/go/bin/go`; GOTOOLCHAIN=local; GOCACHE/GOMODCACHE/GOTMPDIR
в `.local/toolchains/cache/build`, `cache/mod`, `tmp`. Source/tests вне sandbox
читались и запускались с task-authorized escalation; credentials не использованы.
Тесты проверяют auth/revoke/expiry, Origin/CSRF, concurrent limiter, fencing/replay,
immutable answers, 50MiB/+1, quota/disk/concurrency, multipart boundaries, symlinks,
preview bounds/corruption, restart/recovery, transport and native regressions.

Дополнительная targeted synthetic проверка замечания reviewer090: импорт actual
`infra/accord/web/backup.py`, mock только systemctl/private lock/signal registration;
вызов actual `interrupted(SIGTERM,None)` во время `stop` и отдельно `snapshot`.
Оба сценария PASS: `is-active -> stop -> start`, RuntimeError не проглочена.
Настоящие systemctl/сервис/сервер не вызывались. Исправление TERM/INT→exception и
finally restart подтверждено; зависание/принудительный SIGKILL не моделировались.

NOT-RUN в этой WORK: реальный listener slow stream>30s/120s, nginx syntax/runtime,
actual HTTPS cookie/proxy/browser journey и mobile/keyboard; целый процессный
backup→restore→native revoke-all и запуск старого бинарного файла для rollback.
Старый shape-read тест проверен, но не заменяет запуск old binary. Новые process/
E2E по процессу выполняет отдельный tester после code/security approve.
Live VPS/Tandem и два физических ПК не проверялись. Deploy не выполнялся.
N/A: OAuth/enterprise RBAC/virus scanning/распаковка ZIP/изоляция от same UID/root
и pixel-match без макета — вне согласованного среза. Download остаётся недоверенным
содержимым для получателя. Backup на том же VPS не является offsite protection.

## Точная проверенная версия

Ниже SHA256 каждого изменённого/new runtime или test файла и трёх входных
контрактов. Старые неизменённые файлы определяются base commit. SHA256 всего
блока manifest (строки с двумя пробелами, LF включая последний): `35536e070e5577ddbc7bbc0d80e2f90a3bf494cc9d25ba9424d30d382c971e48`.

```text
0b72e465bfdd947751d1efcefa920a3c564f7456c86b176433795158171eb5dd  cmd/agent-board/main.go
ae2c53cf73ef8ca6dae417e000e04b28a12a44d357c0611163723bafdb55a3ee  infra/accord/public/render_public.py
8f7138408f14e7d73226fa2872c15eb9447268f4c4c12f2949bcdd51b573a22d  infra/accord/web/README.md
55e38437bcc6368f386fd2c8974e8f307553af36127c204a6c5f9dbfb4d7a37f  infra/accord/web/accord-web-backup.service
c539a950f552d53573417f5da7549a63b39275312258607c6ab1d52b976f7a85  infra/accord/web/accord-web-backup.timer
282b297caf06af34fba9b8259d76e3901b9281c877ba1d3f109a381dd4128a4e  infra/accord/web/backup.py
c3eae44a95a94de3073a78a4f930320ac7deb269bb61aa17dd2a9b7fb4e40ea0  internal/bridge/resources.go
d79cad37d7f494597c6ef81ba8292f9ebb00ea6c9ef57882951dc4a1e3b21801  internal/core/catalog.go
23415692b8ca6aafeafab655ef7c3e23073fad3540653e5110125d5eb545d079  internal/core/decisions.go
a1e09816eda6f1335dce1abedfc1aa28579e5409493a8de6cb172f8e0beac2b9  internal/store/transitions.go
db0bee6024a9bc71e6ec3999af6a6dfa3f0f19e152dc7a06f338b12edcbe2c52  internal/store/web_auth.go
964183f750081b55665578ab37f55d7bf6c5dd3138a5fe1d84105e6b860dd8cd  internal/store/web_auth_test.go
501a50a8ea1c28b205765ab17e8ca3f2b84ae472b129feb6df3ce335df81aff4  internal/store/web_history.go
1e0dd4ab7bdf8a940a4c577731ebc91398aca248126f7bea65dca7a9e0a32c53  internal/store/web_history_test.go
58625c6e33a751eca1127b026af09c13b157dc4256528a46c3c3403676100b7e  internal/store/web_questions.go
41259315e79bb0eac6458e3c65d2aece6d7f7ecf9a13da9ba257f3290d0a1913  internal/store/web_questions_test.go
34418d1d1ab4d92a407b6a0012f69249d3a1350a6329ee1f30bd6a7ea90d3f95  internal/transport/design_resources.go
d86db1b3d78b648228621bfc653fd84156145b25f4b93652a46248de209a1e99  internal/transport/http.go
37979bf98a086f4c4517e712c064bf866140a046d2d6bc45fae7808f2ad22b68  internal/transport/http_security.go
7226acf9f754f5b7d2593b3cba6ae69c8d35530c700fecf0a879e38fc8333f32  internal/transport/mcp.go
dbd528069e3e9eed1ad7c1310f83d57036a36bfc456532c7ccb8aeaa20d25f0b  internal/transport/web_integration_test.go
7ac21452e79e8215ec4e0f43fefefe0065ff041bc3d54d549313d6fddfda4ab8  internal/web/actions.go
14fe9c40857ac53a042730b6544d7be537071f2d0b5b0349f07ec24dc5ff07be  internal/web/backend_test.go
8bbb6dff33f511f65399b4c84f5a6c3b8ca00f7374ede895df1c872b52889590  internal/web/contracts.go
643ff6dc665d08aa85c13880fb4806c09f5e858c93355a7a32ab5d69094859ce  internal/web/design_http.go
c2866f9d1d1354ca12e83adc9d9b0f4a698408f37e0685c0db98fe9c5ff87775  internal/web/design_read.go
9680a1fe9b9210cd6c7ae9fb24251f69b714a7a64d7c67bd699f9ec38ef7c70e  internal/web/design_store.go
66c01807f43db75e096267516a167d5c258a4d14405b37becbd24f22ca941497  internal/web/files_budget.go
8fc4b348e0beb02f8d670ec24f47413dc96cd3142429edbdc0055fa2f71fcfd4  internal/web/files_cursor_test.go
4babc3ffb1686765e33b93c065defd907fdeab42e0bf288e84926a0aa2315f54  internal/web/files_form.go
93e4ed1c26f006d8fa0b2c357e804a9871d50b9008b6b22cf539b1673ece3cba  internal/web/files_format.go
2e65e4c005e930add6437f692a2c1748212739091b469a0ccf660ab6af62c45e  internal/web/files_integration_test.go
ee5a141d6ac4cbff610232326778b4a0fd80962b631259c28c98de60b757297c  internal/web/files_paths.go
042633c7ac95e16222fbfa049e7e94e444c3df03994fdc001542f92af468196c  internal/web/files_recovery.go
8456850b0ca826712022f1c6be9a6be7fbb01f32d9b3ededdc27723b2e7f24ae  internal/web/files_security_test.go
9ccf5e65f3b9ee2a2fb921b1a206eb034092367dfba601d44741daaf2a0ddc35  internal/web/files_test.go
d5665b7678404992bd695fe2652781afa1ce0193334a04e7a9b85479fd90cc10  internal/web/files_upload.go
b418837157d8fd767e02365c14a441b302b6a7f9f5d0cd8406a79b316c7cc72a  internal/web/http.go
f80a3e874f4f01c5067a0c79244cf574fb5964d70e87bcb1c56e0a7f40f1c259  internal/web/login.go
ac19c4ff5a123a07a284031571358724602fc4bc53c9942bc67a3260c0c66e35  internal/web/login_boundary_test.go
0891ec36aad26d0402aa31efd0b8904055cc48a679a454ed8b89913e4bc64a81  internal/web/mutations.go
bb0f17ebd06d8c2dd7c66d353a63f65f854c4ec3bf787bfe4e8dee82497ac2d6  internal/web/pages.go
53a99f5df0a7c61551eb8495c47d657a7858ac9d9aedb525d03ebb47c2d8df2e  internal/web/pagination_test.go
572c01bcd71f480fe220a31e6936b0fa64acbe84a62fc0c87975a10d1df19b2a  internal/web/server.go
da4222eb34387bc1fb146b7fdb7648caec8c0fec358f7839800a6753565faf50  internal/web/storage.go
fbb9c9966abc1474c777a20865f394e42be88b34d6631a9bf9f3d439344b13b5  internal/web/workflow_test.go
df4ade6fc2e4fd0f3551ddb5484bf82b9ebda90fe7ea20d5a92ea6bddc208445  internal/webui/assets/app.css
3f9468faf5adf5dbec4d8478145d366720e28398f67562a7e287af061d4fea13  internal/webui/assets/app.js
ef6b197783063ee1078f58975b49ef9566cd88e7b20ed8044e2aa9a6e2d3033b  internal/webui/render.go
92cf129e6a15ff5cfe743d637294913003a64c210f6e6b646fe41ee921b73b49  internal/webui/render_test.go
35f0e17d01fa7ea04b8c05cdff18dd7a25cfb42e1b5fe38b4025a73f97f733ab  internal/webui/templates/board.html
b8571c25cb879fb14cf925f85f16e34766cb49f26226dc8dc7a4021c7543186e  internal/webui/templates/designs.html
f6e4bfc90e16b5133a933a1eef1d5a0801dc578c78d3770b4f64026402faf839  internal/webui/templates/history.html
4bc997f9df67c4d30e62c3b9993ea05227892b81be6deeb10c91eef790c6346c  internal/webui/templates/layout.html
3f9988696b5a1c416a1e47cca2dc917b3d8b0283bf39d1bff4c1338303f62bc6  internal/webui/templates/task.html
2ec9772abb17118056a587033df3776767f38e6b8c273a756bf3c90f15aa1fd6  tests/infra/test_web_backup.py
1e24d794f455bb14e370522de00ded44cf616e6a519ffb078e322dfb45006ee4  tests/infra/test_web_proxy.py
35529e29dc428a093246fc9ca00771680837d3bfce86e904006c90afde49dbe3  docs/web-requirements.md
d0ceac3248b239c1f56c9a8c941b836aa116c95ffb584c4e7d49aa5edcccbf7c  docs/web-architecture.md
08892abdaef370746ce54d527b0b9c859da403f6b537e98d634126cae0c6c18e  docs/web-security-contract.md
```

## Журнал и передача

- 2026-10-19T18:15:43+00:00 completed: независимое security review frozen source выполнено,
  APPROVE; источник доказательств — собственные чтения и запуски выше.
- Следующий исполнитель: координатор и отдельный tester для process/browser/
  restore/rollback проверок, затем независимое UI review и разрешённый выпуск.
  Изменения source после этого manifest требуют затронутого re-review; TASK
  не закрывается одним security approve.

## Финальная дельта OS helpers и Referrer-Policy — 2026-10-19T18:18:12+00:00

Координатор сообщил две обязательные дельты после первого manifest: portable
native client build и найденный независимым UI-review092 отказ обычной login
HTML формы с Origin:null. Выполнен отдельный re-review. **Итоговый вердикт:
APPROVE frozen реализации по manifest ниже.** Первый manifest выше сохраняет
историю и больше не является итоговой проверенной версией.

- `files_paths.go` использует платформенные designOwned/designNoFollow/freeDisk.
  `files_platform_unix.go` (darwin/linux) сохраняет точные O_NOFOLLOW, Stat_t UID
  и Statfs Bavail. На прочих OS designOwned=false запрещает web startup; fallback
  не открывает ослабленный file server. Поддержка Windows client не означает
  готовность Windows web server. Windows cross-build здесь лично NOT-RUN,
  результат координатора не подменяет мою security-проверку Linux/macOS semantics.
- Browser HTML получает Referrer-Policy:same-origin. Это сохраняет Origin у
  same-origin обычного form POST и не отправляет Referer за границу origin;
  bearer, answer и другие секретные/контентные поля в URL по-прежнему отсутствуют.
  Exact single Origin/CSRF guard не ослаблен и не принимает null. Основание
  сверено лично с первичным [Fetch §3.2](https://fetch.spec.whatwg.org/#append-a-request-origin-header)
  и [Referrer Policy same-origin](https://w3c.github.io/webappsec-referrer-policy/#referrer-policy-same-origin).
- При чтении первой header-дельты выявлено переопределение native design API
  policy: общий web handler ставил same-origin после transport no-referrer.
  Возвращено координатору одной узкой дельтой. Он добавил no-referrer перед
  handleDesignsAPI. Исправление лично перечитано и проверено test overlay;
  замечание **закрыто**, открытых blockers нет. Старое описание SEC-W03 выше
  уточняется: browser=same-origin; native API=no-referrer.

PASS — targeted recheck без полного повторного набора:

```text
go test ./internal/web ./internal/transport -run 'TestDesignRejectsPathsAndSymlinks|TestDesignPreviewAndAttachment|TestDesignRecoveryPendingOrphanAndReadyIntegrity|TestBrowserOriginCSRFAndDraftBoundaries|TestLoginFileSourceBoundsAndPreauthCSRF|TestWebExtraStaysInsideHostAdmissionAndNativeOriginBoundary' -count=1
web 0.410s; transport 0.315s

go test -overlay=/private/tmp/accord-security091/overlay.json ./internal/web -run TestSecurity091ReferrerPolicyByRoute -count=1 -v
PASS / web 0.346s
```

Временный overlay и один unit test хранятся только в `/private/tmp/accord-security091`;
source repository не менялся. Он проверяет actual handler headers на `/web/login`
(same-origin), `/api/v1/designs` и `/api/v1/designs/missing/download` (no-referrer).
Нового process/E2E до approve не создавал. Browser success из WORK092 передан
координатором, лично browser не открывал; все прежние NOT-RUN ограничения остаются.

Итоговый полный manifest: 59 runtime/test файлов +3 входных контракта.
SHA256 блока ниже (LF, включая последний): `de18d1359d8458a40aec7b4dc566b77b4533020225559c0c4fe3abff6c71fbe1`.

```text
0b72e465bfdd947751d1efcefa920a3c564f7456c86b176433795158171eb5dd  cmd/agent-board/main.go
ae2c53cf73ef8ca6dae417e000e04b28a12a44d357c0611163723bafdb55a3ee  infra/accord/public/render_public.py
8f7138408f14e7d73226fa2872c15eb9447268f4c4c12f2949bcdd51b573a22d  infra/accord/web/README.md
55e38437bcc6368f386fd2c8974e8f307553af36127c204a6c5f9dbfb4d7a37f  infra/accord/web/accord-web-backup.service
c539a950f552d53573417f5da7549a63b39275312258607c6ab1d52b976f7a85  infra/accord/web/accord-web-backup.timer
282b297caf06af34fba9b8259d76e3901b9281c877ba1d3f109a381dd4128a4e  infra/accord/web/backup.py
c3eae44a95a94de3073a78a4f930320ac7deb269bb61aa17dd2a9b7fb4e40ea0  internal/bridge/resources.go
d79cad37d7f494597c6ef81ba8292f9ebb00ea6c9ef57882951dc4a1e3b21801  internal/core/catalog.go
23415692b8ca6aafeafab655ef7c3e23073fad3540653e5110125d5eb545d079  internal/core/decisions.go
a1e09816eda6f1335dce1abedfc1aa28579e5409493a8de6cb172f8e0beac2b9  internal/store/transitions.go
db0bee6024a9bc71e6ec3999af6a6dfa3f0f19e152dc7a06f338b12edcbe2c52  internal/store/web_auth.go
964183f750081b55665578ab37f55d7bf6c5dd3138a5fe1d84105e6b860dd8cd  internal/store/web_auth_test.go
501a50a8ea1c28b205765ab17e8ca3f2b84ae472b129feb6df3ce335df81aff4  internal/store/web_history.go
1e0dd4ab7bdf8a940a4c577731ebc91398aca248126f7bea65dca7a9e0a32c53  internal/store/web_history_test.go
58625c6e33a751eca1127b026af09c13b157dc4256528a46c3c3403676100b7e  internal/store/web_questions.go
41259315e79bb0eac6458e3c65d2aece6d7f7ecf9a13da9ba257f3290d0a1913  internal/store/web_questions_test.go
34418d1d1ab4d92a407b6a0012f69249d3a1350a6329ee1f30bd6a7ea90d3f95  internal/transport/design_resources.go
d86db1b3d78b648228621bfc653fd84156145b25f4b93652a46248de209a1e99  internal/transport/http.go
37979bf98a086f4c4517e712c064bf866140a046d2d6bc45fae7808f2ad22b68  internal/transport/http_security.go
7226acf9f754f5b7d2593b3cba6ae69c8d35530c700fecf0a879e38fc8333f32  internal/transport/mcp.go
dbd528069e3e9eed1ad7c1310f83d57036a36bfc456532c7ccb8aeaa20d25f0b  internal/transport/web_integration_test.go
7ac21452e79e8215ec4e0f43fefefe0065ff041bc3d54d549313d6fddfda4ab8  internal/web/actions.go
14fe9c40857ac53a042730b6544d7be537071f2d0b5b0349f07ec24dc5ff07be  internal/web/backend_test.go
8bbb6dff33f511f65399b4c84f5a6c3b8ca00f7374ede895df1c872b52889590  internal/web/contracts.go
643ff6dc665d08aa85c13880fb4806c09f5e858c93355a7a32ab5d69094859ce  internal/web/design_http.go
c2866f9d1d1354ca12e83adc9d9b0f4a698408f37e0685c0db98fe9c5ff87775  internal/web/design_read.go
9680a1fe9b9210cd6c7ae9fb24251f69b714a7a64d7c67bd699f9ec38ef7c70e  internal/web/design_store.go
66c01807f43db75e096267516a167d5c258a4d14405b37becbd24f22ca941497  internal/web/files_budget.go
8fc4b348e0beb02f8d670ec24f47413dc96cd3142429edbdc0055fa2f71fcfd4  internal/web/files_cursor_test.go
4babc3ffb1686765e33b93c065defd907fdeab42e0bf288e84926a0aa2315f54  internal/web/files_form.go
93e4ed1c26f006d8fa0b2c357e804a9871d50b9008b6b22cf539b1673ece3cba  internal/web/files_format.go
2e65e4c005e930add6437f692a2c1748212739091b469a0ccf660ab6af62c45e  internal/web/files_integration_test.go
099666fcbe9650675a3a250ba18c1e90d9f3d59ca3b58b92be09b774fb4129c2  internal/web/files_paths.go
a9a884e4f8d9db5147b11a6bd8c838caa301221bdd92f34baa6884b205aeebc2  internal/web/files_platform_other.go
fe3e6944a2308a86635956b0dea0e1900da33b4d19b6c745e90f04ebfe0e5c24  internal/web/files_platform_unix.go
042633c7ac95e16222fbfa049e7e94e444c3df03994fdc001542f92af468196c  internal/web/files_recovery.go
8456850b0ca826712022f1c6be9a6be7fbb01f32d9b3ededdc27723b2e7f24ae  internal/web/files_security_test.go
9ccf5e65f3b9ee2a2fb921b1a206eb034092367dfba601d44741daaf2a0ddc35  internal/web/files_test.go
d5665b7678404992bd695fe2652781afa1ce0193334a04e7a9b85479fd90cc10  internal/web/files_upload.go
8cbbafd28385acac39c87955b38f8ac7ef38eea98f0e582025a516b6a3652368  internal/web/http.go
f80a3e874f4f01c5067a0c79244cf574fb5964d70e87bcb1c56e0a7f40f1c259  internal/web/login.go
ac19c4ff5a123a07a284031571358724602fc4bc53c9942bc67a3260c0c66e35  internal/web/login_boundary_test.go
0891ec36aad26d0402aa31efd0b8904055cc48a679a454ed8b89913e4bc64a81  internal/web/mutations.go
bb0f17ebd06d8c2dd7c66d353a63f65f854c4ec3bf787bfe4e8dee82497ac2d6  internal/web/pages.go
53a99f5df0a7c61551eb8495c47d657a7858ac9d9aedb525d03ebb47c2d8df2e  internal/web/pagination_test.go
572c01bcd71f480fe220a31e6936b0fa64acbe84a62fc0c87975a10d1df19b2a  internal/web/server.go
da4222eb34387bc1fb146b7fdb7648caec8c0fec358f7839800a6753565faf50  internal/web/storage.go
fbb9c9966abc1474c777a20865f394e42be88b34d6631a9bf9f3d439344b13b5  internal/web/workflow_test.go
df4ade6fc2e4fd0f3551ddb5484bf82b9ebda90fe7ea20d5a92ea6bddc208445  internal/webui/assets/app.css
3f9468faf5adf5dbec4d8478145d366720e28398f67562a7e287af061d4fea13  internal/webui/assets/app.js
ef6b197783063ee1078f58975b49ef9566cd88e7b20ed8044e2aa9a6e2d3033b  internal/webui/render.go
92cf129e6a15ff5cfe743d637294913003a64c210f6e6b646fe41ee921b73b49  internal/webui/render_test.go
35f0e17d01fa7ea04b8c05cdff18dd7a25cfb42e1b5fe38b4025a73f97f733ab  internal/webui/templates/board.html
b8571c25cb879fb14cf925f85f16e34766cb49f26226dc8dc7a4021c7543186e  internal/webui/templates/designs.html
f6e4bfc90e16b5133a933a1eef1d5a0801dc578c78d3770b4f64026402faf839  internal/webui/templates/history.html
4bc997f9df67c4d30e62c3b9993ea05227892b81be6deeb10c91eef790c6346c  internal/webui/templates/layout.html
3f9988696b5a1c416a1e47cca2dc917b3d8b0283bf39d1bff4c1338303f62bc6  internal/webui/templates/task.html
2ec9772abb17118056a587033df3776767f38e6b8c273a756bf3c90f15aa1fd6  tests/infra/test_web_backup.py
1e24d794f455bb14e370522de00ded44cf616e6a519ffb078e322dfb45006ee4  tests/infra/test_web_proxy.py
35529e29dc428a093246fc9ca00771680837d3bfce86e904006c90afde49dbe3  docs/web-requirements.md
d0ceac3248b239c1f56c9a8c941b836aa116c95ffb584c4e7d49aa5edcccbf7c  docs/web-architecture.md
cf507adf68703828422f8719a53a7f240d7db076580f72ea62482280d69257a1  docs/web-security-contract.md
```

- 2026-10-19T18:18:12+00:00 reopened/rechecked/completed: OS/header дельта проверена,
  найденный native-header mismatch исправлен и подтверждён; итоговый APPROVE.
- Передача: координатор → отдельные browser/process/restore/rollback QA и
  UI-review; source после final freeze не менять без затронутого re-review.
