# WORK-099 — интеграция и выпуск control UI Accord

- Роль: root, координатор/интегратор; модель наследуется. Родитель TASK-005/REQ-005.
- Статус: **завершено и установлено**. QA107/APPROVE115/guarded release приняты;
  source ac6c965, последующие изменения только QA/reports/docs.
- Вход: действующий web release d9aa4c7, Git a6501d8. Текущий source ac6c965.
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

## Последний пакет исправлений и freeze

QA107 обнаружил реальный HTTP400 для canonical epic E-N; root113 подтвердил
fail-before-fix и исправил раздельную E/T валидацию с negative unit regression.
Root111 исправил240px mobile drawer, status dot specificity и пустую рамку.
Independent112 APPROVE ac6c965; actual visual114 APPROVE,8PNG/source214 совпали.
Mac final binary SHA2569efa45893de16e369d3a12bd21add9ca9793f891993f229a5af0539250f25a27.
Manifest SHA256451e6888353900b1ba8016bdf2830c185b961b892d2749bb6fc83d10bae50a1c.
Повторный полный make check оправдан Go113: PASS, exit0, log make-check-final.log.
Прежний browser14/14 на этом binary PASS, новый control suite в завершении.
Linux release собирается в чистом detached ac6c965, поздние QA/docs не меняют
принятые production файлы. Review115 — новые tests плюс release/operator parity.

QA107 final acceptance24cases PASS по независимым scopes: прежний14 один раз,
новые data5/layout1/poll4. Кейс actual hidden/visible требует headedChromeCDP,
не JS override. Reviewer115 нашёл nondeterministic clock-pause race теста,
не app defect; tester107 исправил bounded fixed anchor до navigation. Poll4/4
повторно PASS; полный14/неизменённыйdata без причины не повторялись.
Final testmanifest SHA3f9e0e2835fb68626328cd0425cc3cdfb0906483e35c4ce6a9e75dde97f55f8a;
acceptanceevidence SHAf85892406334575cf418c9d88e81a94c505583fe3dab7a245381b6d3355bb217.

Clean Linux release ac6c965 SHA1db9fee0874c3709b3688bd2a31c4e6b7ac28525760f0a4b91de4102fdd4a9ce;
27files strict manifest4f79b2a9c4daad39008f138d41837b57d4c1e157e24005cc0022ba349136ab2f.
Python explicit USTAR transfer + remote exactfile/hash verification PASS; guarded
stage real nginx/systemd PASS, live_changed=false, inventory/env unchanged.
Существующий pc2credential с этогоMac даёт trustedHTTPS/tools12 PASS доupgrade.
Прежний localmac.token отсутствует (не удалялся в этой задаче); такой доступ
NOT-RUN, токены не переиздаются/не отзываются. Physical2PC остаётся NOT-RUN.
Secrets scan439workingfiles+82newhistoryblobs: доступный realpc2token, private
endpoint и privatekey markers не найдены; отсутствующий macsecret недоступен
для literalcomparison. Proof остаётся private, никаких values вGit.

## Выпуск и закрытие

APPROVE115 получен доapply. Guarded applyac6c965 PASS. Actual authenticated
public trusted HTTPS smoke36checks PASS: pc2credential сMac, anonymous401/Origin403/
cookieboundary/native12, все10browserroutes/controlJSON/poll/mobile/theme, exact
Inter/licensing и ранее загруженныйdesignSHA/MCPresource. Реальные task/design
фиктивно не создавались, usertoken не отзывался. Mac отсутствующийfile NOT-RUN.

Finalize PASS: новый consistency snapshot/validate_snapshot, timer preservation,
postrestart readiness, strict web/private backup ownership/modes, current
revision/binaryhash/manifest PASS. Исходные private Serve/Tandem inventory и
server.env точно сохранены. Private deploy/runtime evidence остаются внеGit.
После закрытия docs публикуются codex/pilot+codex/web-control (безmainmerge/force);
source production совпадает сac6c965, документы/тесты не меняют установленныйcode.
