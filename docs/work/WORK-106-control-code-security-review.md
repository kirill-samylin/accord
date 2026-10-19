# WORK-106 — независимое code/security review control UI

- Родитель TASK-005 / REQ-005; исполнитель `/root/control_review106`, reviewer + security.
- Статус: выполнено; итоговый вердикт **changes requested** по одному P2 ниже.
- База исходного приложения `a6501d8`, документации `38b806e`; рабочая копия
  `.local/web-control-099`. Это самостоятельный Accord, не приложение Давинчи.
- Прочитаны AGENTS Accord, общий процесс, parallel-work и профили reviewer/security
  Давинчи read-only; `control-web-contract.md` §8/9 с уточнением cursor/error,
  TASK-005 и ранний security WORK-105. Tandem OFF по AGENTS Accord.
- Запись только в этот WORK и `/private/tmp/accord-control-099/review/`.
  Source/Git/SSH/deploy/Давинчи не изменяются. Именованные модели Claude здесь
  не действуют; отдельный проверяющий использует унаследованную модель автора.

## Проверенная версия и границы

Backend103: manifest `/private/tmp/accord-control-099/backend-103-sha256.txt`,
SHA256 `e55fb67d263cd916342eec299a92d7b65d3bcf228e4d88baf1cb620720c78bcc`.
Все 16 source/test файлов совпали при `shasum -a 256 -c`.
Итоговый общий source freeze: `46fffb94f73130ddf5f654f3f11b5453123380d8`.
Manifest `/private/tmp/accord-control-099/source-freeze.json`, SHA256
`1d33f1782f2e14b4db9cf49faa07ab3c67c00b8f3bf7b1032367e651f0249064`.
Независимо сверены все212 записи: mismatches=[]; evidence
`/private/tmp/accord-control-099/review/freeze-verification.json`.
Новые production readers/browser projections/polling/static assets и
затронутые старые forms/routes входят в review; общий аудит старого сервиса,
новые E2E и фактический дизайн/HTTPS/load/deploy в эту WORK не входят.

## Модель угроз и проверенные свойства backend

Активы: текущая cookie/auth, session/lease/generation, задачи/ответы/immutable
события, SQLite availability и совместимость native клиентов. Недоверенные входы:
HTTP query/cursor/SID, titles/description/owner/run/machine/project и event facts.
Наблюдаемая сессия не становится actor: browser использует сохранённую сессию,
loadBrowser повторно AuthenticateID, каждый core read — readActor в transaction.

- Overview считает весь board; credential validity batch в той же transaction,
  статус credential_invalid выше TTL. Lease требует effective session, live lease,
  current task generation и singleton role generation. Native TTL-only Sessions
  сохранён. LastEventAt — batch SQL по max immutable cursor, не last_seen.
- ControlTasks выполняет literal Unicode lower-case substring по всему board
  до общей native pagination; фильтры/query входят в opaque cursor fingerprint.
  SQL activity использует bound args, filters до LIMIT+1, strict DESC cursor<before;
  native ASC Events не меняется. Ошибки не превращаются в пустой успех.
- JSON overview не идёт через page/issueAction/Describe/Heartbeat/ensureSession;
  возвращает только счётчики, no-store JSON/errors, auth401 и LEASE_LOST409.
  Host/admission/native Origin и прежние CSRF/forms остаются на прежнем периметре.

## Фактические проверки до UI freeze

- PASS: `.local/toolchains/go/bin/go test ./internal/store ./internal/web -run 'TestControl' -count=1`
  — store 1.349s, web 1.076s. Это synthetic unit/integration, не browser/process.
- PASS: targeted прежние web regressions CookieRotationRevokeAndLogout,
  OriginCSRFAndDraftBoundaries, ExplicitHeartbeatReleaseAndStaleDraft,
  GetDoesNotDescribeOrRenewAndContinueIsExplicit — 0.385s.
- PASS: targeted transport WebExtraStaysInsideHostAdmissionAndNativeOriginBoundary,
  NativeOriginPresencePolicy, ExactHostPolicyBeforeAuthentication — 0.372s.
- NOT-RUN: полный make/race, новые process/browser E2E, actual visual, live/deploy,
  physical second PC, нагрузочные гарантии. Это отдельные этапы координатора.

## Предварительные UI возвраты

До freeze автору переданы pending места, не объявленные финальными дефектами:
agent detail pagination scope; пустая overview Tasks секция без server projection;
pageshow после BFCache, чтобы pagehide не выключал polling навсегда; mobile
навигация без JS. В final freeze все четыре места исправлены и перепроверены.

## Итоговый blocking список

**R106-01 — P2: сохранить точное содержимое textarea при восстановлении формы.**
`internal/webui/templates/task.html:183–185` вставляет LF перед/после `answer`;
парсер HTML игнорирует один начальный LF у textarea, но завершающий LF остаётся
значением поля. При Draft.answer=`exact retained draft` получается
`exact retained draft\n`. То же форматирование есть у release reason
(`task.html:199–201`), readonly сохранённого ответа (`task.html:122–124`),
description (`create.html:25–27`) и recoverable draft (`layout.html:182–184`).

