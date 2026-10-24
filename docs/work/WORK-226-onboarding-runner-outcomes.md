# WORK226 — onboard224 V2: truthful runner outcomes and approved sample

24.10.2026. Root передал managed_qa новую `.local/task014/onboard224/v2/`;
старый224/source9fcd… REQUEST CHANGES сохранён byte-exact. GUI/R14, проект,
HOME, credentials, Codex settings, сервер/production не читались/не менялись.
**SOURCE READY / REAL ONBOARDING NOT-RUN**, trust отдельно PENDING.

P1 исправлен fail-closed. Каждый test starts outcome=None, только addSuccess
устанавливает passed. Failure/error/skip явно сохраняются и сверяются с actual
unittest collections. Skip отражается skipped и даёт exit1. ExpectedFailure,
unexpectedSuccess, любой subtest, multiple outcomes, zero/incomplete/нет terminal
callback/class-setup error без matchingtest отклоняются до новой публикации
JUnit; wasSuccessfultrue не достаточен. Отсутствие report не считается PASS,
предыдущий JUnit не доказательство свежего запуска. Неизвестный исход никогда
не превращается в passed.

P2: USER_APPROVED_PYTHON_SAMPLE записан в README/plan. `add` в seed намеренно
NotImplementedError, те же fixed2tests unchanged. Первое требование: реализовать
positive/negative/zero integer addition одним реальным developer commit, затем
независимые review/QA exactSHA. Actual baseline честно2errors/exit1/JUnit2cases,
никакого pretendPASS. Workflow сейчас не создавался. Trust realproject всё ещё
требует отдельногоявногосогласия, QA219 не применяется.

Worker base/automation drafts, profiles, policy, LaunchAgent, publicCSRF routeplan,
AGENTS, fixedtests/.gitignore byte-exact224. Ownerimplicit/participant3неOScontroller,
7days/300step/1800workflow/12native и versionedconsent сохранены. README prospective
bootstrap: шестьfreshfiles O_EXCL|NOFOLLOW, samefd fchmod0644/fstat owner/nlink/size,
lstat inode/dev и readbackSHA, fullinventory под077, existingtargets не менять.
Executableinstaller не добавлялся; prospective plan не разрешает запись проекта.

## Изолированные author unit проверки

Root отдельно разрешил эти unit, без нового process/E2E/provider. Pinned Python3.14
SHA bd349815… выполнен с -I -B, свежий own repo-local scratch
`.local/task014/onboard224/unit226-5l9p81hk/`. Unitcopy6files под077 использует
exclusivefd/fchmod644/fstat/readback, полный inventory совпал; secondcopyreject
существующегоtarget без overwrite. Только synthetic in-memory unittest +stub
fixture, никаких nativeCLI/HTTP/MCP/model/Git/browser/globalconfig/realTest.

**PASS13 accountingcases**: twoSuccess, actualfailure, error, skip, expectedFailure,
unexpectedSuccess, passing/failing subtests, zero, missingstop, missingterminal,
classsetupError, multipleterminal. Дополнительно actual copied seedbaseline:
2tests/2errors/exit1 и2realJUniterrornodes, source bytes preserved. UnitPASS — это
проверка runner/отклонений, **не pipelinePASS**.

Private unitreport SHA
`d7befa595335270317ffe720e64abf1bf940756eeab46cf2679a09f8d1505e70`;
binding `e8469f49150770637d4af0d9b3ab803f4a3b4c192dc9423e41bd4d899f6094da`.
4Python AST/JSON/plist static PASS; max168Pythonlines. Все6scaffold source bytes/
actual0644 regular/no-links равны new targetmanifest
`c6d0c7b8e419f80703493e16457de75f72c5ba768697766a3d4d29fe14d56aa8`.

19members sourcefreeze
`a7f63db58cee7770764117cd8fd8c3d722bd684afbf82109af46f47d48eab854`,
full `b2ffc0fd6d8a8aa8f8bafbce03d73a04da362f9bd2f2e670a8e92ce3ce83ff64`.
Runner SHA `f39432af917904a3589fbba9c25d31784e51356262ad185e07b632ac527fedc2`.
Передано202 для полногоsource/evidence review; realonboarding/trust/apply/model/
production/push остаютсяNOT-RUN/закрыты отдельнымиgates, authorGO отсутствует.
