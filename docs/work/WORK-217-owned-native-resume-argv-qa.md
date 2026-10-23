# WORK217 — actual resume argv собственного native child

23.10.2026, root. CONTRACT DRAFT; source/runtime NOT-RUN.
PLAN210C2 требует exact UUID не только в provider events, но и actual resume
argument. Public DTO/journal не сохраняют argv; поля выдумывать нельзя. Flattened
`ps args` не доказывает границы argv. Это отдельный read-only QA observer,
не изменение продукта и не управление процессами.

## Узкий Mac observer

QA gui_connection_qa после CONTRACT APPROVE может подготовить private helper в
собственной210C2 harness folder. Только Mac; exact живой worker.child Popen handle
является исходным anchor. Нельзя искать произвольный CLI или переиспользовать PID
после выхода anchor. Допускается bounded numeric process metadata snapshot без
argv/environment для восстановления только descendant chain этого worker.

До чтения argv требуется positive same UID, стабильные PID+birth/start identity+
PPID chain до живого exact Popen anchor и pinned native executable path/identity/
hash. После чтения те же metadata/executable facts проверяются снова; drift,
missing process/неизвестная связь дают FAIL/NOT-OBSERVED, а не приблизительный PASS.
PID используется только как аргумент readonly OS query; signal/adoption/pgrep/
global argv/чужие процессы и env вне scope запрещены.

Допускается version-pinned local `KERN_PROCARGS2`. Apple XNU публикует его под
`__APPLE_API_UNSTABLE`: это нестабильный OS interface, не обещание stable public
API. [Apple source](https://github.com/apple-oss-distributions/xnu/blob/main/bsd/sys/sysctl.h)
и exact installed SDK/OS ABI/header hashes обязательны; unsupported/denied даёт
NOT-OBSERVED без fallback/расширения доступа. Bounded buffer≤1MiB, deadline
observation≤20s внутри общего210C2 budget; numeric inventory count, poll count
и каждая OS query ограничены одним deadline. Парсится только argc и ровно argc
NUL-terminated argv после exec
path/padding. Environment tail никогда не parse/save/print; raw kernel buffer
немедленно discard и не попадает в logs/reports. argc/UTF8/terminators/размеры
проверяются строго, malformed/unsupported/denied — FAIL/NOT-OBSERVED. Не ослаблять
permissions ради доступа к query.

Сохраняется только bounded private0600 sanitized expected subset либо boolean
exact-match tuple: expected pinned CLI, `exec`/`resume` и точный gen1 conversation
UUID в правильной позиции, stable process identity/parent anchor. Не сохранять
полный argv, config values, paths/URL/secret/env. Public projection — только
same_conversation/resume_argv_exact/ownership_proven boolean и stage enum.
Source builder intent и UUID event являются дополнительными проверками, не
заменяют actual vector. Успех нужен отдельно от terminal native exit proof.

## Gates и ownership

Root owns этот contract/TASK; tester owns private observer source и свою C2 WORK,
reviewer202 independently reviews. Использовать official Apple/BSD/local SDK
definitions; не угадывать ABI/offsets/constants. Source preparation после
CONTRACT APPROVE, затем exact source freeze/SOURCE APPROVE. Самостоятельный
observer runtime не разрешён: только внутри separately approved210C2 scenario и
его root GO. GUI/Test/production/Windows/второй физический ПК вне этого contour.

## C2V3 BUILD ONLY — compiler exit0, acceptance FAIL warnings

После SOURCE APPROVE202 exact0ea75f3d… root разрешил и выполнил только сборку,
без единого запуска observer. Verified42sourcepins/14SDKheaders/OS/compilerSHA,
fresh private source copies; pinned clang version и compile через independently
reviewed215capture, bounded15/45s+cleanup8, known natural0/drained/!forced.
Compiler exit0, stdout0/stderr1413bytes: три signedness warnings proc.c19/41/64
и linker common-buffer alignment32768→16384. По контракту общий **FAIL**, artifact
не accepted и не исполнялся; flags не ослаблялись, warnings не подавлялись.

Receipt `.local/task014/observer217/build-1792785065593204000/build-receipt.json`
SHA074cb06dccae3300c05c631f632d1c0c0315c903537df7a002677946d907ab68.
Executed source/raw compiler streams retained private0600. Root narrow SOURCE
REOPEN C2V4: explicit validated positive PID/length conversions; explicit C11
alignment16 для byte buffer (argc читается memcpy, внешнего alignment требования
нет). Нужны независимое source review и новый BUILD GO; helper runtime по-прежнему
только внутри отдельно принятого C2/B2 сценария. 219/203 pipeline остаётся приоритетом.
