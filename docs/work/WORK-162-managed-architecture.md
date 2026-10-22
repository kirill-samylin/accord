# WORK-162 — контракт управляемых агентов

Исполнитель: архитектура,22.10.2026. Область записи: только этот WORK и
`docs/managed-agents.md`; TASK редактирует координатор. Реализация не изменялась.

Прочитаны AGENTS, requirements/architecture/API, multi-project, profiles,
coordination, migrations/schema validation и capable updater. Проверены official
Codex non-interactive/CLI и Claude headless/CLI docs; local Codex help подтверждает
exec/resume/JSON/stdin. OpenAI Docs skill применён для native CLI контрактов.

Контракт фиксирует outbound worker с opt-in host/workspace/controller ACL,
durable commands/ACK, exact native session continuation, отдельный per-run board
bridge, stop дерева процессов и recovery ambiguous windows без двойного start.
Схема5 выбрана для обязательных durable таблиц; registry1/web2 и существующие
floors сохраняются. Production update только после отдельного review/QA с полным
all-project backup и roll-forward.

Уточнения по первой реализации: delivery на границе CLI turn, для немедленного
прерывания отдельный stop. Нельзя обещать произвольную live injection в уже
открытый IDE/desktop чат. Host controller policy согласуется с пользователем;
default — своим ПК, cross-control только после локального согласия владельца.

Результат source review: PENDING. Unit/integration/process/browser/native/Windows/
физический второй ПК: NOT-RUN, это архитектурная работа. Новые process/E2E тесты
пишет отдельный tester после независимого APPROVE. Деплой/GUI/Tandem/секреты не
изменялись. Исключений лимита300строк нет.
