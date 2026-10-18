# WORK-041 — повторное независимое review C и native QA

18.10.2026. Проверяющий `/root/recheck041`, отдельный от авторов032/034/035/038
и тестировщика039. Роли reviewer/security; общий процесс и профили прочитаны.
Именованные модели Claude здесь не закреплены; используется унаследованная
модель. Производственный код и тесты проверяющим не изменялись.

## Версия и вердикты

**C source + native Host integration: APPROVE.** Все три блокера WORK-037
закрыты; новых блокирующих замечаний нет. **Native process tests WORK-039:
APPROVE. Инструкции и runtime-отчёты: APPROVE** в указанных ниже границах.
Это не общий full V1 approve и не готовность непроверенной C/schema2 поставки.

Основная копия: база `99e722c44f83b1808120bba1fabb04cf2f68a5b5` плюс C WIP,
WORK-038 freeze7 и интеграция native `dc9c74cd692e287173def9fb5743dbf18ffdef21`.
Manifest101 `.local/evidence/review-041/manifest.sha256`, SHA256
`f71ae3680a4b19dd56165a6f8a1a709f13f605d1fa587950c511e6c0c5f32204`.
Freeze7 WORK-038 совпал: `375f062953a26e578e6e6bef1479bed095571fb704d1c3aaf424c9be32b7edb6`.
Остальные файлы прежнего C manifest037 сопоставлены: изменения ограничены
исправлениями038, API и native integration; неизменённые части повторно
не объявляются новой реализацией. Offline config033 остаётся отдельным
APPROVE036; runtime configuration данным review не подтверждается.

## Закрытие WORK-037

- REV037-01: `metric_events.go` сохраняет owner фактической server session;
  завершённые попытки, включая expiry/recovery, используют именно SessionID
  попытки. Текущая очередь использует только lease session; unknown остаётся
  пустой группой. Исторические события не переписываются. Исходный независимый
  Bob42/Alice repro PASS; новые regressions проверяют суммы групп, persistence,
  release/expiry/offline reset. `docs/api.md` объясняет participant semantics.
- REV037-02: `bridge/protocol.go` читает проверенный original JSON text сервиса
  и восстанавливает structuredContent через UseNumber. Typed ownership parser
  и downstream wire сохраняют int64. Исходный MaxInt64 repro PASS; regressions
  через настоящий HTTP SDK→stdio worker→SDK проверяют2^53+1/MaxInt64,
  persisted task, event delta, replay, release/next и actual structured wire.
  Успешная серверная мутация больше не переводит worker в quarantine.
- REV037-03: operational predicate исключает terminal states; terminal event
  закрывает уже открытый span, включая typed historical boundary. Gate predicate
  не ослаблен. Исходный cancelled-at10s repro PASS; clipping до/после отмены,
  ready restore и повторная отмена дают ожидаемые10/15секунд. Коррекция старого
  epic regression сохраняет проверку gate/history/atomicity и меняет только
  ошибочное ожидание operational blocked для done.

## Host integration и безопасность

Одиннадцать native source/test/infra файлов в основной копии побайтно совпали
с dc9c74c. В двух совмещённых файлах `http.go`/`mcp.go` сохранены validation
HandlerOptions, внешний allowedHost до auth и условное отключение SDK localhost
guard только за точной настроенной authority; добавлены C routes,12 tools и
notes resource. Все новые пути проходят те же bearer/Origin/admission guards.
Forwarded headers не расширяют список Host. Main CLI совпал с native commit.

Активы и границы: bearer, SID/generation, статистика и исходный JSON ответа;
недоверенные входы — tailnet HTTP/MCP, Host/Origin/forwarded metadata, JSON.
Session owner берётся из серверного факта, клиентское epic.Owner не становится
источником атрибуции. Точный numeric parsing не отменяет typed validation.
Новое чтение resource требует bearer и собственную live session, не выдаёт
SID за credential. Обходов auth/fencing/Origin в объединённой версии не найдено.
Изменённые Go/Python файлы проверенного набора до300строк; новых необоснованных
дублей и подавления ошибок нет. UI здесь отсутствует.

## Фактические проверки

- PASS: независимый `go test -race -count=1 -overlay
  .local/evidence/review-037/overlay.json ./internal/store ./internal/bridge
  -run 'Test(Reviewer037|StatsParticipant|StatsClosedAttempt|CancelledBlockedSpan|ExactInt64)'`:
  store2.421s, bridge2.421s. Overlay — прежние три reviewer repro без изменений;
  source/tests репозитория при воспроизведении не менялись.
- PASS: targeted combined race transport1.703s/CLI1.488s с Host/Origin/native,
  catalog/read/resource и CLI cases; targeted vet store/bridge/transport/CLI.
- PASS: независимо повторены **два** новых process cases в чистой native
  рабочей копии dc9c74c: `TestProxyProcessConfiguredHost` и
  `TestProxyProcessHostSecurity`, race harness2.567s. TLS socket→отдельный CLI,
  initialize/list/describe/create/get, bearer/Origin/Host matrix. SDK не Codex;
  дочерний binary обычный, race инструментирует harness.
- WORK-039 test SHA256 `58b00ee32775350b3d11b189805f24e7c431b908f0aabaf61207f70d4a568ee5`
  совпал. Остальные сценарии сверены по results.json и normal/race JSONL:
  каждый4 верхних/17 subtests, failures0; хеши из acceptance-039 совпали.
  Полный E2E suite повторно не запускался.

Private evidence `.local/evidence/review-041/`: fixes-race.log SHA256
`32a5481aaee0f91932614d8c7866b56b008838b1e6d54d6034be8ca5c4198b51`,
combined-race.log `a4baa7abb56126acc3b49988a0e847aaf32e078ea1a4a165d5e2bf4734bc8063`,
native-process.jsonl `53d9c85d01601ed159c1b1be5aa65d13f9399394c7fff815773711f0791ef04d`.

## Инструкции и доказательства runtime

Проверены second-pc/client-setup/codex-003/runtime-server-003/API. Устаревший
runtime абзац о незакрытом Host blocker исправлен координатором и перепроверен:
failure99 теперь исторический, retest dc9 PASS и test token revoked401.
Инструкции различают отдельный STDIO worker и общий HTTP с явным heartbeat,
отдельные машинные токены, Windows env/ACL и lifecycle внутренних субагентов.
Секреты и действительная private authority в проверенные отчёты не включены.

Codex log SHA256 `72b3bed5884cb7b64b46a49d23bdf61f5e952562d1707845f151e20df3984160`
совпал. Из JSON records независимо извлечены10 completed mcp_tool_call без
error, в порядке из summary/report. Три non-JSON строки CLI не считаются
MCP-вызовами. Revoke/cleanup/Linux/systemd/Tailscale — сохранённые metadata и
результат координатора, **этот reviewer не повторял live/model операции**.

NOT-RUN: main C process suite и обновлённые schema2 fixtures, live C/schema2,
нагрузка, второй физический ПК/Windows runtime, Claude, config runtime D,
UI/OAuth/integrations и полная V1. Native039 approve относится к11-tool native
ревизии; main12-tool fixture и backup/schema expectations обновляет отдельный
tester после этого APPROVE, затем проходит независимое review новых тестов.
Дальнейшее изменение source требует review затронутой дельты. Tandem/live/
commit/push проверяющим не выполнялись.
