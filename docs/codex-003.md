# Настоящий Codex с Mac → общий Accord

18.10.2026. Результат: **PASS,10 реальных MCP-вызовов** через отдельный
stdio worker и частный HTTPS-сервер. Это продолжение TASK-003, не полная V1
и не проверка второго физического ПК.

## Проверенная версия

- Server и native client source: `dc9c74cd692e287173def9fb5743dbf18ffdef21`.
- Linux/amd64 server SHA256:
  `301b4e7e330b69b6d0d3ff304ac717335dde1aaea63888f7cfc2ff300b240c0d`.
- macOS/arm64 client SHA256:
  `7108eef640f229099377e69eb6f31b2067657a2ed80d1947d187fcbd55c60cd1`.
- Codex CLI `0.163.0-alpha.10.2`, ChatGPT login; macOS27.0.1/arm64.
- Сервер Ubuntu24.04/amd64, systemd; частный TLS-прокси8443.

Проверена последовательность describe→create epic→create task→next→get→
heartbeat→release→transition cancelled task→transition cancelled epic→events.
Каждый вызов completed, error=null, бизнес-ошибок нет. E-1/T-2 отменены;
аренда generation1, next_cursor12. Итог клиента дополнительно сверён по JSONL.

Использован отдельный synthetic token. После теста он отозван; повторный запрос
получил401, локальная plaintext-копия удалена. Рабочий токен Mac сохранён
отдельно. Глобальные настройки Codex и очередь Tandem не менялись. CLI запущен
с per-run config, ignore-user-config/ephemeral, shell disabled и read-only
файловой sandbox. Изменения synthetic board явно авторизованы пользователем;
approval_policy=on-request, approvals_reviewer=auto_review только в этом smoke.

## Обнаруженная ошибка и повтор

На99e722c HTTPS readiness/REST работали, но MCP initialize получал403: SDK
localhost DNS rebinding guard отвергал Host reverse proxy. WORK035 добавила
явную exact-host policy перед auth; SDK guard заменяется только при проверенной
конфигурации, Origin/bearer/forwarded-header ограничения сохраняются.
Независимое code/security APPROVE: [WORK036](review-036-proxy-config.md).
После обновления raw HTTPS initialize200 и весь реальный сценарий выше PASS.

Первый локальный запуск harness также имел ошибку TOML override и завершился
до подключения. Это ошибка настройки теста, не автоматический approval rejection
и не бизнес-операция сервера. Неудачные попытки сохранены отдельно.

## Доказательства и ограничения

Private evidence: `.local/evidence/codex003/summary.json`, codex.jsonl;
SHA256 лога `72b3bed5884cb7b64b46a49d23bdf61f5e952562d1707845f151e20df3984160`.
Параметры MCP сверены с [официальным справочником](https://learn.chatgpt.com/docs/extend/mcp?surface=cli).
Длительное молчание/обрыв/EOF проверены отдельными process/SDK тестами;
этот10-call smoke не выдаётся за реальную работу модели дольшеTTL.
NOT-RUN: второй ПК, Claude Code, Windows runtime, OAuth и C/D/E live-функции.
