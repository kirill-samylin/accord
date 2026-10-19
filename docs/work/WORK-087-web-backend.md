# WORK-087 — backend browser auth и workflow Accord

- Исполнитель: `/root/webbackend087`, разработчик; координатор `/root`.
- Статус: implementation complete, передано на независимое ревью; self-approve не выдаётся.
- База: `f06338089bec3b8c7f17f541c463131f5c98f96c`, worktree `.local/web-service-084`, ветка `codex/web-service`.
- Роль подключена профилем Davinci read-only; именованные Claude-модели в этой среде не закреплены, модель наследуется. Davinci не изменялся. Tandem отключён по прямому поручению.
- Разрешённая запись: `internal/web/**` кроме `design*`/`files*`, `internal/store/web_auth.go`, `internal/store/web_auth_test.go`, этот WORK. Соседние изменения других авторов сохранены; commit/deploy не выполнялись.

## Результат

Browser HTTP Server New/Close/Ready, SSR Renderer/Page контракты, private web.db schema1. Вход typed token или единственный token-file до4096байт; случайная preauth cookie+CSRF, новый случайный browser cookie256bit, SHA256 cookie в webDB, только tokenID вместо bearer. На каждом запросе current Store.AuthenticateID заново получает principal/expiry; cookie8h absolute, не позже token expiry. HTTPS cookie `__Host-accord_session`, Secure/HttpOnly/Strict/Path=/, без Domain. HTTP development разрешён только явно настроенному literal loopback, с отдельным cookie name.

Все browser POST требуют один exact configured Origin и CSRF. Отказы сохраняют несекретный draft; token/password не входят в Page. Login limiter атомарно резервирует максимум5 неудачных попыток за минуту. По умолчанию один безопасный глобальный bucket: заголовки proxy не доверяются; это ограничивает общую доступность входа при атаке. Необязательный TrustedProxy требует loopback peer и один валидный X-Accord-Client-IP; карту ограничивают1024 buckets и минутный TTL. Preauth ограничены1024 записями/10мин. Безопасные ответы CSP/no-store/nosniff/referrer policy; autoescape принадлежит независимому UI-author.

Native human board-session открывается только login/явным continue/перед новым create/claim. Stable pending Describe записан до board call. Чтения не делают Describe/heartbeat/claim. Сервер выдаёт отдельные сохранённые action IDs для форм, привязывает operation/payload/native session до вызова, сохраняет результат после. Повтор использует исходный requestID/session, в том числе native replay после истечения; неизвестный результат сохраняет прежний ID. Перед новой session проверяются незавершённые native actions. Повтор cached action не продлевает локально показанный срок native session.

Создание эпиков и всех существующих ролей, review/tester target+commit через действующий Create. Вопрос берётся только адресным Next(TaskKey), nil claim возвращает409. Answer/Release/Heartbeat используют текущую human identity и exact generation; нет agent lease controls. Явный heartbeat возвращает следующий action ID и authoritative lease из Get. Свой ранее взятый вопрос показывается независимо от страницы очереди через сохранённый key и новый authoritative Get. Ошибка/stale lease не захватывает вопрос повторно, draft остаётся копируемым.

Questions используют root-owned ListQuestions с глобальным native queue order, история — root-owned ListDecisions newest first с фильтром эпика. Legacy выводится только для текущего Evidence.Answer без сохранённых human_answered events и маркируется UI отдельно; прошлые тексты не восстанавливаются по догадке. История immutable board events и task author/provenance реализованы root, не двойной записью browser backend.

Design/files интеграция согласована с WORK-088: общие Server/db, initDesigns до готовности, browser identity/current auth, streaming upload свой CSRF/action hook, native bearer handler, exports ListDesigns/GetDesign. Timer15мин вызывает session/preauth cleanup и design cleanup. SQLite synchronous=FULL; private DB/sidecars и существующие path components проверяются до IO. Полный file lifecycle принадлежит WORK-088.

Page фактически map-based Draft и Actions/FieldErrors, согласован с координатором и UI; временной второй DTO-копии нет. StaticHandler монтирует готовый UI. Все authored production/test файлы ≤300строк. Эпики имеют независимый `epic_cursor`/EpicsNextCursor, без потери task filters; pagination beyond100 покрыта integration test.

## Фактические проверки

- PASS: `go test ./internal/web ./internal/store ./internal/webui` после интеграции авторов.
- PASS: `go test -race ./internal/web ./internal/store` на последней реализации (web3.187s, storecached после отдельного race8.723s).
- PASS: `git diff --check` в общем worktree на момент передачи.
- PASS: авторские unit/integration: AuthenticateID current identity/expiry/revoke; cookie flags/rotation/fixation/logout; cookie не авторизует native API; tokenID не является cookie; Origin missing/null/empty/duplicate/foreign; неверный CSRF и сохранение draft; concurrent login16 requests→ровно5 auth failures+11 rate limits; один token source/file4096 bound/preauth replay; browser absolute expiry/revoked POST; конфигурация origin; exact question race; один human_answered при lost web result + restart + native session expiry; mismatched replay409; явный heartbeat с fresh nextID; release/reclaim/stale generation и draft; GET не продлевает аренду/не открывает сессию; явный идемпотентный continue.
- API: проверена текущая installed Go1.26.1 документация net/http.Cookie; остальные API стандартной библиотеки и существующие SQLite зависимость без обновления. Architecture содержит проверенные официальные ссылки на новые file deadline/root API, реализуемые отдельным автором.
- NOT-RUN: независимые reviewer/security approve, process/browser E2E, HTTPS live/proxy/runtime/deploy, физический второй ПК. Эти этапы принадлежат следующим WORK и не подменяются unit/integration.

## Журнал

- 19.10.2026 started — прочитаны agent-board AGENTS, developer role/process Davinci read-only, web requirements/architecture и security086; подтверждены непересекающиеся scopes.
- 19.10.2026 contracts published — contracts.go и AuthenticateID переданы root/UI/files; согласованы hooks и distinct per-form action IDs.
- 19.10.2026 implementation complete — авторские auth/workflow/restart/fencing tests и race PASS; общий web/store/UI suite PASS; передача независимому ревью.
