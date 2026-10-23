# WORK-158 — инструкция подключения GUI к Accord

22.10.2026. Исполнитель документации; общую TASK ведёт координатор.

Область записи по первоначальному назначению — эта карточка и
`docs/gui-accord-onboarding.md`. Координатор расширил её после прямого уточнения
пользователя: GUI `/Users/shagi/Desktop/Project/davinci-gui-`, Windows + Claude
Code у коллеги. Разрешены только GUI `AGENTS.md` (раздел очереди), `CLAUDE.md`,
два абзаца `docs/agents/parallel-work.md`, новый `docs/agents/accord.md`.
Private configs/credentials, production, `.mcp.json`/Codex TOML — область root.

## Контракт и результат

Прочитаны AGENTS, requirements/architecture, multi-project, coordination,
agent-profiles, TASK012, client-setup, second-pc/public-https, CLI mcp и bridge.
Подготовлена инструкция общего GUI: web-only, macOS/Linux Codex, Windows
Claude Code с окружением и восстановлением прежних переменных; точный `--project`, отдельный credential/membership,
актуальный capable binary, независимые bridge/sessions, leases/fencing,
дедупликация и проверка реального SHA. Старые клиентские архивы помечены
устаревшими. Токен/реальный origin/данные проекта в документ не записаны.
В GUI активные инструкции очереди переведены с Tandem на Accord. Существующие
dirty изменения AGENTS и прочих файлов сохранены; historical Tandem infra/docs
не удалялись. Удаление неиспользуемого подключения выполняет root. Точно
разделены управление задачей и отсутствующее удалённое управление OS-процессом.
Отдельно перечислены поддержанные роли Accord: GUI devops/security/
design-reviewer/llm-quality не передают как `describe.role` без явного выбора
поддержанной очереди координатором. Автоматический mapping не придуман.

Документ опирается на текущий CLI и контракты. Официальный Codex MCP формат
проверен координатором по `https://learn.chatgpt.com/docs/extend/mcp`;
проектная конфигурация применяется в trusted project. Claude project scope и
интерполяция `${VAR}` в command/args/env подтверждены официальной документацией
`https://code.claude.com/docs/en/mcp`; Windows setup прочитан по
`https://code.claude.com/docs/en/setup`. Совместимость с реальным
GUI-клиентом и Windows runtime этим чтением не подтверждена.

## Проверки

- PASS: сверка CLI flags и capabilities с текущим исходным кодом.
- PASS: документ меньше 300 строк, без реальных credentials/origin.
- PASS: `git diff --check` обоих репозиториев на текущей версии.
- PASS: active AGENTS/CLAUDE/parallel-work больше не требуют Tandem task_claim.
- NOT-RUN: реальное подключение GUI, scoped describe/reads и клиентский цикл.
- NOT-RUN: физический второй ПК и Windows runtime.
- N/A: новый product code и авторские unit/integration tests.
- NOT-RUN: новые process/E2E проверки; только после независимого approve,
  отдельным тестировщиком.

Независимый WORK160 reviewer: **SOURCE APPROVE**, включая дельту supported
roles; проверил `core/types.go:274`. SHA256 финальных инструкций:

- GUI `docs/agents/accord.md`: `f8e48b7466ab77d4d15292202646f7149e1d95a9772a93b2ac47904e552cdce7`.
- Accord `docs/gui-accord-onboarding.md`: `536b573898d4a189590a50a3ee65adca75028077f98ad72311fdde85f4bab19e`.

Статус: **DOCUMENTATION SOURCE APPROVED**. Финальный фактический статус
подключения/ID и клиентские проверки фиксирует координатор; этот WORK
не объявляет их PASS.
