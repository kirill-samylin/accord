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

## Исправление фоновых RSC prefetch запросов

Root расширил ownership на11Next UI файлов после фактического второго smoke:
автоматический prefetch видимых Next links создавал дополнительные `/app` RSC
запросы и расходовал общий прежний bounded API bucket. Это новое отдельное
наблюдение; первая asset429 причина и её исправление сохраняются.

На всех28существующих Next `<Link>` добавлен explicit `prefetch={false}`.
Навигация по нажатию, href, обработчики, query/cache и Go контракты не менялись.
Новых wrapper/абстракций нет. Native/web/asset nginx лимиты не изменены.
Применён прочитанный vercel-react-best-practices; актуальный App Router Link API
проверен по https://nextjs.org/docs/app/api-reference/components/link#prefetch :
false отключает viewport/hover speculative requests. Для динамического
authenticated UI с nonce это сознательное снижение фоновой сетевой нагрузки.

PASS: `make web-check` с Node24.19.0 и Go7892: формат, TypeScript,42/42unit,
standalone production build; `.local/task008/no-prefetch-check.log`. Статическая
сверка28/28links без пропусков. Independentreview и отдельный meaningful browser
no-prefetch regression — после передачи131/132. VPS автором не менялся.

Исключение размера: `features/shell.tsx` после форматирования explicitprops
имеет305строк (было297). Это только перенос строк в существующем компактном
shell; дробление UI в аварийном сетевом исправлении расширило бы scope без
пользы. Остальные затронутые файлы≤300строк; wrapper ради уменьшения строк
не вводится.
