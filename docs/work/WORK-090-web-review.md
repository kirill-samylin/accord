# WORK-090 — независимое code review веб-Accord

- Роль: reviewer, исполнитель `/root/webreview090`; авторы087/088/089 и root — другие агенты.
- Статус: **APPROVE** текущего кода после исправлений. Открытых P1/P2 не обнаружено.
- Родитель: TASK-004. База `f06338089bec3b8c7f17f541c463131f5c98f96c`, `codex/web-service`, незакоммиченный diff в `.local/web-service-084`.
- Контроль версии: 2026-10-19T21:15:03+03:00; после FREEZE087/088/UI/root/infra.
- Профиль reviewer, общий процесс, parallel-work и code-structure Давинчи прочитаны только для роли. Именованные Claude-модели недоступны; модель наследуется, отдельность автора и проверяющего сохранена. Tandem OFF по явному поручению Accord. Давинчи не изменялся.
- Изменён только этот WORK. Репродукции и manifest — `/private/tmp/accord-review-090/`; production code, Git, deployment и runtime состояния не менялись.

## Объём и вывод

Проверены source и окружающие контракты: SSR UI/escaping/forms/JS, browser cookie/login/preauth/current token auth/CSRF/logout/session continuation, operation-bound request IDs и retry; адресные human claims/heartbeat/answer/release и generation fencing; immutable board decision events, legacy fallback и queue order; потоковые файлы/квота/резервы/версии/preview/recovery/path safety/paging; native bearer design REST, MCP resources/12 прежних tools, worker bridge; общий transport perimeter, cmd flags/readiness; nginx opt-in --web; root backup/retention/restore auth reset/timeout restart.

board.db остаётся schema2, web.db — schema1; D4/E не включены. Ответ и immutable human_answered пишутся одной board transaction, включая native human path, exact replay не создаёт второй event. Сохраняется transactional current principal check перед native mutation/replay. Browser sessions не хранят bearer и перепроверяют tokenID на каждом запросе. Native APIs не принимают cookie и отвергают любой присутствующий Origin. Design preview перекодирован только из bounded PNG/JPEG; WebP attachment-only принят в ADR-W4 и согласован security contract, не выдаётся за полноценный decode/preview.

Логика не подменяет работу агента: UI вызывает authoritative core/store; чужие live leases и evidence gates не обходятся. Переиспользование DTO идёт через contracts.go, templates/layout/task-card/actions общие. Перенос в UI-kit Давинчи здесь N/A: Accord самостоятельный Go SSR сервис. Новых значимых дублей/необоснованных type casts не выявлено; Go any в JSON/generic scan — типизированная стандартная граница, не TS обход типизации.

## Закрытые замечания одного review-прохода

| Приоритет | Место и воспроизведение | Исправление и перепроверка |
| --- | --- | --- |
| P1 | UI templates/designs.html отправлял epic_key, parser files_upload.go отвергал даже пустое поле; обычный upload был нерабочим | Parser выделен в files_form.go, принимает epic_key, проверяет task/epic принадлежность. Собственная isolated TestReview090ActualFormFieldsAreAccepted PASS; авторский actual multipart scope test PASS |
| P2 | bindAction до inspect связывал hash невалидного файла; designPageError повторно выдавал тот же ID, исправленный файл получал409 | Confirmed400/413 выдаёт fresh ID, uncertain5xx сохраняет исходный. Собственная TestReview090CorrectedFileAfterValidation PASS; авторский format/uncertainty test PASS |
| P2 | pages.go терял ListEpics.NextCursor и показывал только первые100 эпиков | Отдельные EpicsNextCursor/epic_cursor, UI continuation сохраняет filters. Тест101эпиков PASS; render helper/link проверен |
| P2 | backup.py finally не выполнялся при default SIGTERM, а unit TimeoutStartSec=10min мог оставить Accord остановленным | TERM/INT преобразованы в RuntimeError до stop; finally делает start; TimeoutStopSec60s. Изолированный signal.raise_signal(SIGTERM) при mocked snapshot дал is-active→stop→start PASS, настоящий systemctl не вызывался |

Рекомендация строгой проверки snapshot дерева также реализована: manifest0600, whitelist каталогов/имён, запрет symlink и неизвестных entries. Source перепроверен, backup tests PASS. Последняя правка088 переносит seq на момент publish; cursor не принимает поздно завершившийся pending в ранее начатую страницу, targeted regression PASS.

## Фактические проверки

Go1.26 toolchain `/Users/shagi/Desktop/Project/agent-board/.local/toolchains/go/bin/go`, GOTOOLCHAIN=local; GOCACHE/GOMODCACHE/GOTMPDIR — существующие private toolchains каталоги Accord. Новые тесты репродукции подключены Go overlay только из /private/tmp, repository tests не менялись.

