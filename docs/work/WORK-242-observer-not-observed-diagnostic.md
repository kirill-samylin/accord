# WORK242 — bounded observer217 NOT_OBSERVED diagnostics

24.10.2026. Root поручил отдельный SOURCE ONLY packet
`.local/task014/observer217/v5/`; product/runtime/исторические freezes не меняются.
**SOURCE READY, BUILD/HELPER/PROCESS/NATIVE NOT-RUN.** B2V8 coverage FAIL остаётся
FAIL с UNKNOWN внутренней причиной. C2 actual OBSERVED не подменяет B2 proof.

База — accepted actual V4 receipt95079a8c… и binary96564be5…: скопированы и
сверены все шесть исходных файлов, ABI/header/compiler pins; оригиналы сохранены.
Новый C source добавляет только sticky first-failure enum и суммарные bounded
counters в NOT_OBSERVED. Последующие callers не перезаписывают первоначальную
ошибку; tick различает QUERY_LIMIT/DEADLINE/CLOCK. Finite stages покрывают
anchor/metadata/tree query или limit/chain/executable I/O или hash/argv query или
format/no exact match. Это место отказа, не доказанная OS-причина. SIGALRM
публикует лишь static DEADLINE_SIGNAL, без чтения counters/динамических данных.

Положительные ANCHOR_CAPTURED/OBSERVED printf+return блоки byte-exact. Negative
false/false и coverage FAIL сохраняются. Счётчики не содержат PID/argv/env/path/
errno/process rows. Временной лимит20s, query20000, poll32, node128, child256,
raw1MiB/argc1024 и все UID/birth/chain/fullhash-before-after/argv/errno/wipe guards
не ослаблены. Новых OS calls, retry/cache/hash skip/permissions/signals нет.

Статически PASS: positive blocks byte equality, OS/crypto/tick callsite counts
unchanged, прежние bounds/constants и V4 sources неизменны, каждый source <300
строк, private dirs0700/files0600. C syntax/logic НЕ компилировалась/исполнялась;
чистые C tests пока NOT-RUN до review. Нет providerCLI/help/config/token/sysctl/
native/model/browser/server/GUI/Testproject действий.

build-plan.json содержит исходный accepted clang/SDK/flags template с новым
source-path и намеренно незаполненным fresh output path. Нового executable build
framework нет. Нужны полный independent202 SOURCE APPROVE → отдельный root
BUILD GO → actual compiler-only receipt с before/after pins и новым binarySHA →
новый reviewed B2 binding → отдельный root scenario GO. Source packet не даёт
runtime-разрешение и не содержит нового executable.

Freeze18members `004e4d5fcf8a92324a78179ed322138489e7ba856c9ccc18d91ba921f0385c3b`.
SHA256SUMS `8384dca3124ee9919ab1566ae2a5e3061d90c66bb10d5a8637961596129d6714`.
Buildplan `f9b06984f1e7e7991e0e0b9fa33719d10a89aa22e779154b3e925ec2d5dfb49e`;
inputs `63b24ee9e8a7e602df83123d01a719097ec778c80dfc18220e1372397732ca56`;
staticchecks `2c783695154717a5ded8fce53397a72ed368fdba4477500536da4a02a12d5376`.
