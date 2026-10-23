# WORK208 — совместимость native config proof без расширения прав

23.10.2026, root. CONTRACT DRAFT, implementation/runtime NOT-RUN.
Продолжение TASK014 после actual private diagnostic207V2.

## Наблюдённые факты

207V2 results `cddd2f6555a723661899b693887869440a99e15da1fdda6ef2973fa7aff8abe7`,
broker `d4898d881dfe31e0985d4cbb811fa431a554a4ab4481cee35a5383361494ef19`.
Три fixed responses/одна notification, native known natural0/drained, config
и source/binary pins unchanged, model0. Raw остаётся private0600 HOME.
Это diagnostic capture, не принятый product permission proof.

Root safe projection: initialize содержит четыре обязательных typed fields;
`remoteControl/status/changed` notification имеет optional `emittedAtMs` int64.
Оба элемента подтверждены schema установленного pinned native CLI.
Effective config выбрал expected profile, approval never, legacy sandbox null,
ровно пять известных disabled MCP, requirements null. SessionFlags profile
byte-semantic exact intent. Effective selected profile отличается только
добавленными map entries со значением JSON null. После удаления только таких
entries он exact равен прежнему expected profile. Никакие non-null/arrays/
empty maps, filesystem rules или network values при этом не изменяются.

## Узкий контракт

Сохранить default-off Git flag и все прежние authority/path/identity/managed/
MCP/effective-profile/два прохода guards. Не менять overrides, native mode,
global config, user defaults, scope прав, deadlines или число native calls.

1. Product config dialogue принимает optional `emittedAtMs` только на
   notifications: JSON integer/int64, не null/bool/string, без id/result/error.
   Принимаются только прежняя `configWarning` и наблюдённая штатная
   `remoteControl/status/changed`; параметры лишь bounded-discard, не логируются.
   Это пассивное чтение, никакой remote-control команды или ответа серверу.
   Unknown method/server request/reply ID/metadata на reply по-прежнему FAIL.
   Те же две passive notifications можно исчерпать до EOF после stdin close;
   любые ответы/requests после EOF остаются FAIL. Общие2MiB/frame,4MiB/35frames,
   outgoing8KiB,20s+7s cleanup и lifeline/owned exit не меняются.
2. При сравнении **effective selected profile** удалить только entries maps
   со значением JSON null, рекурсивно. Null — отсутствие установленного значения,
   не grant. Не удалять false/0/empty string/maps/arrays; не преобразовывать типы.
   После этого требуется прежнее полное exact equality expected profile.
   Любой дополнительный non-null grant/unknown non-null field/extra root,
   workspace root, наследование или network option по-прежнему FAIL.
   Raw layer projection/digest и managed constraints не ослабляются.

Нужны regression units: наблюдённая notification+metadata до response и после
EOF; malformed/int64 overflow/reply metadata/server requests/unknown methods/
response после EOF; effective typed-null defaults accepted, любой meaningful
extra и altered .git/config/hooks/network/extends rejected. Existing source
tests/race сохраняются. Никаких model/config RPC/process E2E автором.

## Ownership / gates

Root ведёт TASK/контракт. Separate author204 получает только
`internal/worker/config_dialogue.go`, `config_dialogue_test.go`,
`permission_projection.go` и новые `permission_profile_normalization*.go`,
свою private `.local/work208/` и эту WORK после CONTRACT APPROVE202.
Общие source manifests предыдущих кандидатов immutable; новый delta manifest.
Reviewer202 независимо CONTRACT/SOURCE review; QA198/206 только после source
approve готовит новый config-only fixture и process checks. Новые E2E sources
также отдельно review до root GO. После нового candidate ещё обязательны
обычный accepted permission proof, actual native commit, OS-negative, pipeline.
Production273/schema5 и GUI worker62e остаются прежними.

## Дополнение контракта до повторного source approval

