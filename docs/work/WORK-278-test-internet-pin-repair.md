# WORK278 — восстановление запуска Test_internet после обновления Codex

24.10.2026. Координатор: root. Source baseline
`d2af9c882683da7067056351f72218b03b141659`.
Поручение: пользователь сообщил о неудачном запуске из Accord и приложил
скриншот ошибки local executable/workspace identity changed.

## Диагноз

Сервер и local journal согласованы: `mr_69cf2ea3e4eec251d2b368cc53e047d0`
failed/generation1/prepare_pins, native UUID отсутствует, pending0.
Проверка сработала до запуска native процесса. Codex по прежнему каноническому
пути обновлён с `0.163.0-alpha.11.1` до `0.163.0-alpha.11.2`;
provider identity изменилась с `1000011-54cd664` на `1000011-5592e74`.
Workspace identity `1000011-55cc618` осталась прежней.

Новый executable SHA256
`248ff9873b6fb35124a86bdffa013a09e1bd17f106385124e3dda73e9eb157ad`.
codesign verify strict exit0, OpenAI OpCo LLC/TeamIdentifier2DC432GLL2,
notarization ticket stapled. Installed worker SHA256
`f6291d57963620554da09f50f8908ec4b0fe735cc6e04d23aba160016780dadb`
сохранён. Binary/source/VPS этим восстановлением не обновляются.

## Ограниченное восстановление

Независимый review плана — [WORK279](WORK-279-test-internet-pin-repair-review.md).
До остановки подтверждены terminal-only local runs, pending0 и отсутствие
дочерних процессов у точного Test_internet worker PID. Другие Codex процессы
и GUI worker не затронуты.

Выполнены штатные bootout только `com.accord.test-internet.worker`, private
backup config/plist/full state, существующая команда `worker consent` и
bootstrap прежнего LaunchAgent. Consent exit0, bootstrap exit0, pins совпадают
с текущими identities. Config byte-exact; прежние runs/receipts сохранены,
pending0. Controllers0, workspace/provider allowlist, automation1, сроки,
бюджеты, policy, native permissions не расширены. State вручную не редактирован.

Private evidence и backup: игнорируемая `.local/task016-pin-repair/`.
Исторический failed run сохранён; его native разговор не существовал,
поэтому fake stopped/resume недопустимы. Повтор — отдельный новый run.

## Приёмка

Отдельная QA нового обычного managed launch —
[WORK280](WORK-280-test-internet-pin-repair-qa.md).
Fresh run `mr_f36f593ffcbe7f143bf1706c157a315b` прошёл running → completed;
native UUID `01a16e3d-629b-7261-b6e0-3c720baf71ff`, ответ
`PIN_REPAIR_LAUNCH_OK`, last_error отсутствует. Это реальный native Codex,
не SDK/stub или повтор failed разговора. Host version1→2/worker epoch4→5,
automation snapshot полностью совпадает после JSON decode. Новый binary сервиса
не собирался, поскольку изменялась только существующая local consent привязка.
QA: HEAD `bf77b6db0fadfc5a18da3045f0f4fd5bf510a314` и clean status до/после
совпадают; pending0, server active0, живых owned descendants/наблюдённых
process groups0. Прежний failed сохранён без native UUID. Независимый
review принял окончательные материалы: APPROVE в WORK279. Изменены только три
WORK-карточки этого восстановления; продуктовый source/config не изменялись.

Исходное задание на создание `c1.js` не повторялось. Оно было отправлено
обычным managed launch без task_key; для изменения файлов с планированием и
проверками предусмотрен экран «Выполнить требование» либо отдельная назначенная
задача обычного управляемого агента. Ремонт локальной привязки не равен
завершению пользовательского задания,
полной V1, Windows/Claude execution или второму физическому ПК.
