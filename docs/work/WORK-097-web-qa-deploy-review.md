# WORK-097 — независимое ревью browser/process QA и выпуска веб-Accord

- Роль: reviewer, отдельный от авторов095/096/root агент `/root/webqareview097`.
- Статус: **выполнено — APPROVE** новых process/browser tests, окончательной backup дельты, packaging и frozen операторов выпуска. Открытых блокирующих замечаний нет.
- Родитель: TASK-004. Проверенный release `d60070d818034faa5dc2e8f4d069fc869532e74d`; Go/web runtime source6245829 неизменён. Runtime QA на native macOS binary624 и final backup helper; deploy release — отдельный Linux binaryd600 с новыми VCS metadata.
- Запись reviewer: только эта WORK и `/private/tmp/accord-web-084/qa-review/`. Product/test/infra source, Git, live, SSH, Tandem и Давинчи reviewer не менял.
- Прочитаны AGENTS самостоятельного Accord, reviewer profile, общий процесс, parallel-work и code-structure Давинчи read-only. Tandem OFF по прямому поручению; named Claude model mapping недоступен, модель наследуется и совпадает с авторами. Отдельность автора/reviewer сохранена.

## Проверенный объём и вывод

Новые `tests/web/README.md`, `browser.cjs`, `harness.cjs`, `lifecycle.py`, `state-hash.py`, `upload-process.cjs`; финальная WORK095/evidence; поздние3строки `infra/accord/build-release.sh`; web дополнения README/handoff/receiver/second-PC/requirements/guide; дельта `infra/accord/web/backup.py`, README и2unit regressions; frozen private `driver.py`/`remote_operator.py`096. Code090/security091/form093/UI092 остаются отдельными заключениями, это ревью их не подменяет.

Harness использует настоящие Go server processes, OS pipes, stdio MCP worker и Chrome. Normal browser forms не подменяют Origin; explicit headers относятся к отдельным adversarial/deadline probes. Assertions проверяют наблюдаемые statuses, lease/generation/draft, actual answer/event, file bytes/SHA и авторизацию. Subsets маркируют scope; cleanup failure теперь FAIL. Process state/token paths изолированы, credentials не попадают в отчёты. Общий harness переиспользует lifecycle/native клиента; короткие multipart probes имеют разные входы и не требуют дополнительной абстракции.

Backup исправление сохраняет строгий manifest: source immutableRO разрешён только после проверки отсутствия непустого WAL; копия нормализуется DELETE, проверяются SQLite header1/1 и integrity/schema; только новые закрытые target sidecars regular0600 текущего владельца могут удаляться, WAL только пустой. Исходные данные/sidecars не удаляются. Остаток committed frames вызывает отказ вместо потери данных.

Новые authored codefiles и helper менее300строк. Private remote operator315 строк принят как одна переносимая транзакционная host-state-machine с общим аварийным восстановлением; исключение описано096. Новых существенных дублей/обходов типов нет. UI-kit Давинчи N/A для отдельного Accord.

## Закрытые замечания

1. **P2 rollback order.** Первоначальный096 мог пропустить old current из-за ошибки nginx config write. Исправление возвращает binary первым, независимо пытается repairs/reload/start/readiness, сохраняет errors и не выдаёт неполный rollback за PASS. Source перечитан, независимая fault injection PASS: synthetic nginx restore error не пропускает prior binary/daemon-reload/start/readiness; incomplete результат остаётся FAIL. Команды mocked, live не вызывался.
2. **P1 backup runtime portability.** На actual stopped Go fixture systemPython3.9/SQLite3.54 первоначальный backup воспроизвёл retained WAL/SHM; первая root попытка DELETE оставляла SHM32768 и тоже FAIL. Финальная узкая дельта закрыта source review, двумя independent unit checks и точным повтором ранее падавшей fixture. Snapshot8files + два validate PASS, sidecars отсутствуют. Strict whitelist не ослаблен. Исторические отказы сохранены095, за PASS не засчитаны.

## Лично выполнено

