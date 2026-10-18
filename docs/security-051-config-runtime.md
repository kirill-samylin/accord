# WORK-051 — независимая проверка безопасности D

Дата: 18.10.2026. **Вердикт: REQUEST CHANGES.**
Проверен source HEAD `932aa2a3017da0b4fc06032e5134d4ed8fdf2d5b`,
`codex/config-runtime`, каталог `.local/config-runtime-003`, diff от approved
main `1e9025cd6e3122f0adf6165e09845202736d004a` до HEAD. На начало и конец
проверки source checkout чистый. Merge содержит D `dad4c5f` и approved C QA.

Manifest `.local/config-runtime-003/.local/work043/source-manifest.json`:
aggregate `source_sha256=414a24ff535517f14d74e92195122b8775922d9a31386b5e2459a05079aff481`;
SHA256 самого JSON `8de46f25fbb6908828ae7eba1061f545235c38d845045bdc26e22996fbe497e8`.
Независимо сверены все85 file hashes, расхождений0. Это два разных hash,
исходного расхождения между заданием и freeze нет. Протокол:
`.local/evidence/security-d-051/manifest-check.json`.

## Граница и модель угроз

Профиль: `davinci-gui-/docs/agents/roles/security.md`, общий процесс и правила
параллельной работы прочитаны. Контракты: `requirements`, `service-contract`,
`config-contract`, `security-contract`, `configuration-runtime`, WORK043.
Проверяющий отдельный от автора; именованные Claude модели недоступны,
использована та же наследуемая модель среды, что у автора. Tandem OFF по
прямому актуальному указанию; подключений/claims не было. Исходники и тесты
замороженного среза не менялись. Review-only Go overlay находится в private
`.local/evidence/security-d-051/` и не входит в исходный manifest.

Активы: полномочия bearer/scopes, аренды и поколения, история конфигурации,
evidence и обязательства проверок, credentials в SQLite/backup. Недоверенные
входы: MCP/REST JSON, выбор роли, request_id, поля задач и evidence.
Локальная CLI/OS identity с правом доступа к DB является доверенным admin;
она вправе применить согласованный config, но изменение config не должно
воскрешать утраченную аренду или обходить неизменяемые safety floors.
Сервис координирует области, не блокирует файловую систему отключённого агента.

## Блокирующая находка, воспроизведённая WORK051

### S051-01 / P1 — live update областей обходит single_editor_role

Место: `internal/store/tasks.go:203-213`; существующая обязательная проверка
редактора находится только в `internal/store/claim_rules.go:24-33` и вызывается
из claim. После проверки owner/generation Update канонизирует новые Areas и
проверяет overlap, но не single_editor_role.

Условия и шаги:

1. Config содержит public и restricted; у restricted `single_editor_role: analyst`,
   exact alias secret. У developer разрешены next/update.
2. Developer получает task с areas=[public] и действующее generation.
3. Та же сессия вызывает Update своей живой задачи с areas=[secret] либо
   areas=[restricted/child]. Других live task на этих областях нет.
4. Ожидание: FORBIDDEN_ROLE и прежние areas/lease/version без изменений.
   Факт: SUCCESS, task получает restricted либо restricted/child и живую lease.

Последствие: агент приобретает серверное разрешение работать в области другой
роли после корректного claim, без admin/config изменения и без роли analyst.
Это обход авторизации областей; запрет claim не обеспечивает требуемую границу.
Не заявляется доступ к OS-файлам или обход bearer.

Исполняемое подтверждение: FAIL всех6 негативных подслучаев overlap
`off|warn|block` × exact alias/canonical child. Тест требует отказа и падает на
фактическом успехе. `.local/evidence/security-d-051/single-editor.log`,
`security_probe_test.go`, `overlay.json`.

Минимальное исправление: применить общую обязательную area-role проверку к
новым каноническим областям живой задачи до принятия Update, независимо от
area_overlap mode. Сохранить возможность coordinator задавать область будущей
незахваченной задачи для её целевой роли. Отказ должен атомарно сохранять
прежние метаданные/lease/event/version. Regression: обе формы путей, все3
режима, правильная роль проходит; cross-session/stale generation не проходят.

