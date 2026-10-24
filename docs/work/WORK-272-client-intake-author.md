# WORK272 — native client intake: author candidate

Дата: 24.10.2026. Scope root: read-only CLI inventory и private launcher candidate,
без model/login/global settings/production mutations. Владение: эта карточка и
`.local/task015/client-author/`. Canonical TASK/guide и installed client не менялись.

## Проверено

- Codex installed CLI `0.163.0-alpha.11.1`, прежний bundled executable. Help/version
  exit0. Существующий installed WORK269 остаётся Codex-only.
- В PATH Claude нет; стандартные nvm/bin/native roots не содержат CLI.
  Настоящий bundled **Claude Code 2.1.308** найден в private desktop-managed
  executable cache. Это native CLI, не Electron GUI binary и не SDK wrapper.
- Version/help/auth-status-help exit0. `auth status --json` exit1,
  `loggedIn=false`, `authMethod=none`, `apiProvider=firstParty`.
  Raw auth response/identity/email не сохранялись; записаны только allowlisted
  статусы. Auth blobs/keychain/provider tokens не открывались, login не выполнялся.
- Модель, MCP tools, workflow и проверки этим inventory не запускались.

## Кандидат, не установлен

Private candidate состоит из прежних `client.json`, `launch-client.py`,
`claude-mcp.json`, с narrow изменениями:

1. Точный native executable и SHA обоих providers; ordinary executable check
   после literal-true selected-provider gate, до execution.
2. Codex прежние process-only namespace overrides и read-only sandbox сохранены.
3. Claude точный native path, `--restricted --tools "" --strict-mcp-config
   --mcp-config <private-config>`. Builtin tools отключены, user/project/local
   settings игнорируются; managed settings сохраняют силу. MCP tools отдельны.
   Это ограничение coordinator, а не новая ОС-изоляция.
4. Claude provider readiness **false**, auth не готов. Codex flag не включает
   Claude. Не добавлены allowedTools/approve/bypass/auto-review; не меняются
   global registration, approvals, model или provider authentication.

Реальное интерактивное испытание требует независимого source approve и root
запуска. Claude дополнительно требует обычного входа владельца. Если Codex
non-interactive MCP mutation требует approval при policy never, фиксируется отказ:
предыдущий synthetic grant describe/next не распространяется на production submit.
Нужен настоящий tool-call receipt из native transcript и совпадающий серверный
workflow; text намерения, bridge harness и config catalog не заменяют intake proof.

Оба intake могут запускать существующую Codex-команду: Claude как клиент не
означает Claude execution roles. Policy/host/profiles/deadlines/budgets не меняются.
Один checkout сериализует работу; одновременный запуск двух workflow не обещается.

## Источники и ограничения

Официальные страницы прочитаны 24.10.2026:
[Codex MCP](https://learn.chatgpt.com/docs/extend/mcp?surface=cli),
[Codex CLI](https://learn.chatgpt.com/docs/cli/reference),
[Codex non-interactive](https://learn.chatgpt.com/docs/non-interactive-mode),
[Claude CLI](https://code.claude.com/docs/en/cli-reference),
[Claude MCP](https://code.claude.com/docs/en/mcp),
[Claude headless](https://code.claude.com/docs/en/headless).
Installed help подтверждает candidate flags. Claude `--restricted` требует
v2.1.248+, найден2.1.308; `--tools ""` не убирает MCP tools. `--bare` не выбран,
поскольку он игнорирует subscription OAuth/keychain и требует API auth.

PASS: read-only inventory/help/status и private candidate preparation.
NOT-RUN: source review, installation, real Codex intake, real Claude intake,
client-origin full workflows, управление ими через Accord. Claude AUTH_BLOCKED.