- **PASS два bounded subsets после FREEZE:** `browser-core` —4journeys (file login JS/basic POST,8desktop/mobilepages, human task→native12tools/describe/next+XSS, explicitheartbeat/read-no-renew/400correction→answer/history); real HTTP `byte-boundary-only` —52428800→303,52428801→413. JSON в `qa-review/browser-core/` и `qa-review/upload-boundary/`.
- **PASS** system `/usr/bin/python3` SQLite3.54: оба новых unit regressions sidecar-free WAL snapshots и refusal uncheckpointed WAL. Отдельный повтор exact stopped Go fixture с final helper + повторная validation PASS8files/noWAL/noSHM. Evidence `snapshot-fixed-results.json`.
- **PASS** frozen operator independent fault injection, реальные службы/SSH mocked; `operator-recovery-results.json`. Начальная ошибка собственного mock path mapping исправлена, за implementation defect не засчитана.
- **PASS** финальные QA095:14/14, SHA файлов tests/runtime source/binaries совпадают с captured manifests. Raw answer/event/schema hash финальной stopped state независимо пересчитан и совпал с before/after old/new значением. Два полного slow browser runs reviewer не повторял.
- **PASS** source provenance: primary cmd/internal/go.mod/go.sum не отличаются от f063380; те же пути1e902..f063 без diff. Old QA binary — настоящий прежний C2 source build с dirty documentation state, а не точный production Linux binary. New QA binary624 также dirty documentation/test state; source hashes проверены. Между624 иd600 изменены только helper, его docs иunit tests; Go runtime source одинаков.
- **PASS** release manifestd600:26files,13web/public assets; все local hashes/SOURCE проверены. От624 отличаются только SOURCE, binary VCS metadata, helper README и helper. Checksummed assets действительно включены3строками packaging, лишних/symlink файлов не выявлено.
- **PASS** exact new stage proofd600: nginx=true/systemd=true; full inventory равен прежнему frozen stage, полный Serve равен pre-web baseline, server.env hash сохранён. Реальный SSH/nginx/systemd запуск принадлежитroot/096; reviewer лично читал private evidence и сверял значения, SSH не запускал.

Последние test-only дельты после собственного subset: CLI reason строго `restore`, immutable read остановленной БД, дополнительные preview/anonymous/header assertions, сравнение old typed API по stable event identity. Static re-review выполнен; core/boundary выполненные пути не менялись, полный финальный прогон09514/14 проверен на exact текущих SHA. Full raw durable hash не ослаблен из-за законной old API projection.

## Доказательства и версия

| Артефакт | SHA256 |
| --- | --- |
| final reviewed source manifest |c06517c60799c382c1f1bde52793320ffef435a4e8120377064b596050158ff9|
| final095results14/14 |7bf1ea5fc7778be300e2b554bb701958b79f4d1b3c849b326d8b5bf2c8331ec2|
| final backup helper |0dc601d2c7acab4e53180d77e8bbd60cea869195c566ac3a605f8eba9d7d9b51|
| releaseSHA256SUMS d600 |c53846ace97ecea42d377bff609fede01350da3fafe580485870aa72fc7503df|
| Linux releasebinary d600 |32130ea65f265eaaba1297260c990ebca96a9674289a8f68d007d0e05d321bbb|
| frozen remote operator096 |c56eb4cb604c189d4690b0a9522be0af31d980ce4e16ac38f563144d6338183e|
| frozen driver096 |b709039116c969b8f774532b69bdd3e1ecddefe801321f23cd20f5ef58f5651d|

Source manifest: `/private/tmp/accord-web-084/qa-review/final-source.sha256`.
Остальные independent результаты: `review-subsets-manifest.json`, `final-qa-results.json`,
`final-release-results.json` там же. Финальный095 source/evidence manifest и14/14 JSON:
`/private/tmp/accord-web-084/qa/run-095-final-pass/`. Exact new private stage proof:
`/private/tmp/accord-web-084/deploy/stage-d60070d-proof-private.json`.

Отдельный120s run095 действительно завершён самим server через120007ms/ECONNRESET,
ready200; runtime Go binary неизменён. После assertion-only delta новые SQL/staging
postconditions выполнены отдельно: pending/deadline rows0, reserved0, staging0.
`qa/upload-final/upload-results.json` и `postcheck.json` разделены честно; поздняя
версия всего120s test source не объявлена выполненной. Timing input не менялся,
повтор120s не нужен. Browser31s run финального095 дал31027ms→303.

## Граница approve и передача

**APPROVE позволяет координатору начать guarded apply указанного d600 release** с
frozen операторами, затем выполнить actual authenticated native/browser HTTPS smoke
и finalize либо rollback. Operator сначала сохраняет старые config/absence/current,
делает board backup, использует общий maintenance flock; rollback не восстанавливает
БД и не удаляет новые web files. Новый daily snapshot сохраняет7копий, старые hourly
backup/certificate timer/server.env/Tandem/full Serve защищены проверками.

