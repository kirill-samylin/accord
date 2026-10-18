# Независимая повторная проверка — TASK-002

18.10.2026. Reviewer: отдельный агент `recheck002`, WORK-016.
Вердикт: **APPROVE** проверенной дельты локального пилота после WORK-015.
Все четыре блокирующих замечания audit-002 закрыты. Это разрешает передать
дельту на финальный зачётный ретест; не означает завершение V1, переключение
VPS или подтверждение второго физического ПК.

## Проверенная версия

База `21cb3e0`, незакоммиченная дельта WORK-015 и контрактные документы.
Manifest 43 Go/build/module-файлов:
`.local/evidence/recheck002/source-manifest.sha256`, SHA-256
`699678fd6d9912d554ea10e4b663db5de84d8eda48d4c9652a54ec6f81343432`.
Включены существующие process tests; новые Go-тесты входят в manifest.

Отдельный manifest восьми документов AGENTS, README, requirements, architecture,
api, roadmap, second-pc и codex-002:
`.local/evidence/recheck002/docs-manifest.sha256`, SHA-256
`e383c21e71aeee03294b45e0f76317b1a665450c926d99c343a5d6f273cf1a45`.
Позднее изменение этих файлов требует проверки затронутой области.

## Закрытие замечаний

- **A2-01:** штатный pinned SDK разрешает две поддерживаемые им версии.
  Legacy HTTP initialize/initialized/list/call и SDK с принудительным
  2025-11-25 проходят; существующий SDK-тест основного протокола тоже проходит.
  Общий Store/авторизация не обходятся, самописного server handshake нет.
- **A2-02:** обычный и фильтрованный `next` требуют open-родителя. Три варианта
  cancelled-эпика не выдают lease и не меняют дочернюю попытку; явное open
  восстанавливает выдачу с первым generation. Каскадного удаления истории нет.
- **A2-03:** ненулевой generation в update/link/unlink всегда проверяет
  текущую lease через общий `owned`; requester/coordinator не обходят fencing.
  Регрессии проверяют expiry, неизменность task/session/events/idempotency
  при отказе и сохранение явно административного пути generation=0.
  Дополнительно reviewer проверил release/reclaim, см. ниже.
- **A2-04:** клиент читает limit+1, отклоняет overflow до stdout и сообщает
  неопределённость владения. Новые тесты используют валидный JSON размеров
  limit−1/limit/limit+1: первые два сохраняются полностью, последний отвергается.
- **Umask fixtures:** явные Chmod0644 устанавливают именно намеренно публичный
  fixture. Проверки production permissions не ослаблены; глобальный umask
  самим кодом тестов не меняется.

Новые regressions проверены не только на зелёной дельте: отдельный приватный
Go overlay вернул четыре production-файла к базе21cb3e0. Все четыре группы
падают по исходным сценариям (legacy notification400, cancelled claim,
stale mutation и response overflow). Для компиляции нового CLI-теста в overlay
оставлена лишь константа лимита; исправленное чтение ответа не переносилось.
Это ожидаемый RED, а не результат проверки текущей версии.

## Независимые проверки на этом Mac

- **PASS:** targeted store/transport/CLI regressions и permissions fixtures,
  `-count=1 -v`, umask077.
- **PASS:** то же существенное подмножество с `-race -count=1 -v`, umask022.
- **PASS:** приватный reviewer overlay, 12 сочетаний developer/coordinator ×
  release/reclaim × update/link/unlink. Старый generation отвергается с
  LEASE_LOST; задача и срок сессии не изменяются. Production-файлы не менялись.
- **PASS, 3/3:** process acceptance ConcurrentClaim, LeaseExpiry,
  RetryAndFilteredPagination; umask077, отдельные настоящие процессы.
  Бинарник SHA-256
  `b7abf6b5925b83a5cbba984b47d967735a5d3583bd2253ea8fe206a051234eee`.
- **PASS:** git diff --check. Изменённые Go-файлы ≤185 строк; новых
  существенных дублей, скрытого упрощённого MCP-пути или обходов типов нет.

Профиль reviewer и общий процесс переданы путями; автор и reviewer различны,
используется та же доступная модель. Закрепление именованных Claude-моделей
здесь недоступно. UI/design review — N/A, UI в дельте нет.

## Документы и реальный клиент

README/API/requirements/architecture согласованы с изменённой семантикой
generation, отмены эпика и MCP. second-pc требует одну общую БД/сервер,
не предлагает копировать SQLite между работающими ПК, сохраняет секрет вне
Git/config и не отключает SSH-проверку ключей или разрешения клиента.
Предпосылки HTTPS/SSH и непроверенный второй ПК явно названы. Windows указан
как возможный клиент с приватным ACL, без обещания проверенного сервера.

Отчёт [codex-002](codex-002.md) проверен как отдельное свидетельство
координатора. Его summary содержит четыре completed tool calls без ошибок,
одинаковый E-1 при create/get и cursor2. Reviewer не запускал этот реальный
Codex самостоятельно; SDK PASS не выдан за собственный real-client PASS.
Отчёт честно различает исходный protocol FAIL, ошибку настройки approvals
стенда и финальный4/4 PASS; объём smoke не назван полным этапом0B.

## Доказательства и ограничения

Локальный префикс `.local/evidence/recheck002/`, секреты не публикуются.

| Файл | SHA-256 |
| --- | --- |
| targeted077.log | ddee1cbd5b1daf5f2ac7c4187ac5a0a5356cd819cc451b135cfa62b1db6f7dca |
| targeted022-race.log | 4a7422e9787e4cebf2463ad9d967dd01658f1cb392d37ceecb4af2d095c31f32 |
| release-reclaim.log | f6ec319ee199fb5709de15ff0c27440127df755284f5496398b343f9d56a1682 |
| process-subset.log | 3eeb6e1dff0404c9143163806cc97b700df9968b896a9ce15bf36e44e4337d51 |
| baseline-red.log | 8b9467635cc4128efcfc4a81751688ef5ddbc5ac312e047f2681171dc93c9113 |
| release_reclaim_test.go — приватный overlay | 4a410876706d827051699cc211146d6bdb39ddb4cedb432c61b36450950a5c80 |

Полный make check и зачётный process-набор — следующий tester; этим reviewer
целиком не запускались. Отдельный security-аудит не подменён общим ревью;
закрытие file-security fixtures проверено точечно. NOT-RUN: физические два
ПК, Claude Code, удалённый Accord/TLS, Windows/Linux runtime, полная V1,
нагрузочные цели и fault injection. Tandem/сервер reviewer не изменял.
Рекомендация без блокировки: перенести приватную release/reclaim-матрицу
в постоянное regression-покрытие при следующем изменении fencing.
