# WORK-088 — Дизайн и файлы веба Accord

Роль: developer. Родитель: TASK-004 / WORK-084. Статус: выполнено разработчиком,
передано на независимое ревью; это не approve и не закрытие TASK.
База: f06338089bec3b8c7f17f541c463131f5c98f96c; рабочая копия
`.local/web-service-084`, ветка `codex/web-service`. Контроль: 2026-10-19T18:12Z.

Прочитаны Accord AGENTS.md, требования web-requirements, web-architecture,
security threat-review WORK-084, общий процесс/parallel-work и профиль developer
Давинчи (только чтение). Tandem OFF по прямому указанию. Именованные роли Claude
и закрепление их моделей в среде недоступны; использована унаследованная модель.
Запись только назначенные `internal/web/design*.go`, `files*.go`, их tests и эта WORK.
Git, transport, UI, store, cmd, deploy и Давинчи не изменялись.

## Результат

- В общей web.db/schema1 отдельная таблица immutable design assets, pending
  reservations и ready версии. board.db/schema2 не меняется. Versions выдаются
  атомарно по группе; стабильная граница cursor выделяется при публикации, поэтому
  завершившийся позднее старый pending не входит в начатую пагинацию.
- Browser list/upload/download/preview, native bearer+live X-Session list/metadata/
  download; экспортированы Server.ListDesigns/GetDesign для MCP bridge. Все чтения
  перепроверяют живые token/principal/session, роли не превращены в отдельный ACL.
- Реальный multipart UI: csrf первым, action_id вторым, metadata до единственного
  последнего file. Включён optional epic_key: task принадлежит указанному epic;
  project key берётся из живой browser session. До обработки файла проверяются
  CSRF/action/live actor/scope. Есть отдельный малый бюджет на multipart headers.
- Stream ≤50 МиБ, тело ≤50 МиБ+64 КиБ, два upload одновременно, 120s deadline
  через ResponseController; unsupported deadline отклоняет upload/download.
  Никакого чтения файла целиком или ParseMultipartForm. Полный резерв50 МиБ до
  stream, квота1 ГиБ включает pending/ready/preview; свободное место≥512 МиБ
  с резервами проверяется до и по ходу. Оператору доступен DesignStorageStatus.
- Private0700/0600, случайные basename, os.Root, O_NOFOLLOW, O_EXCL, regular-file
  и owner checks. fsync→exclusive hard-link publish→directory fsync→ready DB
  transaction. Аудит successful/aborted upload в web_audit, без credentials.
- Startup удаляет partial/pending/orphan, освобождает резерв, проверяет размер/SHA
  всех ready оригиналов; missing/corrupt ready блокирует старт. Live staging cleanup
  через backend timer15min исключает активные pending. Ready blobs не удаляются.
- Exact request+canonical metadata+SHA replay возвращает прежний asset/version;
  изменённые поля/байты дают409. На format/validation400/413 выдаётся свежий action,
  uncertain5xx сохраняет прежний ID. Несекретные поля остаются в ошибке; файл выбирают
  снова. Ошибка после ready commit не удаляет принятый immutable файл.
- PNG/JPEG: sniff, DecodeConfig, ≤8192 на сторону и ≤16 Мп, полный Decode, только
  один decoder одновременно, новое JPEG preview≤8 МиБ. Preview входит в квоту;
  при отсутствии места под необязательное preview остаётся attachment.
  WebP без нового decoder — download-only; базовая RIFF container проверка.
  PDF/ZIP/HTML/SVG opaque attachment/octet-stream, nosniff+sandbox CSP; ZIP не
  распаковывается, SVG определяется по ограниченному XML prefix и не исполняется.

## Проверки

- PASS: `go test ./internal/web -count=1` — 17 design/file unit/integration tests
  плюс текущие тесты browser backend другого исполнителя.
- PASS: финальный `go test -race ./internal/web -count=1` после cursor/audit (8.559s).
- PASS: финальный `go vet ./internal/web` после cursor/audit, diagnostics нет.
- Сценарии: actual UI multipart с пустым epic_key; task/epic mismatch; exact50MiB
  accepted/+1 rejected; interrupted stream без quota/file leak; bounded headers;
  CSRF до file, unsupported deadline; quota24 concurrent reservations →20 admitted/
  4 rejected, disk floor/failure; symlink/traversal/control filename; PNG corruption/
  bounds, JPEG preview, active formats attachment; request/digest mismatch; версии,
  cursor в том числе late pending publication; restart/recovery/hash corruption;
  native auth/no-session/foreign-session/Origin/revoke; ready-файлы приватны.
- NOT-RUN: process/E2E, живой HTTPS/nginx, второй физический ПК, restore drill.
  Новые process/E2E по процессу принадлежат тестировщику после independent approve.

## Точные файлы

Все файлы ≤300 строк. SHA256 переданы для ревью текущей незакоммиченной версии:

- `internal/web/design_http.go` — `643ff6dc665d08aa85c13880fb4806c09f5e858c93355a7a32ab5d69094859ce`
- `internal/web/design_read.go` — `c2866f9d1d1354ca12e83adc9d9b0f4a698408f37e0685c0db98fe9c5ff87775`
- `internal/web/design_store.go` — `9680a1fe9b9210cd6c7ae9fb24251f69b714a7a64d7c67bd699f9ec38ef7c70e`
- `internal/web/files_budget.go` — `66c01807f43db75e096267516a167d5c258a4d14405b37becbd24f22ca941497`
- `internal/web/files_cursor_test.go` — `8fc4b348e0beb02f8d670ec24f47413dc96cd3142429edbdc0055fa2f71fcfd4`
- `internal/web/files_form.go` — `4babc3ffb1686765e33b93c065defd907fdeab42e0bf288e84926a0aa2315f54`
- `internal/web/files_format.go` — `93e4ed1c26f006d8fa0b2c357e804a9871d50b9008b6b22cf539b1673ece3cba`
- `internal/web/files_integration_test.go` — `2e65e4c005e930add6437f692a2c1748212739091b469a0ccf660ab6af62c45e`
- `internal/web/files_paths.go` — `ee5a141d6ac4cbff610232326778b4a0fd80962b631259c28c98de60b757297c`
- `internal/web/files_recovery.go` — `042633c7ac95e16222fbfa049e7e94e444c3df03994fdc001542f92af468196c`
- `internal/web/files_security_test.go` — `8456850b0ca826712022f1c6be9a6be7fbb01f32d9b3ededdc27723b2e7f24ae`
- `internal/web/files_test.go` — `9ccf5e65f3b9ee2a2fb921b1a206eb034092367dfba601d44741daaf2a0ddc35`
- `internal/web/files_upload.go` — `d5665b7678404992bd695fe2652781afa1ce0193334a04e7a9b85479fd90cc10`

## Журнал и передача

- 2026-10-19: получены границы WORK-088, согласованы hooks с WORK-087 и поля с089.
- Реализованы безопасные storage/metadata/routes и unit/integration проверки.
- Исправлены замечания early integration: optional epic_key и fresh action после
  confirmed format failure; оба сценария имеют regression. Проверки повторены.
- Следующий исполнитель: независимый reviewer/security на перечисленных SHA;
  координатор связывает результат с UI/transport/deploy и запускает E2E после approve.

FREEZE-088: 2026-10-19T18:13Z; финальные unit/integration/race/vet PASS, source SHA выше.