**NOT-RUN reviewer:** actual production apply/finalize/rollback, HTTPS browser smoke,
live daily timer. Это обязательные отдельные действия выпуска, данный approve не
объявляет их пройденными и не закрывает TASK004. Физический второй ПК NOT-RUN;
HTTP loopback двух browser contexts не является двумя компьютерами. Полный
 developer delivery/evidence/review cycle этим browser QA не заявлен. Offsite backup
не настроен. Платформенные archive checks и полный make check/race принадлежатroot,
лично повторно reviewer их не выполнял. Git main merge не входит в approve.

## Журнал

- 2026-10-19 — started: scope/профили прочитаны, независимое ревью начато.
- 2026-10-19 — returned096: P2 rollback order; исправление и isolated recheck закрыли замечание.
- 2026-10-19 — returned root: actual SQLite3.54 backup defect, первая correction недостаточна.
- 2026-10-19 — rechecked: final helper0dc601 + оба unit regressions + actual stopped fixture PASS.
- 2026-10-19 — checked: два bounded independent subsets PASS; final09514/14 exact manifests/raw durable SHA проверены.
- 2026-10-19 — completed: **APPROVE** exact released600/operators/source manifest; переданоroot для apply→authenticated smoke→finalize/rollback. Новое source/operator изменение требует затронутого re-review.


## Узкий повторный review private operator: nginx reload readiness

Вердикт: **APPROVE** frozen private operator SHA256
`3c934e54af980dccad37878dcc980a939f9e66e532287199e64664830aefb697`.
Driver неизменён, SHA256 `b709039116c969b8f774532b69bdd3e1ecddefe801321f23cd20f5ef58f5651d`.
Это одобрение только operator delta для следующей попытки выпуска; приложение
текущего d9aa release отдельно проверено WORK098 и здесь повторно не аттестуется.
Предыдущий operator hash c56eb4 в таблице выше относится к прежнему review.

Root передал факты actual apply failures/automatic rollback и transient root404
с readyz200 сразу после nginx reload. Reviewer SSH/live не запускал и собственным
runtime доказательством этих production наблюдений их не называет.

Дельта ограничена новой `public_ready(origin)` и её вызовами: после nginx reload
в apply вместо единственного assert и после backup restart в finalize. Обратное
текстовое преобразование этих трёх изменений даёт точный прежний reviewed SHA
`c56eb4cb604c189d4690b0a9522be0af31d980ce4e16ac38f563144d6338183e`.
Иных изменений оператора не найдено.

Guard требует фактический200: максимум20attempts, request timeout5s сохранён,
между попытками sleep0.5s. Это не обещание общего wall-clock timeout10s: время
отдельных requests добавляется. Persistent401/404 либо connection error не
превращаются в успех; после исчерпания attempts RuntimeError содержит только
числовые statuses. Auth/Origin/CSRF, native metrics404 и root pending-auth-smoke
этап не ослаблены. Redirect behavior существующего HTTP клиента не менялся.

**PASS6/6**, независимая isolated проверка без реальных IO/process/network:

- 404→200: успех ровно на второй попытке, одна задержка0.5s.
- Постоянный401, постоянный404 и OSError: по20попыток, затем RuntimeError.
- Ошибка нового guard в actual apply function вызывает rollback и не выставляет
  pending_auth_smoke/committed.
- Ошибка нового guard в actual finalize function останавливает backup service,
  получает mocked maintenance lock и вызывает rollback; daily timer не включается,
  committed не выставляется.

Source functions импортированы с отключённой записью bytecode; все paths,
filesystem mutations, commands, locks, signals и network подменены только внутри
private reviewer fixture. Продуктовая реализация не редактировалась. Существующие
rollback handlers и их finally-restart logic неизменны. Новых блокеров нет.

Воспроизведение: `/private/tmp/accord-web-084/qa-review/check_public_ready_delta.py`.
Результат: `/private/tmp/accord-web-084/qa-review/public-ready-delta-results.json`.
Следующий исполнитель root: перенести именно frozen operator, проверить его hash
и выполнить разрешённый guarded apply; затем actual HTTPS native/browser smoke и
finalize либо rollback. Повторный live apply этим reviewer **NOT-RUN**.

- 2026-10-19T18:54:57.133980+00:00 — rechecked/completed: public readiness operator delta APPROVE; independent6checks PASS; handed_off root.