## Связанные блокеры WORK050, проверенные по source и его evidence

Ниже не отдельные повторные runtime прогоны WORK051. Координатор передал
находки отдельного reviewer050; прочитаны его overlay source и фактические FAIL
логи. Security повторно проверил соответствующие production-ветви. Владельцем
первичных воспроизведений остаётся WORK050; они также препятствуют security approve.

### S051-02 / P1 — выключение singleton воскрешает утраченный SID

`internal/store/runtime_roles.go:146-147`: sessionLive перестаёт проверять
RoleExpiresAt/generation, если текущий config singleton=false.
`config_history.go:112-116` допускает такой apply, когда старая role lease уже
истекла и других live sessions нет. Сценарий050: roleTTL1s/sessionTTL1h, expiry,
apply architect.singleton=false, Heartbeat старого SID → SUCCESS. Прочитан
`.local/evidence/review-d-050/singleton-repro.log` (FAIL негативного assert).
Риск: config меняет историческую семантику уже потерянной role lease; старый
writer получает право новых мутаций без новой серверной сессии.
Исправление: зафиксировать семантику выданной role lease в session и проверять
её expiry/generation независимо от текущего singleton-флага; смена config не
возвращает утраченное владение. Перепроверить toggle/delete/re-add и старый SID.

### S051-03 / P1 — эпик закрывается при evidence_complete=false у done ребёнка

`internal/store/gates.go:128-139`: epicClosable проверяет done/отсутствие lease и
blocked(child), но не успешность самого child/EvidenceComplete. В отличие от
этого gateSuccessful проверяет EvidenceComplete. Сценарий050: у developer
дополнительный pr обязателен, evidence_required=warn, done без pr; затем epic
done → SUCCESS, DTO показывает done и blocked=true. Прочитан `repro.log`
WORK050, TestReview050IncompleteEvidenceEpic. Риск: финальная приёмка эпика
обходит установленное требование, что неполное evidence не открывает completed
gate. Исправление: одинаковая проверка успешности непосредственных детей при
явном закрытии/реконсиляции, с сохранением правил исторических superseded проверок.

### S051-04 / P1 — human отменяет чужой epic при can_cancel=false

`internal/store/transitions.go:119-149`: именованная human роль допускается в
transitionEpic до проверки canCancel; ветка cancelled её не повторяет.
Сценарий050 использует отдельный human token, requester чужого epic другой;
config human.can_cancel=false; cancelled с reason → SUCCESS. Прочитан `repro.log`
WORK050, TestReview050HumanCancelPermission. Риск: динамическое ограничение
отмены обходится special-case именем human. Исправление: проверять cancel grant
или requester для cancelled независимо от имени роли; coordinator/human права
закрытия не должны неявно выдавать несогласованное право отмены.

## Проверенные защитные цепочки и предел доказательств

- **Auth/scopes/tools:** токен заново аутентифицируется в transaction; scopes
  сохраняются как exact roles, новые defaults только agent. Human mint/scope set
  требуют allowlisted owner; Describe и новые операции повторно проверяют owner,
  role members. Human category/complexity проверяются при claim и done.
  Replay требует текущий валидный bearer, возвращает старый DTO без renewal.
  Результат source+прошедших существующих tests; S051-02/04 остаются исключениями.
- **Local admin:** ApplyConfig, SetTokenScopes, mint/revoke и local handoff не
  входят в remote Service/MCP catalog; HTTP config/history только authenticated GET.
  Raw YAML/comments не входят в read DTO. Результат source+transport tests.
- **Config transaction:** writer lock предшествует loadActiveConfig; snapshot
  отдельный, Store.cfg используется для bootstrap. CAS идёт до no-op; history
  только INSERT, исторический apply создаёт новую version. Failed audit откатывает
  history/active. Two-connection CAS и live rejection воспроизведены tests/race.
  Это не отдельный two-process CLI/reload PASS.
