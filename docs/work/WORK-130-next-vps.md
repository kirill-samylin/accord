# WORK130 — Next VPS deployment developer

TASK008; область записи: `infra/accord/next/**`, `tests/infra/test_next*.py`,
`web/next.config.ts` только standalone, эта WORK. SSH/live/Git/TASK — root.
Прочитаны AGENTS, TASK007/008, next-web, requirements/architecture, исходные
Go service/public/web deployment contracts. Tandem и соседние проекты не используются.

## Реализация

- Next standalone с сохранением `.local/next`; buildtime Go7892. Локальная
  production сборка успешна Node24.19.0; зависимости не обновлены.
- Пакет immutable: server/static/public/licenses + официальный Linux Node,
  SOURCE/manifest с BuildID/hashes. Mac native исключены, exact-lock Linux
  sharp/libvips проверяются SRIsha512; другие nonLinux native отклоняются.
  Linux runtime/version/native load проверяются на VPS до active switch.
- Отдельный hardened accord-next UID без групп Accord, loopback7893; временный
  staging7894, heap192MiB/MemoryMax320MiB. Реальные data/env/backup недоступны.
- Узкий nginx transform сохраняет все bytes прежних native/ACME/web/upload
  маршрутов, меняет только exact `/` и добавляет `/app`, `/_next/static`, fonts,
  license. Upload прямо в Go; unused image optimizer не опубликован.
- Guarded stage/apply/finalize/rollback, protected manifest/config/pointer guards,
  existing web snapshot до переключения и после внешнего smoke, общий admin и
  maintenance locks без backup deadlock. Существующие env/units/timers хэшируются
  для контроля, не переписываются. Pointers поддерживают canonical relative и
  inventory-confirmed absolute managed target; rollback сохраняет исходную форму.
- Независимое восстановление pointer/nginx/unit и запуск прежнего Go даже при
  ошибке отдельного файла. Новые DB writes не восстанавливаются из backup.
  SIGTERM/INT/HUP запускают recovery; repeated signals игнорируются в recovery.
  При SIGKILL/питании нужен manual rollback по durable root-only journal.

## Проверки

PASS: standalone production build, Python compile, focused unit7/7: узость nginx
transform, неизменность Go/upload/Origin routes, invalid/repeated inputs,
absolute pointer lifecycle, rollback с принудительной ошибкой nginx сохраняет
новые DB bytes и запускает writer. Команды systemd/SSH в unit mocked.
Root дополнительно сообщил client format/typecheck/unit42 PASS. Файлы <300строк.
Новых process/E2E тестов автор не пишет.

NOT-RUN автором: Linux/VPS stage/apply/rollback, native/HTTPS/browser acceptance,
реальное измерение ресурсов. Это выполняют root/tester после independent review.
Продуктовый UI/DB/auth/domain не изменён. Фактическая публикация не заявлена.

## Исправление фактического smoke: static429

Root сообщил FAIL первой VPS-кандидатуры80486fc: inherited nginx20r/s burst40
применялся к Next chunks/fonts,7static429 оставили login без hydration. Root
вернул предыдущий release. Это не клиентский Go/auth дефект и не PASS deploy.

Исправление: отдельная bounded per-IP asset zone50r/s, burst100 только в
`/_next/static/`, `/fonts/`, exact license. `/app` по-прежнему наследует старый
20/40; native/web/upload/plaintext/ACME и global32connection-limit сохранены.
Наследование limit_req проверено по официальной документации nginx:
https://nginx.org/en/docs/http/ngx_http_limit_req_module.html . Никакие лимиты
не отключаются. Unit8/8 PASS: новый regression проверяет zone/триlocation,
отсутствие override у app и byte-identical native/web/upload/ACME. Python
compile/diffcheck PASS. VPS повторный прогон — NOT-RUN автором; передано root
и reviewer131. Ни SSH, ни Git, ни active config автор не изменял.
