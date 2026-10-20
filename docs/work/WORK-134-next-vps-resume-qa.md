# WORK-134 — продолжение production QA Next.js

TASK008, 20.10.2026. Отдельный tester после прерывания предыдущего turn.
Владение: эта карточка и ignored `.local/task008-qa`; source, tests, infra,
SSH/VPS, TASK и Git tester не изменяет. Прочитаны AGENTS, TASK008, WORK131/132
и существующий независимо одобренный browser smoke. Новых тестов не добавлено.

## Проверенная версия и запуск

После explicit root GO выполнен неизменный `tests/integration/next_vps_smoke.cjs`
на действующем HTTPS release `490bf55e59b16c32817cb2ad4b9f7da7e1df13d9`.
Script SHA256 `88f749f7ceff6a13a93bd8c8c2f6539a3af30d1347c3f26b04d93cafc2700266`
совпадает с independent APPROVE131. Node24.19.0, Chrome155.0.8059.85,
отдельный browser context. Локально20.10.2026 14:09:06–14:09:32 МСК.

Отчёт: `.local/task008-qa/production-04/next-vps-results.json`, SHA256
`641ce01b2596811f88da96903eee223fdb870b8729dbe9840fab692500d46c1c`.
Исторические production01/02/03 FAIL сохранены и не перезаписаны.

## Фактический результат — PASS8/8

- HTTPS root направляет на `/app`; native без bearer401, public metrics404.
- Next CSP nonce совпадает с HTML; unsafe-inline/eval в script-src отсутствуют,
  no-store/nosniff присутствуют.
- Вход существующим токеном, HttpOnly/Secure/SameSiteStrict cookie с Path `/`;
  токен не попадает в localStorage. В tracked evidence значения не сохраняются.
- После viewport settle и hover фоновых prefetch0; явный click действительно
  получает JSON200 и открывает следующий раздел.
- Authenticated native read с собственной browser board-session и MCP12
  initialize/tools-list без describe и вызовов tools.
- Все10 экранов: light/dark ×1440/390;40PNG, Inter, document overflow,
  mobile drawer и command palette — PASS.
- Legacy SSR, JS/CSS/font/license assets, неизменность собственной session expiry
  после чтения — PASS.114 browser app/web responses, HTTP429=0;11 static assets,
  failed assets=0, page errors=0, severe CSP/hydration console errors=0.
- Собственный logout и последующий cookie401 — PASS. Единственные browser
  mutations — login/logout; tasks/uploads/leases не изменялись. Browser завершён.

## Визуальная проверка

Пересчитаны40/40 screenshot hashes из report — совпадают. В ignored runtime
созданы и фактически просмотрены два contact sheet, охватывающие все40PNG.
Отдельно в полном размере осмотрены desktop overview dark/tasks light и mobile
stats dark/designs light. Layout, контраст тем, меню/нижняя навигация и формы
не обрезаны по горизонтали; мобильные карточки сохраняют ширину viewport.
Focus outline заголовка соответствует реализованному фокусу после навигации.
В overview PNG зафиксировано промежуточное loading secondary task-list: эта
съёмка не доказывает отдельное завершение каждого secondary widget query.
Ответы app/web в итоговом сетевом отчёте успешны; новых runtime findings нет.

Реальные изображения, текст, paths и диагностика остаются только в `.local/`.
Tracked карточка содержит только версии, counts, statuses и hashes.

## Границы

Это production read-only smoke с одного физического macOS компьютера.
Chrome390 viewport не является физическим телефоном/вторым ПК. Реальные
task/upload/lease мутации, внешние Codex/Claude Code и полная V1 этим прогоном
не проверялись. Synthetic mutation acceptance остаётся отдельным TASK007.
Finalize/backup и окончательную publication closure выполняют root/reviewer;
tester их завершение не заявляет.
