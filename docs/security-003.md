# Security review TASK-003

Первичный вердикт: **REQUEST CHANGES**. Проверка этого этапа завершена возвратом
SEC003-01/02. Исправление infra и авторский freeze bridge ещё ожидаются;
после них нужен отдельный независимый recheck. Это не approve будущего deploy.

## Модель угроз

Доверены root/оператор VPS, UID accord и владелец клиентского токена.
Недоверены HTTP/MCP JSON, browser Origin/proxy headers и task text. Активы:
bearer и его hashes/backups, история/lease generations/idempotency, root-owned
release, доступность общей очереди и независимый работающий Tandem443.
Отдельные токены означают identity/revoke, а не scopes/RBAC/OAuth.

## Блокирующая находка

**SEC003-01 / P1: неполный отказоустойчивый переход release.**

На исходном прочитанном варианте `infra/accord/install.py:166–177` и
`infra/accord/rollback.py:76–83`: прежняя служба останавливается, snapshot
создаётся, current переключается, затем вызывается `systemctl start`.
Восстановление ссылки есть только внутри `if not healthy()`. Если start
выбрасывает CalledProcessError (невалидный executable/unit, отказ systemd,
предел запуска), условие не достигается и новая ссылка остаётся активной,
хотя обновление объявлено неуспешным. Ошибка snapshot после stop аналогично
оставляет прежнюю исправную версию остановленной до ручного вмешательства.
В общей очереди это отключает heartbeat и после TTL возвращает задания;
такое поведение не соответствует заявленной воспроизводимой процедуре отката.

Исправление: общий путь failure для каждого шага остановки/backup/switch/start/
readiness. До switch при неудачном snapshot вернуть прежнюю работающую службу.
После switch восстановить прежний managed current и запускать старую версию
только при подтверждённой совместимости текущей схемы; безусловный откат DB
недопустим. При первом install без previous — остановленное fail-closed состояние
и точная диагностика. Нужны изолированные failure-injection tests для backup,
start и readiness с проверкой current/команд, отсутствия DBrollback и чужих
service/Serve операций. Именованные состояния в документации должны совпадать
с реализацией. Координатор принял этот возврат.

**SEC003-02 / P2: boot guard не синхронизирует directory entry до restore.**

`infra/accord/restore.py:105–109` создаёт guard и fsync самого файла, но
синхронизация каталога STATE выполняется лишь после замены DB в конце
процедуры. fsync файла не даёт общего POSIX-обещания сохранности нового имени
в каталоге. При аварийном отключении до финального directory fsync нельзя
доказать заявленное восстановление после reboot с обязательным guard,
закрывающим старые credentials. Живой crash на VPS не воспроизводился.

Исправление: fsync каталога STATE сразу после закрытия нового guard и до
stop/замены данных; при отказе синхронизации не продолжать restore. Проверить
failure-injection/order: marker file fsync → directory fsync → stop writers →
замена DB; при отказе directory fsync ни stop, ни DB replace не выполняются.
После успешного reset удаление guard — отдельное осознанное действие root.

## Первичная проверка

В прочитанном варианте не найден auth/fencing bypass в новых native маршрутах,
client credential transport, bridge heartbeat или admin recovery. Это вывод
чтения и существующих synthetic tests; он не снимает finding выше и не является
утверждением об отсутствии всех уязвимостей.

Native guard отвергает присутствующий Origin до аутентификации, в том числе
GET/null/пустой/multiple header; SDK/CrossOriginProtection сохранены. Admission
использует один bucket и bounded semaphore, без attacker-key map; metrics
требует bearer и не содержит secret/task/session labels. Shared client запрещает
remote HTTP/credentials в URL и redirects/off-origin transport.

Bridge сериализует lease changes и heartbeat, не reacquire после uncertainty,
не принимает чужой session_id, не продлевает TTL историческим replay; EOF
останавливает pulse. Он физически не блокирует самостоятельные file/shell tools.

Admin recovery атомарно отзывает восстановленные токены, закрывает sessions/
attempts/leases, сохраняет fencing counters и audit; backup использует SQLite
snapshot с integrity/fsync, private stage и публикацию без overwrite. Offline
recovery требует остановленных writers; CLI само не доказывает их остановку.

Независимый `go test -race -count=1` пяти пакетов — PASS на Mac. Живая privateTLS,
MainPID/ownership/systemd sandbox, реальные backup/restore на Linux и сохранность
Tandem443 должны дополнительно проверяться после review до зачёта SEC01–12.
Физический второй ПК/Claude runtime/offsite NOT-RUN и не подменяются этим аудитом.

## Проверенная версия

Первичное чтение ещё не замороженного TASK-003; private manifest 72 файлов —
`.local/evidence/security-003/intermediate.sha256`, SHA256
`60efd1916c01629b8bf8bb5ff25d15c035b23fd3c93af317835456375aeb66df`.
Это историческая версия обнаружения, не финальный состав выпуска. Окончательный
hash/вердикт фиксирует отдельный recheck после freeze и проверки infra fixes.
