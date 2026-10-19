# WORK-089 — функциональный веб-интерфейс Accord

Роль: разработчик. Статус: на проверке. Этап: реализация → независимое ревью.
Дата: 19.10.2026. База worktree: `f063380`, ветка `codex/web-service`.

## Вход и границы

Профиль разработчика и общий процесс Давинчи прочитаны только как ролевые
инструкции; продуктовый код Давинчи не затрагивался. Прочитан самостоятельный
`agent-board/AGENTS.md`: Tandem отключён прямым поручением пользователя.
Именованные Claude agents/model bindings в этой среде недоступны; используется
унаследованная модель. Самопроверка не является независимым approve.

Вход: `docs/web-requirements.md` WEB-01..07;
`/private/tmp/accord-web-084/architecture/web-architecture.md` WORK-085;
`/private/tmp/accord-web-084/security/threat-review.md` WORK-086.
Пользователь разрешил функциональный SSR без отсутствующего макета Claude Design.
Новый дизайн, Sites, React, UI-kit Давинчи, backend/Git/deploy в границы не входят.
Запись только `internal/webui/**` и эта WORK; запись в отдельный Accord worktree
прошла с task-authorized escalation. Соседние файлы исполнителей сохранены.

Фактический frozen контракт согласован напрямую с WORK-087/088:
`web.Page` содержит `Draft map[string]string`, `Actions` и `FieldErrors`;
`DesignAsset` использует `AssetID`, `UploaderOwner`, `PreviewURL`.
Операции различаются action IDs; поддержаны exact `answer:T-N` и другие ключи.
Формы не передают principal/session как полномочие. Browser routes — `/web/*`.

## Реализация

- Go `html/template`, buffered renderer, embedded CSS/JS, без зависимостей.
  Контракт: `New() (*Renderer,error)`, `Render(w,status,name,web.Page) error`,
  `Static() http.Handler`; метод `Renderer.Static()` — совместимый wrapper.
- Русские разделы «Вопросы», «Задачи», «Дизайн», «История решений»; шапка с
  реальным проектом/участником, выход, фильтры, cursor links, пустые состояния.
- Вход личным password token или файлом до 4 КиБ. Токен не входит в Draft,
  выводимые значения или browser storage; клиент проверяет один источник.
- Реальные карточки/состояния, адресный claim только ready human task; ответ и
  release только при серверном `CanEdit`, generation из текущего Draft/Task.
  Чужое владение показывает чтение; stale draft остаётся копируемым.
- Создание task/epic: заголовок, описание, роль, эпик, приоритет, ответственный,
  области, review/tester target/commit. Значения и FieldErrors экранируются.
- Дизайны: multipart с csrf FIRST/action_id SECOND/file LAST; scope/key,
  дополнительная связь эпика, новое material/version, реальные metadata,
  SHA-256, download и preview только при выданном backend `PreviewURL`.
  Выбор файла после серверного отказа повторяется, остальной Draft сохраняет backend.
- История использует immutable `human_answered` EventData: ответ, автор,
  серверное время, question title, generation, epic, task и cursor reference.
  Legacy отдельно с честным отсутствием автора/даты/прошлых ответов.
- Opt-in heartbeat POST только у owned editor. ID обновляется после успешного
  tick; при неопределённом исходе прежний ID сохраняется для явного повтора.
  Hidden/offline/pagehide/form submit/error останавливают heartbeat. Превью
  не запускает HTML/SVG/PDF/ZIP; raw Markdown не интерпретируется.
- Answer/release используют progressive fetch для сохранения введённого текста
  при потере сети; 401/403/409 прекращают отправку и дают явный переход.
  Подтверждённый validation response может обновить action ID; 503 сохраняет
  прежнюю отправку. Без JavaScript обычные защищённые формы остаются рабочими,
  а кнопка heartbeat скрыта.
- Адаптация к узкому экрану, видимый keyboard focus, подписи полей, skip link,
  live submit/status сообщения, блокировка повторного клика. Сессия, которую
  надо явно продолжить, не изображается успешно загруженным пустым списком.

## Файлы и проверенная версия

Все исходники ≤300 строк. Проверенная версия — SHA-256 owned файлов:

| Файл | SHA-256 |
| --- | --- |
| `internal/webui/render.go` | `127ea40d09e5ff412631ca824aa53e46c7f2e5beebe2b4c76395b29f141c22e2` |
| `internal/webui/render_test.go` | `92cf129e6a15ff5cfe743d637294913003a64c210f6e6b646fe41ee921b73b49` |
| `internal/webui/templates/board.html` | `58544322a50d5aac5663cb394f97bb1418f6e31cc0f79ca15c84c9263f86e5db` |
| `internal/webui/templates/designs.html` | `b8571c25cb879fb14cf925f85f16e34766cb49f26226dc8dc7a4021c7543186e` |
| `internal/webui/templates/history.html` | `f6e4bfc90e16b5133a933a1eef1d5a0801dc578c78d3770b4f64026402faf839` |
| `internal/webui/templates/layout.html` | `4bc997f9df67c4d30e62c3b9993ea05227892b81be6deeb10c91eef790c6346c` |
| `internal/webui/templates/task.html` | `3f9988696b5a1c416a1e47cca2dc917b3d8b0283bf39d1bff4c1338303f62bc6` |
| `internal/webui/assets/app.css` | `df4ade6fc2e4fd0f3551ddb5484bf82b9ebda90fe7ea20d5a92ea6bddc208445` |
| `internal/webui/assets/app.js` | `3f9468faf5adf5dbec4d8478145d366720e28398f67562a7e287af061d4fea13` |

## Фактические проверки

PASS: локальный Go toolchain `.local/toolchains/go/bin/go`:
`go test ./internal/webui` (0.325s последний запуск).
`GOTOOLCHAIN=local`, caches/tmp — существующая `.local/toolchains/` основного Accord.
PASS: `node --check internal/webui/assets/app.js`.
PASS: gofmt owned Go files.

Unit-проверки: исполнение всех шести templates; XSS в title/description/answer/
filename/event; multipart порядок CSRF/action/file; own vs foreign editor;
stale draft copy; claim только ready human task; login без token echo;
history из event facts без reconstruction; filters/cursor; whitelist static
assets/methods; unknown view до записи ответа. Первый тест обнаружил слишком
широкую client claim predicate для waiting — исправлено и повторный запуск PASS.

Первоначальная попытка несуществующего Homebrew toolchain: NOT-RUN/exit127;
после уточнения пути использован actual локальный toolchain, без установки.

NOT-RUN: новый process/E2E/browser/mobile journey, screenshot QA и независимый
UI review — по процессу отдельный tester после независимого approve.
NOT-RUN: live/VPS/deploy, чужие backend/security integration suites.
Макета для визуального pixel matching нет; функциональный вариант согласован,
но независимая проверка состояний/адаптивности остаётся обязательной.

## Передача

Далее: независимый code reviewer и security reviewer текущей frozen версии;
после approve — tester для реального browser journey и design reviewer для
полноты состояний/keyboard/mobile. Backend продолжает own files параллельно;
общая готовность сервиса и итоговая TASK принадлежат координатору.
