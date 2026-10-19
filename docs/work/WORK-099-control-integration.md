# WORK-099 — интеграция и выпуск control UI Accord

- Роль: root, координатор/интегратор; модель наследуется. Родитель TASK-005/REQ-005.
- Статус: code/unit приняты; actual visual/browser/process QA в работе, выпуск впереди.
- Вход: действующий web release d9aa4c7, Git a6501d8. Текущий source a30746c.
- Владелец общей TASK/REQ/документации/release — root. Исходники authored103/104;
  независимые review106/109, ранние analyst100/architect101/design102/security105.
- Davinci read-only, Tandem OFF по прямому указанию пользователя. Нет имитации
  MCP task_claim. Проект самостоятельный, named Claude model bindings недоступны.

## Согласованный объём

Пользователь выбрал Paperclip как источник возможностей и визуального языка;
зафиксирован upstream5739a53, MIT notice и exact self-host Inter4.1/OFL provenance.
PC-01 закрыт прямым ответом: подключение и контроль уже запущенных агентов.
Текущий control UI наблюдает реальные сессии/доступ/аренды; не управляет процессами
и не выдаёт эпики за независимые проекты. Будущая registry/ACL/execution граница
описана архитектурой и матрицей возможностей, не подменена dummy controls.

## Реализация и независимая приёмка исходников

- Backend103: точный overview в одной transaction, эффективные sessions с batch
  credentials и актуальным task join, whole-project search до pagination,
  descending activity с SQL filters, browser-only data reader без renewal.
  Native12 и auth/Origin/CSRF/fencing/schema2+web1 сохраняются.
- UI104: Paperclip current streamlined sidebar, Inter/neutral tokens/density,
  light/dark/mobile, новые control pages, tasks list/board, polling только counters,
  прежние protected forms и app.js. MIT/OFL включены в embedded distributable.
- Review106 exact46fffb9/common manifest212 PASS кроме одного P2 R106-01:
  formatter newline менял exact textarea drafts. Результат CHANGES REQUESTED
  сохранён и не объявлен approve. Developer108 исправил пять закрывающих LF,
  сохранил initial sentinel согласно WHATWG; новый exact regression на11cases.
- Review109 exact a30746c/common manifest213 APPROVE узкой дельты. Targeted
  textarea и две UI-регрессии PASS. В совокупности106+109 разрешён QA107;
  никакой browser/live успех этими source заключениями не присвоен.

## Фактические проверки root

Полный `make check` PASS один раз на a30746c: fmt, vet, все Go unit/integration и
existing native process acceptance, 76 infra tests, все race packages, build.
Лог `/private/tmp/accord-control-099/make-check.log`. UI Node6/6 выполнены
автором/106 отдельно; их не выдаём за browser proof.

QA Mac binary (Go1.27.2 darwin/arm64, buildvcs=false) SHA256:
`0fb2f2a78623f659a2be46e68390fc3cd50296b9bb5ece5e1204e11f1d6a3aff`.
Source manifest `/private/tmp/accord-control-099/source-freeze-fixed.json` SHA256
`7afa42654df715f1df3ddb7b33cf18facf43f86bf3d0589261b779639c0ad4ad`.
Release binary будет собран отдельно из чистого checkout с VCS metadata;
равенство бинарных SHA разных OS/build flags не заявляется, source проверяется.

Действующий VPS release/hash и service/backup/cert timers read-only проверены
перед обновлением: PASS, изменения пока не вносились. Новый private deploy driver
отличается от прежнего reviewed только каталогом evidence; remote operator
SHA3c934e54af980dccad37878dcc980a939f9e66e532287199e64664830aefb697 unchanged.
Публичный адрес, токены и реальные private routes в Git не записываются.

## В работе и ограничения

- QA107: новые real process/browser tests после approve; старые14 browser кейсов
  адаптируются только к действительным UI changes без снятия domain assertions.
- Design110: actual synthetic fixture, source-derived reference comparison,
  отдельный независимый вердикт; не pixel-perfect claim отсутствующего current
  Paperclip whole-shell capture.
- После их приёмки: tests delta review, clean release, guarded apply, actual HTTPS
  browser/native/control/file smoke, новый backup и post-restart checks, push.
- Физический второй ПК, реальные provider execution и load/SLA здесь NOT-RUN;
  текущий scope не является обещанием полной Paperclip parity/полной V1.
