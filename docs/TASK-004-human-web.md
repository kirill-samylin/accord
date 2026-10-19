# TASK-004 — рабочая веб-версия Accord для людей и агентов

- Статус: выполнено в согласованном объёме. Веб установлен и проверен через публичный HTTPS 19.10.2026.
- Источник: прямое поручение пользователя19.10.2026 доделать вопросы/ответы,
  загрузку дизайна, задачи и историю решений с фактической работой людей/агентов.
- Родитель: TASK-003. База: f063380, publicHTTPS к liveC2/schema2; Tandem OFF.
- Пользователь разрешил реализацию и развёртывание на существующем VPS;
  реальные токены/адреса остаются вне Git. Соседний Davinci не изменяется.

## Приёмка пользовательского результата

1. По существующему HTTPS-адресу открывается рабочий адаптивный веб-интерфейс.
2. Вход по отдельному действующему токену, безопасная browser-session,
   logout, истечение и отзыв без raw bearer в браузерном хранилище/логах.
3. Вопрос от агента виден людям; один человек атомарно берёт его и отвечает;
   ответ и история видны агенту. Живые чужие аренды не снимаются.
4. Человек создаёт эпик/задачу, назначает роль; агент видит и берёт работу
   через существующий MCP. Состояния и доказательства не подменяются UI.
5. Человек загружает дизайн, связывает с проектом/эпиком/задачей, видит версии
   и может скачать. Агент получает metadata/link и скачивает с авторизацией.
   Upload — новое разрешённое исключение прежнего REQ-01 о хранении ссылок.
6. История решений показывает вопросы/ответы и provenance с автором/датой,
   сохраняется после reload/restart. Ошибки/пусто/ожидание/конфликты видимы.
7. Проверены auth/CSRF/XSS/file safety/limits, browser user journey, native
   MCP совместимость, backup/restore новых данных и безопасный live upgrade.
8. Статус физического второго ПК подтверждается только реальным запуском.

## Границы

- Требуется полноценная работа перечисленных сценариев, не UI-заглушки.
- Минимальный функциональный веб по прямому поручению; отдельного ClaudeDesign
  экспорта Accord нет. UI-review проверяет поведение/состояния/удобство,
  не выдает сравнение с отсутствующим макетом за pixel approval.
- Не смешивать с невыпущенными D4/schema4/E integration branches. Избежать
  конфликтующих номеров миграций; решение архитектора до изменения БД.
- Native auth/Origin/idempotency/fencing/MCP12 и existing tokens сохраняются.
  OAuth остаётся отложен пользователем. Tandem/приватные маршруты сохраняются.
- Coordinator — единственный редактор TASK. Отдельные авторы/review/QA.

## Назначения

- WORK084 analyst: сценарии/состояния/acceptance/user-facing contract,
  запись только /private/tmp/accord-web-084/analysis.
- WORK085 architect: границы web/native/store/files/browser identity и
  crash/recovery/rollback, запись только /private/tmp/accord-web-084/architecture.
- После контракта: разработчики с непересекающимися файлами, независимые
  security/code/UI reviews; process/browser E2E отдельным тестировщиком.
- Профили читаются из Davinci read-only; named Claude mapping недоступен,
  модели наследуются, автор и reviewer — разные агенты одной модели.

## Реализация и приёмка19.10.2026

WORK084/085/086 контракты завершены. Авторы087backend,088files,089UI передали
реализацию. Independent090codeapprove baseline;091security+OS/browser delta,
092actualUI review. Root094integration и выпуск. Form delta093 approve; QA095 после approve;097review QA/выпуска.
make check PASS до финальной узкой platform/headerdelta; targeted+browser retest
для неё выполняются. Физический второй ПК пока user-reported, не agent-verified.


## Выпуск и итоговая приёмка 19.10.2026

**PASS:** четыре рабочих раздела, безопасный browser login, совместная работа
через native MCP, файлы/версии и сохраняемая история. На VPS установлен чистый
Linux release `d9aa4c745a4c2038ba22bdc7d92f89dded63ad7f`; board schema2 и
web schema1. Адрес и существующие индивидуальные токены сохранены вне Git.

- Независимые code/security/UI/form/QA/deploy/root-navigation reviews завершены.
- Полный `make check`: fmt, vet, unit/integration/process, 76 infra, race, build PASS;
  последняя root-navigation дельта — четыре targeted regression PASS.
- QA095: 14/14 browser/process сценариев и 2/2 сетевых проверок PASS, включая
  фактические 50МиБ, отказ на +1байт и server timeout120007мс; backup/restore и
  C2→new compatibility проверены. Финальная узкая дельта покрыта review098.
- Actual production HTTPS/browser/native smoke: **29 проверок PASS**. Реальный
  Chrome открыл основной адрес, вошёл обычной формой с файлом токена, открыл
  четыре раздела и вышел. Служебный PNG загружен через веб, скачан native
  клиентом побайтно с совпадающим SHA256; агент получил design resource metadata.
  Проверены оба существующих токена и12MCP tools, auth/Origin/cookie boundaries.
- Guarded apply/finalize PASS. Новый согласованный board+web+files snapshot
  фактически создан и проверен; daily timer активен, прежние backup/cert timers
  сохранены. После backup restart проверены readiness, public login, release hash,
  приватные права web storage и snapshot, а также точная неизменность Tandem,
  полной private Serve конфигурации и прежнего server.env.
- После backup restart authenticated download ранее загруженного файла повторён: SHA256 PASS.
- Клиенты для пяти платформ и инструкция находятся в отдельном локальном
  каталоге передачи. Старому MCP bridge требуется обновление бинарного файла
  для появления design resources; для браузера установка клиента не нужна.
- Scan373 tracked files + Git diff +5clientZIP: 0 совпадений с реальными machine
  tokens, SSH key, public origin/IP. Повторная проверка после итогового отчёта перед push выполнена: PASS. В репозиторий передаются общие инструкции, не приватные настройки.

Инструкция пользователю: [web-guide.md](web-guide.md). Фактический журнал выпуска:
[WORK-094](work/WORK-094-web-integration.md). Подробности и границы независимых
проверок — WORK090–098. Физический второй ПК остаётся **NOT-RUN** для агента;
проверка его токена с этого Mac не выдаётся за запуск второго компьютера.
Полная V1, D4/E, OAuth, offsite backup и отдельная SLA/load приёмка не входят
в завершённый объём этой задачи.
