# WORK183 — независимое ревью контракта автоматизации

Reviewer: managed_review. TASK014, docs/automation.md + WORK181 автора.
Область записи только эта карточка; implementation, canonical TASK и ops не
изменяются. Прямое поручение пользователя расширяет прежний manual contour до
полного analyst→developer→review→QA→fix workflow из UI либо scoped native MCP.

## Статус

**CONTRACT APPROVE.** Final contract300lines и author appendix252lines
независимо прочитаны; blocking architecture/security замечаний не осталось. Реальной автоматизации в установленном273 нет; source176 только
диагностика native preparation. Config-only PASS177 не подменяет native pipeline.

## Проверяемые границы

- Durable policy/grant связывает настоящих issuer/requester/host owner, текущие
  ACL/credential epochs/local consent. Нет сохранённого human bearer, fake
  coordinator session или Describe(role=human) как административного обхода.
- UI/native intake общий и идемпотентный; managed child привязан к своей task/
  step и не получает рекурсивный submit/admin/control другой работы.
- Аналитический plan валидируется и материализуется атомарно; независимые
  reviewer/tester conversations/sessions, task generation и actual Git SHA.
  CLI exit0, model-provided SHA, text event не являются проверкой результата.
- Local check IPC принимает только fixed preset IDs с fenced attempt/lease и
  durable execution receipt; stop/watchdog охватывают native и owned checks.
- Bounded retries/budgets/questions/control/recovery переживают restart и
  потерянные ответы; offline/accepted/unknown не называются applied/stopped.
- Schema6+automation floor, capable all-project upgrade/restore сохраняют
  данные и историю, не дают восстановленному workflow автоисполниться.

## Замечания к промежуточному draft

1. IPC/attestation протокол должен зафиксировать endpoint ACL+nonce, frame bounds,
   idempotency и fsync order, процессную остановку checks и удержание checkout lock
   до durable worker proof. Автор добавляет appendix; окончательный verdict впереди.
2. Fixed executable/argv не изолирует npm/python/test runner от repository code.
   Native sandbox не распространяется на worker-launched checks автоматически.
   Требуется owner-configured enforced isolation либо отдельный default-off явный
   trusted-code consent per preset/workspace; unsupported isolation блокирует работу.
3. Restore обязан инвалидировать automation authority и исключить auto-dispatch;
   unknown ownership не снимает reservation по времени. Refresh grant после policy/
   epoch change — только явный human action, без переписывания старых evidence.

До закрытия этих пунктов и проверки final DTO/proof appendix реализации нет
независимого APPROVE. Это contract review; unit/process/native доказательства
будут отдельными стадиями после source review реализации.

## Final draft181: закрытые и оставшиеся пункты

Авторский appendix231lines закрывает первоначальные замечания: private IPC с
nonce/ACL/strict frames, durable check receipts, lock/exit proof ordering, Git
source/artifact boundaries и parsed test reports; отдельное явное local consent
для unrestricted repository code, default disabled. Restore инвалидирует authority,
сохраняя историю и unknown ownership. Добавлен server-recognized per-attempt
ExecutionCapability; owner bearer остаётся parent, omitted binding не даёт unbound
Describe. Same-OS-user credential theft не объявлен предотвращённым sandbox.

Также согласованы per-role Codex/Claude bindings на одном consenting host и
редактируемая initial team через ordinary profile CRUD с partial/unknown receipts.
Отдельные physical PCs, platform-native evidence и future cross-PC Git integration
не подменяются этим контрактом.

Перед final APPROVE отправлены автору два узких validator/DTO пункта:

1. Срок policy упомянут в authority, но в DTO нет expires_at. Workflow grant duration
   не ограничивает срок повторного intake; нужен явный policy deadline, которым
   дополнительно ограничивается expires_at grant, без replay extension.
2. required_check_ids может быть пустым. Каждый developer task должен требовать
   хотя бы один разрешённый kind:test preset; пустой либо lint/build-only набор
   не допускает vacuous tester passed. Parsed report обязан содержать actual tests.

Остальная архитектура соответствует полному порученному контуру; для этих
уточнений не требуется новая user permission или сокращение объёма.


## Итоговый architecture/security verdict

**APPROVE** final frozen contract:

- docs/automation.md SHA256
  `772256519315a47541bd9827fe0bec579d8730d0ed95f1f9458aa22bcf5965e0`;
- WORK181 SHA256
  `6666bd34e559b6b4c3ef2bc2e10ea66e5325d28e9e330aa249de8f77881f4c9d`.

Оба final пункта закрыты: policy expiry24h/default, max7d и не позднее local
consent; grant=min всех сроков, replay не продлевает. Policy и каждый developer
plan task требуют хотя бы1 required kind:test preset, включая весь mandatory set;
zero/optional/lint-only наборы отвергаются. ExecutionCapability имеет explicit
DTO/register tuple, UNIQUE run/generation, bounded128active/4096total history.
Durable counters расходуются при enqueue один раз; истории/отмены не обнуляют
лимиты. Clarification/replan и human questions также ограничены.

Сохранён полный analyst→developer→review→test→fix contour из обоих intake,
per-role Codex/Claude на одном выбранном consenting host, права человека на
управление и recovery. Первичная editable команда использует обычный profile CRUD
с честным partial/unknown outcome recovery, не фиктивные готовые native workers.

Реализация допускается после распределения непересекающихся source областей.
Это **contract approval**, не source/runtime acceptance. Нужны независимое review
реального кода, отдельный QA новых тестов, actual native pipeline, negative fix
loop, контроль/epoch/restart/restore проверки, capable schema6 upgrade; их PASS
здесь не заявлен. Текущий GUI native172 остаётся FAIL prepare_discovery_timeout;
ProcessType гипотеза ещё требует controlled config-only184. Windows/Claude и
разные физические ПК сохраняют отдельные NOT-RUN границы.