Сценарий: форма после validation/stale ответа без JS возвращает черновик с
добавленным переводом строки; пустой editor получает непустое whitespace-значение,
а ответ на границе16384 байт при восстановлении становится16385 и перестаёт
проходить прежний лимит. Это регресс существующих forms/draft semantics вследствие
переформатирования, не изменение продукта и не XSS. Требуемое исправление:
не добавлять template formatting whitespace внутрь textarea data; regression
проверяет byte-exact значения (пустое, обычное, намеренные переносы, HTML-символы,
предельную длину) во всех затронутых полях.

Доказательство — независимый unit probe через Go `-overlay`, исходники не изменены:
`/private/tmp/accord-control-099/review/textarea-overlay.json` и
`textarea-regression.json`. На точном freeze FAIL:
`got "exact retained draft\n" want "exact retained draft"`.
Проверка разбирает реальный renderer output и применяет правило одного initial LF;
это unit/source evidence, не browser E2E. Базовая форма a6501d8 держала данные
между `>` и `</textarea>` без добавленных LF. Root и UI104 уведомлены.

**Других блокирующих P1/P2 в согласованной дельте не обнаружено.**
Security-границы TASK-005 в проверенной реализации пройдены; общий code approve
до R106-01 не выдан. Это не разрешение пропустить обязательный actual UI/QA.

## Итоговый UI/security проход и evidence

- PASS: targeted `go test ./internal/webui -run 'TestControl|TestOverview|TestAgent|TestStats|TestBoard|TestAllPagesEscapeUntrustedContent|TestOwnedAnswerAndStaleDraft|TestStaticOnlyKnownAssets|TestHistoryUsesEventFacts|TestUploadCSRFFirstAndFileLast' -count=1` —0.353s.
- PASS: `node --test internal/webui/control.test.cjs` —6/6; syntax control.js и
  `git diff --check`. Тесты — pure Node unit harness, не browser E2E.
- PASS source: contextual html/template escaping, textContent для счётчиков,
  отсутствие raw HTML/JS/CSS casts/innerHTML/eval/CDN; fixed route helpers,
  exact embedded static allowlist, MIME/nosniff и сохранённый CSP. Forms POST,
  CSRF/action IDs/fencing/explicit editor heartbeat остаются прежними.
- PASS source/unit: overview scoped freshness, no fake empty task sample,
  session-specific detail paging, full counts, missing-token coverage; polling
  single-flight/timeout/abort/backoff/hidden/offline, terminal401/LEASE_LOST409,
  BFCache fresh read и last-good snapshot retention. Poll не заменяет draft DOM.
- PASS supply provenance: self-host Inter SHA256
  `693b77d4f32ee9b8bfc995589b5fad5e99adf2832738661f5402f9978429a8e3`
  совпал с `Paperclip5739a53/ui/public/fonts/InterVariable.woff2`; reference NOTICE
  указывает Inter4.1/SIL OFL1.1. Embedded copyright и полный OFL, Paperclip MIT
  доступны в footer. Static icons — собственные geometric paths по WORK104,
  динамические SVG данные не принимаются; сторонняя icon library не заявлена.
- PASS структура: затронутые текстовые source/test файлы ниже300 строк,
  native task filters переиспользованы через общий helper, повторяемые SSR rows/
  links вынесены. UI-kit Давинчи неприменим к самостоятельному Go SSR Accord.
- API contracts сверены по официальным
  [html/template Go1.26.1](https://pkg.go.dev/html/template@go1.26.1),
  [Fetch Standard](https://fetch.spec.whatwg.org/),
  [WHATWG history/pageshow](https://html.spec.whatwg.org/multipage/nav-history-apis.html#event-pageshow).
  Go module minimum1.26.1; фактически tests запускались Go1.27.2 darwin/arm64.
- PC-01 superseded: принят ответ «Подключение и контроль уже запущенных агентов»
  из конца контракта19.10; нынешний readonly контроль соответствует ему.

После узкой author fix R106-01 нужны affected unit tests + новый source SHA и
дельта-перепроверка reviewer106. Затем отдельный tester107 и независимый
дизайн-ревьюер выполняют actual browser/process/visual приёмку. Full make/race,
capacity, live HTTPS/deploy, physical second PC в этом review NOT-RUN.

## Журнал

- 2026-10-19T22:29:12+03:00 started — фактический date; профили и контракт прочитаны.
- 2026-10-19T22:31:23+03:00 review — backend manifest совпал, targeted checks PASS;
  backend блокирующих P1/P2 на этом этапе не найдено, UI final review pending freeze.
- 2026-10-19T22:35:03+03:00 completed — фактический date, общий freeze212/212
  совпал; один R106-01/P2 changes requested, security boundaries PASS в границах
  source/unit. Вне WORK и `/private/tmp/accord-control-099/review/` запись не велась.
