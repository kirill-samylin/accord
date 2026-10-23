# WORK205 — локальная подготовка согласия без Go-helper

23.10.2026, root. CONTRACT DRAFT / implementation NOT-RUN.
Обнаружен operational gap TASK014: экспортированные AutomationIdentity и
AutomationPresetDigest доступны QA helper, но установленный native клиент
не позволяет владельцу получить эти значения без сборки собственного Go кода.
Это мешает полной инструкции для второго компьютера.

## Узкий read-only контур

Новая команда `agent-board worker automation-preview --config PRIVATE`.
Читает только собственный private config теми же ACL/ownership/size/strict JSON
правилами, что LoadConfig. Не открывает Engine/state/token, не подключается к
серверу, не запускает процессы, не меняет config, consent или политику.

Владелец заранее указывает точные workspace/roles/roots/check executable/args,
время и лимиты в draft config. Preview получает canonical directory identity,
вычисляет hash выбранных Git/check executables и имеющихся dependency roots,
затем preset consent digest. Если уже заданный binary/dependency hash неверен,
это отказ, не автоматическая подмена pin. Пустой hash можно заполнить предложением.
В отдельной in-memory копии подставляет предложения и проверяет полный Config.Validate;
только полностью допустимое предложение выводится.

Вывод — bounded JSON с `applied:false`, nonsecret workspace aliases/preset IDs,
local versions, executable/dependency hashes и consent digests. Не выводит raw
config, токен, server URL, controllers, arbitrary args или provider config.
Контекст отмены30s проверяется между bounded reads/обходами; лимиты файлов и
числа entries существующих hash helpers сохраняются. Это не обещание жёсткого
таймера прерывания зависшего kernel filesystem IO. Вывод сначала полностью
валидируется и ограничивается1МиБ, затем печатается без partial-success статуса.
Сгенерированный digest сам по себе не означает согласия или включения. Владелец
проверяет и переносит значения в свой файл, затем отдельно выполняет существующее
`worker consent` и явно обновляет browser policy. Никакого automatic regrant.

Совместимость: прежние enroll/run/identity/consent и default-off automation не
меняются. Пропущенный automation блок даёт явную ошибку. Команда не создаёт
готовый check для неизвестного проекта и не выбирает стек/команду за владельца.
Git opt-in WORK204 и tester artifact flag остаются отдельными local decisions.

## Ownership и проверка

Root: новый worker setup-preview helper, отдельный cmd wrapper, короткая ветка
dispatch в cmd/agent-board/worker.go, pure unit/integration tests и docs.
Автор204 не меняет эти новые файлы. До implementation нужен independent202
contract review; затем SOURCE APPROVE новых файлов и unit результатов.
Новые process/E2E sources по-прежнему пишет отдельный QA после source approval.

Проверить private ACL/path/symlink/duplicate/unknown JSON, отсутствие token/state/
server/process side effects, correct digest/existing-pin mismatch, malformed roots,
changed executable/dependency contents, current flags/local roles, deadline,
bounded sanitized output и backward-compatible old commands. Ошибки без secret
values. Эта работа не заменяет actual native permission/pipeline/Windows QA.

## SOURCE READY для независимого review

Contract APPROVE202 получен до кода. Root реализовал6 Go files: setup preview,
bounded private IO/hash helper и author tests; CLI wrapper/test и dispatch.
Manifest `.local/work205/source.json`, SHA256
`0adcb75c5d136d07e342f197eb4a7d339d33f29d2bf1831ea55c6926c7c39129`.
Все новые файлы меньше300 строк. Config loader и author204 файлы не менялись.

Targeted pure-file/unit race PASS: worker1.440s, cmd1.408s. Проверены missing
hash proposals, exact completed consent binding, неизменность private draft,
отсутствие token/state (даже с несуществующими путями), отсутствие server/args
values в выводе, stale binary/dependency pins, unsafe roots/native grants,
duplicate/unknown/trailing/oversized JSON, permissions/symlink и cancellation.
Matching consent с nil dependency map сохраняется без изменений; первая unit
проверка обнаружила nil→empty normalisation, до final pin это исправлено.
Изменённый dependency content не заменяет старый pin автоматически.

Никаких новых process/E2E/native/config RPC/сетевых вызовов не выполнялось.
Actual CLI process/Windows остаются NOT-RUN; это не operational permission proof.
SOURCE APPROVE202 этого delta ещё требуется; docs с полным примером настройки
будут привязаны к принятой версии и отдельному runtime checkpoint.

Позднейший202 **SOURCE APPROVE** six-file manifest выше; independent hashes6/6
match и targeted pure unit race worker1.509s/cmd1.500s PASS. Root final full
Go race запущен после204 P1 final approval; общий итог ещё pending. Source
approval не заменяет CLI process/native/Windows runtime.

Root final full Go race после204+205 PASS: store102.833s, worker13.130s,
acceptance30.352s, web35.919s. Composite source pins сверены:29 нового scoped
delta и244 прежних source files без drift; prepared45files secret-scan findings0.
WORK199/202 превышают300 строк намеренно: сохраняют хронологию независимых
FAIL/review/GO без стирания свидетельств; новые product Go files меньше300 строк.