После freeze первоначальных5 files root дополнительно проверил safe projection
207V2 против текущего `unexpectedPermissionKey`: effective config имеет один
дополнительный non-null key `include_permissions_instructions:true`, active
layers не имеют неожиданных permission keys. Имя подтверждено static installed
CLI constant и официальным upstream source, не скопировано из private values.
Это настройка добавления developer instructions, а не OS permission grant:
[ConfigToml](https://github.com/openai/codex/blob/main/codex-rs/config/src/config_toml.rs),
[packaged defaults](https://github.com/openai/codex/blob/main/codex-rs/config/defaults.toml).

Нужна дополнительная узкая совместимость: известный ключ допускается только
как boolean true; false/string/number/map/array по-прежнему FAIL. Null/отсутствие
сохраняют прежнюю обработку absent defaults. Если true явно присутствует в
active layer, его включают в raw relevant projection/digest; effective grant
validation не меняется. Никаких overrides для изменения этого флага, никакого
общего исключения `include_*`, неизвестные похожие keys остаются FAIL.

Сначала independent CONTRACT review этого дополнения, затем separate author
меняет только уже принадлежащий ему permission_projection.go, добавляет units
в свой normalization test, пересчитывает delta manifest. Старый pin00e724…
сохраняется как исторический freeze, не объявляется финальным. Native/runtime
по-прежнему NOT-RUN, повторного config invocation root не делал.

## Реализация author204 после CONTRACT APPROVE202

Автор `/root/automation_worker_impl`; SOURCE READY/FROZEN для независимого
review202. Base `8c43df9f408712526e223796905bba787e336d5e`. Изменены только
config_dialogue.go/config_dialogue_test.go/permission_projection.go и новые
permission_profile_normalization.go/_test.go. Предыдущие manifests не изменены.

- Optional emittedAtMs проверяется как JSON int64, с явным отказом null;
  наличие id запрещает эту metadata. Разрешены только passive configWarning и
  remoteControl/status/changed, без result/error. Params discarded, не выводятся
  и не инициируют response/remote-control operation. После закрытия stdin те же
  notifications дочитываются до stdout EOF с общими прежними limits; reply/
  request/unknown method/metadata остаются отказом.
- Effective selected profile сравнивается с exact expected после copy-only
  удаления null map entries рекурсивно. False/0/empty string/map/array значения
  сохраняются; arrays не преобразуются. Missing required grant после удаления
  null не равен expected и отклоняется. Raw layers/projection/managed policy и
  оба proof passes не нормализуются; overrides/consent/режим прав не менялись.
- Unit fixtures моделируют15 null defaults: accepted proof совпадает с прежним,
  raw layers не мутируются, а изменение raw layer nullable field меняет digest.
  Проверены meaningful extras и изменения config/hooks/network/extends/roots,
  null required grants, metadata до response и после stdin EOF, int64 extremes,
  invalid/null/bool/string/fraction/exponent/overflow timestamps, reply metadata,
  server requests/unknown methods/extra replies. Точный общий предел сохранён:
  3 responses+32 passive tail notifications PASS; +33 FAIL.

PASS: targeted compatibility race1.663s; итоговый общий race worker12.705s,
IPC1.277s, cmd2.853s. Linux/Windows amd64 worker test cross-build PASS,
`git diff --check` PASS. Все5 Go files≤300 строк. Новых process/E2E tests нет:
только pure units/static fixtures, прежние integration tests сохранены.

Delta manifest `.local/work208/source-manifest.json`, sorted relative path +
NUL + SHA256 + newline, 5 files:
`00e72442007dfc90f3a50d73564b8f513893f11dca586dd68b49331e4ee18b9f`.
Это source fingerprint, не установленная версия или принятый runtime proof.

NOT-RUN автором: config RPC/native/process/model/OS-negative/full pipeline,
production/GUI и physical Windows. Настройки/config/global defaults не менялись.
Обычный accepted permission proof и все последующие runtime gates обязательны
после независимого SOURCE APPROVE и отдельного QA/root GO.

## Реализация addendum после CONTRACT ADDENDUM APPROVE202

По отдельному root source GO изменены только permission_projection.go и
permission_profile_normalization_test.go. Exact include_permissions_instructions
допустим только как boolean true; false/other types/unknown non-null include_*
отклоняются. Null/absent обрабатываются как прежде. Explicit active-layer true
включён в raw relevant projection/digest; effective-only typed true не подменяет
grants и не меняет прежний digest. Overrides/native permissions не менялись.

Новые pure unit regressions проверяют effective true, active true с изменением
digest, nullable/absent совместимость, false/string/number/maps/arrays и похожие
неизвестные ключи отдельно в effective и layer maps без alias. Targeted race
PASS1.524s. Final общий race PASS: worker13.009s, IPC1.367s, cmd3.279s;
Linux/Windows worker cross-build PASS; diff-check PASS; все5 files≤300 строк.

SOURCE READY/FROZEN final addendum pin:
`c3d8785329db52dd4f2891d2543330a6f1228f2f65198742e55565d0d94a42d5`.
Текущий manifest `.local/work208/source-manifest.json`; старый00e724 manifest
сохранён неизменным `.local/work208/source-manifest-00e724.json`. Прежние WORK204
и другие manifests не редактировались. Независимый final source review202
остаётся обязательным; actual config/native/process/model/activation NOT-RUN.

## Уточнение root после P1 source review202

Pin c3d878… получил REQUEST CHANGES: blanket reject всех `include_*` создаёт
новую несовместимость с четырьмя unrelated instruction settings из actual
207V2. Root сформулировал предыдущее поручение неоднозначно. Требование
«не давать общего исключения include_*» означает сохранить прежний guard
permission/sandbox/approval keys, а не запретить все unrelated include settings.

Нужно удалить только generic include_ prefix reject. Exact
include_permissions_instructions true-only и active raw projection сохраняются;
unknown похожие ключи со словами permission/sandbox/approval по-прежнему FAIL.
Прежнее поведение прочих instruction fields не меняется. Pure regression
проверяет наблюдённые четыре true fields: include_apps_instructions,
include_collaboration_mode_instructions, include_environment_context и
include_environment_context_time. Actual configuration не переписывается.
После CONTRACT clarification APPROVE202 автор пересчитает новый manifest,
сохранит c3d878 как прошлый fail set и заново передаст final source review.

## Исправление P1 после CONTRACT CLARIFICATION APPROVE202

По отдельному root source GO удалён только generic include_ prefix reject.
Exact include_permissions_instructions по-прежнему true-only, explicit active
true остаётся в raw projection/digest, unknown permission/sandbox/approval keys
по-прежнему отклоняются. Прочие instruction fields получают прежнюю обработку;
прав/overrides/native/config изменений нет.

Pure units отдельно для effective/layer проверяют четыре наблюдённых true поля
include_apps_instructions/include_collaboration_mode_instructions/
include_environment_context/include_environment_context_time. Additional
unrelated non-null include_unknown_flag сохраняет прежние accept/proof bytes
для bool/string/number/map/array; похожие permission/sandbox/approval ключи
остаются negative cases. Targeted race PASS1.592s; final общий race PASS:
worker12.998s, IPC1.259s, cmd3.021s; Linux/Windows worker cross-build PASS,
diff-check PASS; все5 files≤300 строк.

SOURCE READY/FROZEN corrected final pin:
`10cd3bbc4754ceb675f0169e94e9bc74ec3721a2f454ee4d298c6d8c64e0b3c4`.
Новый `.local/work208/source-manifest.json`; c3d fail set сохранён неизменным
`.local/work208/source-manifest-c3d878.json`, исходный00e724 также сохранён.
Final source review202 pending. Native/RPC/process/model/activation NOT-RUN.
