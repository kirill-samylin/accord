# TASK-005 — панель управления Accord по Paperclip

- Родитель: [REQ-005](REQ-005-paperclip-control.md); прямое поручение19.10.2026.
- Статус: **завершена и установлена**, этап выпуска принят.
- База a6501d8, ветка codex/web-control; production release **ac6c965**.
- Контракт: [control-web-contract.md](control-web-contract.md), DTO§8/уточнения§9.
- Paperclip source5739a53/MIT, Inter4.1/OFL; exact provenance/лицензии сохранены.
- Общую приёмку ведёт root099. Давинчи read-only, TandemOFF по прямому указанию.

## Результат

Рабочая боковая навигация, обзор проекта, действительные сессии агентов с текущими
задачами, список/доска задач, поиск по всему проекту, лента событий, статистика,
текущий проект и настройки. Светлая/тёмная тема, мобильное меню, self-hosted Inter.
Прежние вопросы/ответы, загрузка дизайна и история решений продолжают работать.
Автоматически обновляются только overview counters; roster/events имеют свои
времена чтения. Сессия/действительный доступ не выдаются за работающий OS процесс.

PC-01 закрыт прямым ответом пользователя: «Подключение и контроль уже запущенных
агентов». Текущий scope не запускает внешние процессы. Изоляция нескольких
проектов, agent profiles/provider execution, бюджеты/skills/integrations относятся
к будущим волнам WORK100. Единственный настроенный board честно показан одним
проектом; эпики не объявлены tenant projects. Полная Paperclip parity не заявлена.

## Участники и независимая приёмка

| WORK | Роль / результат |
| --- | --- |
| 099 | root coordinator/integrator/release — завершено |
| 100 | analyst —18-category inventory/AC/roadmap принят |
| 101 | architect — readers/DTO и будущие ACL/isolation границы приняты |
| 102 | design-reviewer — ранний source/pixel audit, не final approve |
| 103/104 | разные backend/UI authors — implementation/targeted tests завершены |
| 105 | security — ранний contract audit, не final source approve |
| 106 | independent code/security46fffb9 — P2 exact draftLF найден |
| 108/109 | другой author исправил draft; независимый delta APPROVEa30746c |
| 110 | actual visual a30746c — два P2 mobile width/status colors найдены |
| 111/113 | root исправил CSS/empty paragraph и epic E-N validator; fail-before proof |
| 112 | independent code/security batch APPROVEac6c965/source214 |
| 114 | independent actual visual APPROVEac6c965,8осмотренныхPNG |
| 107 | independent tester —24accepted checks в точных scopes |
| 115 | independent tests/release review — APPROVE; data5/poll4 собственные PASS |

Reviewer115 нашёл race тестовых clock.setup, tester107 исправил общий helper;
исходный FAIL сохранён, affected4/4 повторены tester и reviewer. Product assertions
не ослаблены. Reviewer не автор соответствующего исправления. Именованные модели
Claude недоступны; авторы/проверяющие — разные агенты с наследованной моделью.

## Фактические проверки

- Полный make check после Go113 PASS: fmt/vet/Go/native process acceptance,
  76infra tests/race/build. Python3.14 ResourceWarning сохранены, suite exit0.
- QA107: прежние14browser journeys один раз PASS; новыеdata5/layout1/poll4 PASS.
  Это24accepted checks по scopes, не выдуманный один монолитный прогон.
- Реальные Go/native/Chrome; SQL112task counts/search/pagination/filters/credential
  revoke/DESCactivity/stats. Poll offline/timeout/one-in-flight/retained drafts,
  actual headed hidden/visible, настоящие401/409 и terminalstop PASS.
- Final12test manifest SHA2563f9e0e2835fb68626328cd0425cc3cdfb0906483e35c4ce6a9e75dde97f55f8a;
  accepted evidence SHAf85892406334575cf418c9d88e81a94c505583fe3dab7a245381b6d3355bb217.
- Linux release из чистого detached ac6c965, source214 совпал сQA. VCSmodified=false,
  Linuxamd64CGO0; SHA1db9fee0874c3709b3688bd2a31c4e6b7ac28525760f0a4b91de4102fdd4a9ce.
  Mac QA binary имеет другую платформу/флаги, byteidentity не заявлена.

## Выпуск

27manifestfiles переданы явным Python USTAR без AppleDouble/extras; remote exact
file/hash verification PASS. Guarded stage actual nginx/systemd PASS; env/inventory
parity independently115 PASS. Guardedapply PASS; authenticated public HTTPS
browser/native smoke **36checks PASS**, включая все10разделов, cookie/Origin/auth,
font/license assets, control JSON/poll, темы/mobile и прежнийassetSHA/MCPmetadata.
Finalize PASS: consistent snapshot проверен, timers активны, postrestart readiness,
permissions иmanifest PASS, исходные private routes/Tandem/env точно сохранены.

Application source послеac6c965 не менялся: поздний commit включает QA/отчёты/docs.
Публикация проверенных codex/pilot и codex/web-control выполняется координатором;
main не меняется. Точный server origin, параметры иcredentials вGit не записаны.

## Практические ограничения

- Токен второгоПК проверен сMac; физический второйПК **NOT-RUN**.
- Старый mac.token сейчас отсутствует локально; этот credential smoke NOT-RUN,
  он не удалялся/не перевыпускался вTASK005. Доступ второгоПК сохранён.
- Реальные provider process execution, SLA/load и полная V1/Paperclip parity
  **не входят в поставку**. Пределы и следующие этапы — REQ005/WORK100.
- Browser/process source/evidence находятся в WORK107/115; приватные runtime
  снимки, токены, базы и параметры не копируются вGit.
