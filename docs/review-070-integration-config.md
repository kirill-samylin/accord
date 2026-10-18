# WORK-070 — независимое code/security review WORK-067

Вердикт: **APPROVE** для pure offline config/portability на commit
`d1d3fb87d4c6d55895a9b0701dd6ae1112121e75`, base
`4bacf888a1eb28f32f3b8f2f8b1e0ce9a6c6c5e2`. Блокирующих замечаний нет.
Это не runtime/schema5/outbox/gates approve и не приёмка всего этапа E.

Ревьюер `/root/review070` независим от автора067; профили reviewer/security,
README и parallel-work Давинчи прочитаны read-only. Accord AGENTS и актуальный
MAIN `docs/integration-contract.md` прочитаны. Tandem OFF по прямому указанию
пользователя; Claude named bindings недоступны, модель унаследована без снижения
относительно автора. Исходники/тесты/шаблоны автора не изменялись.

## Версия и модель угроз

Сверены14 delivery files и2 unchanged templates по SHA256; все16 совпали.
Обе схемы byte-identical; HEAD соответствует назначению, worktree clean до/после.
Авторский manifest.head обозначает входную базу до commit; фактический HEAD
и пофайловая привязка зафиксированы отдельно в `.local/work070/config-freeze.json`.
Все5 hashes авторских logs также совпали.

Недоверенные входы — YAML/JSON config и JSONL identities/references. Активы —
явный repository trust, endpoint identity и отсутствие переноса полномочий.
Parse остаётся offline: новые вызовы CanonicalRepository/ValidateURL не читают
DNS/env/сеть/DB. Данные provider, project metadata и имя env не становятся
авторизацией, secret value либо readiness evidence.

## Проверенное поведение

- Explicit host prefix в config проверяется отдельно от короткой provider формы.
  ASCII/bounds≤256bytes, max16, case-insensitive canonical dedupe, `.git` и
  URL components проверяются. Enabled events требуют1..16 repositories;
  disabled events + absence/[] означают отсутствие доверия. D metadata
  `project.repository` сохранены и не дают trust.
- Endpoint использует прежний offline ValidateURL: canonical HTTPS/DNS syntax,
  no IP/credentials/query/fragment, default443 collapse, empty root→`/`.
  SHA256(`canonicalURL + NUL + secret_env_NAME`) проверен независимым фиксированным
  SHA oracle; subscriptions не разделяют duplicate identity. Только4 events.
  Public resolved-IP policy остаётся runtime sender обязанностью.
- GitHubEvents/Notifications по умолчанию false, непустые capabilities отклонены;
  OAuth не включён. Старые шаблоны не изменились; snapshots/Decode независимы,
  canonical round-trip и strict unknown/duplicate syntax сохраняются.
- Shared identity validator применяется к Parse/Encode и typed refs. Empty/NUL/
  >256bytes отвергаются до mapping callback;256byte multibyte ref принимается.
  Невалидный export не публикует finalfile и не оставляет staging. Stream Encode
  вправе уже записать manifest/предыдущие records; атомарность даёт ExportFile.
- Новых production зависимостей/SQL/HTTP/CLI/migrations нет; затронутые Go файлы
  21–193строки. Переиспользованы provider/notification helpers, нового
  существенного дубля или необоснованной декомпозиции не найдено.

## Фактические проверки

Go1.27.2 darwin/arm64; GOTOOLCHAIN=local, общий назначенный cache/mod/tmp.
CLI toolchain: `/Users/shagi/Desktop/Project/agent-board/.local/toolchains/go/bin/go`.

| Проверка | Результат |
| --- | --- |
| `go test -count=1 ./internal/config ./internal/interchange ./internal/integrations/github ./internal/notifications` | PASS4 packages |
| `go test -race -count=1 ./internal/config ./internal/interchange` | PASS2 packages;11.524s/1.474s |
| `go test -overlay=/Users/shagi/Desktop/Project/agent-board/.local/work070/overlay.json -count=1 -run TestReview070 -v ./internal/config ./internal/interchange` | PASS5 independent tests; config/interchange |
| `git diff --check BASE HEAD`, final worktree/frozen SHA checks | PASS |
| Авторские unit/race/vet logs и baseline-P3 negative log | SHA verified; прежний P3 baseline EXPECTED-FAIL, не текущий FAIL |

Own overlay лежит только в MAIN `.local/work070/`, source checkout не изменён.
Проверены combined config canonicalization/strictness, duplicate keys/unknown
new field, no metadata trust, endpoint delimiter/default port/subscriptions,
неканонические authorities, exact9007199254740993 evidence const round-trip,
multibyte typed-reference boundary/no callback/no publication. Последний
observational test отдельно фиксирует inherited repo grammar для `.`/`..`.
Логи: `config-independent-probes.log`, `config-race.log`,
`config-unit-tool-output.log`, `config-author-log-hashes.json`.

## Ограничения и отложенные замечания

Provider helper уже допускал repository path `.` и `..`; новый config переиспользует
его грамматику и также принимает эти строки. Это не новая возможность перейти
между allowlisted repositories: сравнение exact canonical identity и provider
HMAC не обходятся, DNS/URL по входу не запрашиваются. Согласованный ASCII контракт
не задаёт полную GitHub naming policy; ограничения длины реальных owner/repo и
специальных repo names здесь также не были запрошены. Поэтому это **P3 рекомендация**
для общего provider helper при следующей задаче этого модуля: явно отвергнуть
целые dot segments и добавить общий regression, если будет принято требование
строго пригодных GitHub identities. Текущий pure scope не блокируется.

**NOT-RUN:** SQL/schema5, real provider ingress/delivery/readiness/env secret,
outbox/CAS, restore/migration/runtime config apply, full repository suite.
Авторский vet PASS проверен по hash, независимо vet здесь не повторялся.

REV046-01 остаётся открытой SQL обязанностью: PR pointer version должен сравнить
Fact.Hash + normalized Match(task_key,reason) + Binding. Equal-time T1→T2 и
matched→missing/unknown/ambiguous должны дать order_conflict; этот approve
не закрывает persistence regression. REV046-02/P3 NUL export устранён и
независимо подтверждён в ограниченном library scope.

Следующий владелец — координатор root: принять067 отдельно от backup068;
runtime E допускается только после соответствующих performance/schema4 gates.
