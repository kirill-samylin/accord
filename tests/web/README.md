# Process/browser acceptance TASK-004

Набор запускает настоящий Go server и native stdio MCP worker, Chrome/Playwright;
HTTP/board/store не подменяются. Данные — только synthetic. Runtime, токены,
SQLite и JSON результата остаются в отдельном приватном QA-каталоге.
Предусловие: актуальный независимый code/security approve проверяемой версии.

## Запуск

Собрать текущий binary и прежний совместимый C2 отдельно. Экспортировать:

```sh
export ACCORD_QA_BINARY=/absolute/path/to/new-binary
export ACCORD_QA_OLD_BINARY=/absolute/path/to/old-C2-binary
export ACCORD_QA_DIR=/absolute/private/new-run-directory
export ACCORD_QA_PLAYWRIGHT=/absolute/path/to/node_modules/playwright
export ACCORD_QA_CHROME=/absolute/path/to/Chrome
export ACCORD_QA_PYTHON=/absolute/path/to/python3
export ACCORD_QA_BACKUP_MODULE=/absolute/path/to/infra/accord/web/backup.py
node tests/web/browser.cjs
```

Для отдельной real server границы файлов и 120s timeout — новый приватный
ACCORD_QA_DIR, те же binary/Python; `node tests/web/upload-process.cjs`.
Существующий directory должен быть0700; suite создаёт уникальные state/token
имена и случайный свободный loopback port. Не использовать production state,
реальные токены или публичный bind. Test process завершает только свои server,
worker и browser. Тесты не делают cleanup данных: JSON и synthetic fixture
сохраняются для независимого исследования. При повторе выбрать новый QA_DIR
(oversized file создаётся exclusive). После проверки весь QA_DIR можно удалить.

Быстрые review-subsets: `ACCORD_QA_STOP_AFTER=browser-core` (login,8pages,
human/native claim,answer/history), `browser-race` (добавляет20s expiry/race),
`ACCORD_QA_UPLOAD_BOUNDARY_ONLY=1` (exact50MiB/+1byte без120s). Результат содержит
scope; subset PASS относится только к перечисленным check IDs.

## Кейсы и ожидаемый результат

| ID | Действия | Ожидается |
| --- | --- | --- |
| login-real-file-js-and-basic-post | Два независимых участника входят файлом; отдельный JS-off context выполняет plain form POST |303, видимая identity, same-origin referrer policy без подмены browser headers |
| four-sections-desktop-mobile | Открыть вопросы/задачи/дизайн/историю на1280x900 и390x844 |200, четыре nav links, h1, нет document overflow |
| human-create-native-12-tools-describe-next-xss | Человек создаёт epic/developer task, native worker list/describe/next; HTML в title |12tools, точный task claim, текст экранирован |
| explicit-heartbeat-read-no-renew-validation-correct-answer-history | Агент создаёт human question; browserclaim/read/explicitbeat; пробелы→400; исправить безreload→answer |Чтение не продлевает; heartbeat продлевает; возле поля error/freshID; get done/answer и одинhuman_answered |
| two-browser-exact-claim-race-stale409-draft | Два context одновременно claim одной question; победитель ждёт server TTL, отправляетdraft |Точно303/409; stale409 сохраняет текст и блокирует submit |
| release-whitespace400-corrected-ready | Взять новыйquestion; вернуть с пробелами→400; исправитьпричину безreload | Near-field error/сохранённыйdraft; correctedrelease303/state ready безlease |
| design-browser-upload-3scopes-versions-native-resource-rest-sha | PNG(task),SVG версия2,HTML(epic),ZIP(project) через настоящие формы; agent resources/list/get + RESTdownload |Версии1/2, scope/key и SHA совпадают; originals attachment/octet-stream; одинPNG preview |
| auth-native-origin-csrf-foreign-session | Anonymous/wrongtoken/cookie-as-bearer; native Origin empty/null; badCSRF/missing/null browserOrigin; foreignresourceSID |401/403, worker не допускает чужой SID |
| browser-50MiB-limit-preflight | Выбрать реальный sparse file50MiB+1byte и submit |Видимый limit error, POST не отправлен |
| slow-stream-over-30s | Отправить ZIP multipart, пауза31s посреди body |303 после31s, override global30s работает |
| restart-browser-cookie-tasks-history-design-bytes | Остановить только свойserver; стартоватьновый process на тойжеБД; открыть прежнийbrowser |Cookie не отозван; tasks/history/assets сохранены; каждыйdownload SHA совпал |
| offline-snapshot-integrity-hashes-restore-auth-reset | Остановленнаяfixture; настоящийbackup.snapshot/validate; offlinecopy; prepare_restored_auth+native maintenance reset |Manifest/integrity/schema2+1 valid, browser/preauth/actions очищены |
| actual-old-C2-rollback-new-restart-events-webfiles | Запустить прежнийC2 binary на новойboard.db безweb; прочитатьevents; вернутьnew |Raw task answer+event+schema hash неизменен; всёweb дерево побайтово неизменно |
| restored-old-token-cookie-revoked-fresh-login-history-files | Стартоватьrestoredfixture; oldbearer+cookie; CLI выпуститьfreshsynthetic token иlogin |Oldauth отклонён; freshlogin видитhistory и скачиваетидентичныеfiles |
| server-inclusive50MiB-and-over-limit | НастоящиеHTTPmultipart50MiB и50MiB+1byte |303 и413 |
| actual120s-incomplete-upload-deadline | Valid metadata+4ZIPbytes безостатка; клиентне ставит120s deadline |Serverзакрываетоколо120s; ready200, pending/reservation/staging отсутствуют |

## Границы доказательств

HTTP fixture разрешён только literal loopback. Secure/HttpOnly/SameSite cookie
и fail-closed HTTPS configuration проверяются существующими Go tests отдельно;
production TLS/proxy и физический второй ПК этим набором не аттестуются.
Human→developer task кейс проверяет создание/видимость/захват, не полный цикл
выполнения developer/evidence/review. Независимый UI вердикт — WORK092.
