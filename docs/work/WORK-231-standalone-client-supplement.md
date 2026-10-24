# WORK231 — standalone Codex/Claude intake supplement, source only

24.10.2026. Root scope только новый private `.local/task014/client231/` и эта
карточка. **SOURCE READY / LAUNCHERS NOT-RUN / NO INSTALL**. Применён OpenAI Docs
skill, просмотрены официальные Codex/Claude CLI/MCP страницы и actualAccord source.
NativeproviderCLI/help/configRPC, userconfig/token/providerenv не читались.
Текущий desktop chat/MCP не регистрировался; F6 runtime/globalconfig proof не тронут.

Пятьфайлов: client.json(metadata), launch-client.py(process-only), claude-mcp.json,
GUIDE.md, references.json(officialURLs+actualsource7hashes). Ничего не установлено
в HOME/realTest/LaunchAgents/server. client.json содержит только ссылки на ownMac
credential и будущий be95worker path, не bearer. launch_ready=false стопорит
запуск до фактических QA/install/serverupgrade/ownerpolicy/realprojecttrust gates.

Ключевые факты источников: bridge принимает --server/--project/--token-file,
**не --config/client.json**. Pythonlauncher только переводит своюmetadata в эти
flags. Standalone describe coordinator открывает отдельную session и обычный
workflow_submit/workflow_command; workflow_get доступен как readresource
accord://workflow?workflow_id=… (не вымышленныйtool). Managedchild не наследует
submit/command/callerOScontrol: backendACL/policy/hostconsent отдельны.

Codex -c mcp_servers.accord_test=…/--cd process-only, read-only coordinator sandbox
по explicitrootoptionalapproval, никаких normalapproval/model/autoapprove overrides.
Root толькоnamespace сообщил knowncomputer-use/node_repl; эти2точных имени
отключаются -c enabled=false. Сам QA globalconfig не читал. Effectiveplugins/
profiles UNPROVEN, полнойmanagedworker isolation не обещаем и Tandem не включаем.
Claude documented --mcp-config +--strict-mcp-config, обычные approvals; enforced
managedpolicy не обходится. Root/installedclient may need normaltrusted prompts;
пакет сам trust/globalsettings/registration не меняет.

GUIDE даёт полные futureownerMac install/start commands, источник настоящего
policyID/version UI, owner-only client_intake, exactfirstUSER_APPROVED_PYTHON_SAMPLE
prompt/describe machine, resources/currentversion commands и actualACK ожидания.
Windowsreceiver: личныйcredential/machine/privateACL, native.exe/checkout placeholders,
отдельные Claude и Codex processcommands с envтолькоcurrentchild/finallycleanup.
Macbearer/hostproof/state не копируются, editor не становится intake/OScontroller,
owner-only policy дляreceiverдоgrant закономерно403. Windows/Claude/физический
второйПК/currentdesktopregistration и отправкаworkflow NOT-RUN.

Freeze5members source
`365ced9bf96c9ab7bbb8c263a91e29a763a7e5f1f140e0ca2d95e5b1b1098d64`,
full `228abc38f08dc5e6c282c0c2b62f8066ae0d7706d435e64f19750e96265e8367`.
client e81145ce90f9193d3968ba9dc3d4b1509a55cf092bbffec3199be5778dd93356,
launcher ebcc148347948c4513818e4f3aa0e6dc48f3e18627d13720b100438bd6bcda28,
Claude cfcf006b19d7478df6875c680937e69f2e200b8e7c792f42c22dc6b58cae2758,
guide bd5f0becee960d7cd9a6ad88e1d6d30d2d07b88de5744a0e3bc524340fb204de,
refs d5695d20d5e96a7f7ead407875cb0b1d3b58b1412157a8ce561d201bf2a4380e.
StaticJSON/PythonAST/TOML+flags/data consistency/sourcehash PASS. Всеfiles0600,
root0700. Launcher/module/helper не выполнялись. Независимый202source review и
rootactualgates обязательны; preparedwrapper не доказательство подключения.