- **Evidence/provenance:** built-in floor и типы не снимаются warn/off, builtin
  properties нельзя переопределить; raw JSON сохраняет точные integer, duplicate
  keys отвергаются; tokens_spent на completion проверяется как int64. Custom
  verifier требует schema+independent_from; сервер задаёт target SHA/generation и
  author session, ordinary custom completion не подменяет approved/passed.
  Latest verifier obligations глобальны, historic OutcomeField сохраняется после
  удаления неактивной роли. Source+tests; финальная приёмка имеет S051-03.
- **Fencing:** role/task/session TTL разделены; expiry materialization, owned
  generation и server sessions защищают обычные вызовы; bridge ограничен
  minTTL/role_expires_at и не принимает чужую describe-role. Source+tests/race;
  конфигурационный переход singleton имеет S051-02.
- **Capabilities/files:** config parser не раскрывает env values, не делает
  includes/remote schema load или scripts; неподдерживаемые auth.github/events/
  notifications отвергаются. Импорт E отсутствует и не объявляется реализованным.
  Schema3 migration атомарна, legacy scopes ровно7, prebackup private+checksum;
  Backup/restore/recovery признают1/2/3, recovery отзывает credentials и сессии.
  Source+unit tests; реальный restore/deployment в WORK051 не запускался.
- **Workflow:** owner/handoff не переносят task lease; DAG и one_task safety
  остаются block. Area canonicalization и claim restriction есть, но live area
  update имеет S051-01. Queue progress finding050 остаётся общей code-review
  находкой и не учитывается здесь как дополнительная security уязвимость.

## Фактические проверки WORK051

Среда: go1.27.2 darwin/arm64, `GOTOOLCHAIN=local`; бинарник
`.local/toolchains/go/bin/go`, build/mod caches из MAIN `.local/toolchains/cache/`.
Логи private в `.local/evidence/security-d-051/`.

| Проверка | Результат |
| --- | --- |
| HEAD/clean checkout, все85 manifest hashes | PASS, manifest-check.json |
| `go test ./cmd/... ./internal/... -count=1` | PASS всех7 пакетов, unit-integration.log |
| `go test -race ./internal/store ./internal/bridge -run 'TestConfigCASAcrossConnectionsNoOpAndFailureAtomicity\|TestConfigAuditFailureRollsBackHistoryAndActiveVersion\|TestLiveConfigRejectsChangesAndPreservesLeases\|TestRoleTTLExpiryFencesLiveSessionAndReclaimsTask\|TestSecurityDesignAndCustomVerifierObligationsStayGlobal\|TestRoleLeaseBoundsWorkerDeadlineWithLongSession' -count=1` | PASS обоих пакетов, race-security.log |
| review-only overlay `TestSecurity051SingleEditorAreaUpdate` | FAIL ожидаемого запрета,6 воспроизведённых обходов, single-editor.log |
| `git diff --check 1e9025c..HEAD` | PASS |
| Новые WORK050 runtime прогоны этим исполнителем | NOT-RUN; прочитаны source и3 относящихся к security FAIL evidence050 |
| Full process/acceptance, two-process CLI live reload, crash/old binary/restore drill | NOT-RUN |
| Actual VPS, внешний TLS/auth providers, remote Codex/Claude, UI, physical2PC | NOT-RUN |

Заведомо старые process fixtures не запускались и не считаются PASS: token
bootstrap TTL15m против явного serve TTL30s/1s даёт корректный CONFIG_CONFLICT;
пустая DB требует bootstrap owner; backup expected schema должна стать3.
Отдельный tester после актуальных approve адаптирует входы и проверяет D;
это не основание ослаблять source-инварианты или обходить эти security findings.

## Передача

Автору WORK043: исправить S051-01 и связанные блокеры WORK050, приложить
регрессии и новый точный source freeze. После исправлений нужны независимые
повторные security/code review новой версии; только затем новые process/E2E.
Общая TASK, source/main/Git/push/deploy этим исполнителем не менялись.
Неблокирующих новых security замечаний отдельно не выявлено. Рабочий выпуск
и E-функции этим отчётом не одобрены.
