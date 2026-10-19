# WORK-096 — эксплуатация и безопасная поставка веб-Accord

- Роль: DevOps; отдельный исполнитель `webdevops096`.
- Статус: в работе; staging завершён, фактическое переключение ожидает GO координатора после QA095 и независимого ревью097.
- Проверенная версия: `6245829b5db7c27534be421565dab5cddddf2573`.
- Область записи: эта WORK и private `/private/tmp/accord-web-084/deploy/`; source, Git, общая TASK, Давинчи, Tandem не изменяются.
- Унаследованная модель среды; именованная модель Claude-роли недоступна.
- Требования/эксплуатация: `infra/accord/web/README.md` reviewed release, текущие web requirements/architecture; профиль DevOps и общий процесс прочитаны.

## Сценарий поставки

Private `driver.py` передаёт только immutable reviewed release с SOURCE и полным SHA256SUMS, проверяя их локально и на VPS. Ключ/host key используются существующим pinned SSH transport; значения не читаются и не выводятся. Релиз передан в неактивный каталог, текущий symlink не переключён.

Private `remote_operator.py` поддерживает отдельные stage/apply/rollback/finalize. Stage проверяет кандидат nginx через временный главный конфиг с заменённым include и systemd-analyze verify без установки unit. Apply закрыт явным GO координатора, сначала делает старый online board backup, затем под общим maintenance flock сохраняет старые файлы/их отсутствие, symlink, исходное окружение и metadata baseline. Устанавливаются отдельный web.env, service drop-in без изменения ресурсов и старого server.env, reviewed nginx и web backup. После atomic symlink и readiness выполняются публичные root/metrics проверки. Каждый основной изменяющий шаг фиксируется в приватном журнале.

После apply оператор остаётся в pending-auth-smoke: координатор проверяет настоящий native bearer/MCP и браузерный вход. Затем finalize создаёт настоящий согласованный board+web+blobs snapshot, проверяет restart/readiness, включает daily timer и сверяет неизменность старых таймеров, исходного окружения, Tandem и полного Tailscale Serve. При ошибке apply/finalize запускается rollback. Ручной rollback получает тот же lock, возвращает бинарный symlink и прежние конфиги, включая исходное отсутствие web.env/drop-in/timer. БД не восстанавливается, новые private web-файлы не удаляются. Возврат исходной работающей службы находится в finally.

## Фактические проверки

- PASS: read-only live inventory — текущая native revision `1e9025cd6e3122f0adf6165e09845202736d004a`, служба active, прежний hourly backup result success, certificate renewal timer active.
- PASS: полный release SHA256SUMS локально и после передачи на VPS.
- PASS: настоящий nginx syntax test из isolated main config, настоящий systemd-analyze verify кандидатов.
- PASS: после staging исходные Tandem container ID/image/StartedAt/restarts/full mounts и полный Tailscale Serve JSON точно равны pre-web baseline. Current symlink неизменён, службы не останавливались, активные конфиги не устанавливались.
- PASS: синтаксис private операторов Python.
- NOT-RUN: реальное apply/rollback, authenticated live web smoke, новый live snapshot/timer — до explicit GO и приёмки QA/review.
- Independent review операторов передано отдельному reviewer097; самопроверка не является approve.

Sanitized evidence: `/private/tmp/accord-web-084/deploy/stage-result.json`. Полный baseline и диагностика private0600 в том же каталоге0700; адреса, origin, credentials и реальные данные в WORK/Git не включены.

## Журнал

- 2026-10-19T21:26:04+03:00 — started/staged: текущий live inventory и staging пройдены; подготовлены и заморожены операторы для независимого ревью097. Active switch запрещён до GO координатора. Следующий исполнитель — координатор после QA095/review097, затем DevOps для apply/postcheck.

### Дельта независимого ревью097

Исправлено замечание P2 к аварийному пути: возврат old current выполняется первым, восстановление каждого файла независимо, ошибки накапливаются; daemon-reload и попытка запуска прежнего writer находятся в finally. Сбой отдельного nginx-файла не пропускает восстановление drop-in или запуск службы. На изолированной файловой системе принудительный отказ записи nginx обнаружен, old binary/drop-in восстановлены, reload/start выполнены, новые bytes web.db сохранены; rollback корректно сообщает ошибку. Доказательство: `rollback-fault-result.json`, воспроизведение `check_rollback.py` в private каталоге. Это fault injection, не live rollback.

Замороженный оператор SHA256 `c56eb4cb604c189d4690b0a9522be0af31d980ce4e16ac38f563144d6338183e`; driver SHA256 `b709039116c969b8f774532b69bdd3e1ecddefe801321f23cd20f5ef58f5651d`. После дельты повторены настоящие stage nginx/systemd и полное сравнение исходного baseline: PASS. Оператор немного превышает ориентир300строк: это единая приватная транзакционная host-state-machine с общей логикой восстановления, декомпозиция увеличила бы количество зависимых переносимых файлов. Продуктовые source-файлы не изменялись.
