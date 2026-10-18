# WORK-034 — доступ к спискам, заметкам и статистике

- Родитель TASK-003; роль разработчик, координатор/root.
- Статус: реализация готова к независимому review вместе с WORK-032.
- Область: transport HTTP/MCP, bridge resources, CLI read consumers, их unit/
  integration tests; эта WORK. Общая карточка редактируется отдельно как root.
- Контракты: service-contract C, config-contract C/D; Evidence остаётся typed.
- Именованные модели Claude здесь не закреплены. Независимый reviewer обязателен;
  автор не выдаёт self-approve. Tandem OFF, сервер работает frozen99e722c.

## Реализация

12-й note tool; REST списки/notes/stats без увеличения tool budget;
MCP notes resource с bearer+board-session проверкой; bridge resource injection
только собственной сессии. Strict query parsing и CLI DTO decoding сохраняют
целые числа без float64 и отвергают неизвестные поля. Чтения не продлевают
аренду; отдельный worker timer продолжает явно передавать heartbeat.

18.10.2026 15:03:39 UTC: начата реализация; фактический timestamp окончательной
проверки и source manifest будут добавлены после freeze032.

## Проверки автором

- PASS: unit/integration transport/bridge/CLI, включая REST→note resource,
  repeat note idempotency, soft-delete provenance, strict query errors,
  worker-only resource session, настоящий CLI adapter read/notes/stats.
- PASS: race transport2.884s, bridge13.283s, CLI1.723s; targeted vet.
- PASS: git diff --check. Новые process/E2E не добавлялись автором.
- Freeze15файлов: `.local/evidence/transport-c-003/manifest.sha256`, SHA256
  `edf90092f95c535d3ab96b4d84ab63124f783008cd196525617c1465ea90e1cb`.
- NOT-RUN: независимое review, tester acceptance, C live schema2 deployment.

Фактическая отметка freeze:18.10.2026 15:08:36UTC. Native proxy fix находится
в отдельной рабочей копии; после интеграции его HTTP/MCP дельты manifest будет
обновлён и проверен reviewer, без распространения approve на новые файлы.