- PASS до freeze: три existing checks — TestExactQuestionRaceAnswerCrashReplayAndFencing, TestHumanDecisionIsImmutableAndReplayDoesNotDuplicate, TestDesignVersionsReplayRecoveryAndAuth.
- PASS после freeze: обе TestReview090, TestEpicsHaveIndependentPaginationAndPreserveFilters, TestExactQuestionRaceAnswerCrashReplayAndFencing, TestHumanDecisionIsImmutableAndReplayDoesNotDuplicate, TestWebExtraStaysInsideHostAdmissionAndNativeOriginBoundary, TestSDKStatelessToolsShareRESTState. Пакеты web/store/transport PASS. Фильтр TestRenderer не совпал с именами UI-тестов; этот конкретный UI шаг `[no tests to run]`, за PASS UI не засчитан.
- PASS отдельным UI subset: TestAllPagesEscapeUntrustedContent, TestUploadCSRFFirstAndFileLast, TestPaginationPreservesFilters.
- PASS последней дельты088: TestDesignCursorExcludesEarlierPendingPublication, TestDesignFormatFailureRotatesIDAndUncertaintyPreservesID, TestDesignActualMultipartScopeAndOptionalEpic.
- PASS: `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest tests.infra.test_web_backup` —5 checks после окончательного hardening. Proxy4 tests прочитаны как новый unit-контракт; их полный зачётный прогон принадлежит root.
- PASS: изолированная Python TERM проверка с synthetic private paths и mocked systemctl/snapshot, без управления сервисами.
- Один промежуточный overlay запуск до freeze попал в незавершённую запись actions.go (`a.Fresh undefined`) и не скомпилировался. После freeze повтор PASS; это не финальная ошибка.
- Новые authored web/webui/files/backup файлы ≤300 строк. Существующий internal/transport/http.go301 строка — принимаемое исключение одной компактной route/perimeter ответственности, прирост небольшой; декомпозиция ровно ради1 строки ухудшила бы обзор границы.

Полный go/race/vet прогон не повторял: авторские результаты описаны WORK-087/088/089, root выполняет общий check. Это независимый code approve, не security/UI/runtime approve и не подтверждение выполнения всех WEB-AC.

## Ограничения и отложенные рекомендации

- NOT-RUN этим reviewer: новый process/browser E2E, screenshots/mobile/keyboard, настоящий nginx/systemd/VPS/HTTPS, live upgrade/rollback, restore drill со старым бинарным файлом, второй физический ПК. По процессу новые E2E начинаются после code approve отдельным tester; их тесты потребуют review. Старый-shape reader unit не является фактическим old-binary rollback.
- Отдельные security и UI reviewers обязательны. Без макета pixel match N/A; полнота состояний и удобство проверяются отдельно.
- Legacy history bounded первыми100 done human tasks и без собственного cursor: при большом числе pre-web ответов нужна отдельная пагинация/явная ссылка. Принятый bounded fallback не блокирует paginated immutable history новых ответов; прошедшие тексты не восстанавливаются по догадке.
- Не расширять эту задачу OAuth, D4/E, enterprise RBAC, file execution/unpack, UI-kit Давинчи или новое native MCP tool.

## Проверенная версия SHA256

Полный manifest также сохранён `/private/tmp/accord-review-090/final-sha256.txt`.
SHA256 текста manifest: `e0489d264d809289654612f94708d0b355033a3f142ce1aedc4ae2da64d9758d`. Совпадение необходимо перед использованием этого approve; последующие source changes требуют review дельты.

```text
0b72e465bfdd947751d1efcefa920a3c564f7456c86b176433795158171eb5dd  cmd/agent-board/main.go
6a35d85ba3d48cd3944a455ee7ca5fdd8607146d9cd84720efd379d0a14ed777  docs/TASK-004-human-web.md
d0ceac3248b239c1f56c9a8c941b836aa116c95ffb584c4e7d49aa5edcccbf7c  docs/web-architecture.md
db17b3e4c2269c9e20f26766a2dd852c43c466138abb09f851ba0656ef38890a  docs/web-guide.md
35529e29dc428a093246fc9ca00771680837d3bfce86e904006c90afde49dbe3  docs/web-requirements.md
08892abdaef370746ce54d527b0b9c859da403f6b537e98d634126cae0c6c18e  docs/web-security-contract.md
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
```

## Передача

Координатор: принять code review этой версии, дождаться независимых security/UI результатов, затем tester выполняет browser/process и backup/restore evidence. Выпуск — только с их фактическими результатами и проверкой действующего HTTPS. Этот WORK не объявляет TASK-004 завершённой.
