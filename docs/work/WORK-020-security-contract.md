# WORK-020 — security-контракт общего сервиса

- Parent: TASK-003.
- Роль: security; отдельный агент `/root/securityplan003`.
- Статус: выполнено; результат — контракт, не approve реализации/deploy.
- База source: `966dda698da5e615b1517faec853c0b7271b2f9c`.
- Границы записи: `docs/security-contract.md`, этот WORK. VPS/config/Tandem и
  исходный код не изменялись; секреты не читались и не выводились.
- Профили: общий процесс и роль security из соседнего проекта, read-only.
  Модели именованных Claude-ролей в этой среде не закреплены.

## Входы и выполненная работа

Прочитаны AGENTS, TASK-003, requirements/architecture/API, source authentication,
fileprivacy, CLI token/client, HTTP/MCP transport и прежний server-readiness.
Сверены официальные Go/Tailscale/SQLite API и свойства.
Зафиксированы модель угроз, SEC-01..12, минимальные controls и способы проверки,
bootstrap/revoke, bridgeheartbeat, backuprestore и границы прав двух токенов.
Координатору переданы обнаруженные недостающие remoteOrigin/rate-limit/bridge;
DevOps переданы требования к serviceidentity, закрытым данным и backup.

## Находки и вердикт

Вердикт: **контракт подготовлен; новый remote deploy требует проверки**.
На исходной базе нет explicit Origin policy для safe methods, bounded
rate-limit, native поставки/backup/bridge. Они перечислены как конкретные
зависимости нового сетевого запуска. Имеющаяся bearer-auth не является OAuth,
scopes или защищённостью от локального владельца OS identity.
Актуального security approve новых артефактов в этом WORK нет.

## Фактические проверки

- PASS: чтение конкретных перечисленных source/docs, ревизия базы git.
- PASS: официальная справка CrossOriginProtection/Serve/Online Backup.
- NOT-RUN: тесты новой реализации и VPS runtime; они отсутствовали на старте
  контракта и входят в следующие review/test/deploy WORK.
- N/A: продуктовый UI/OAuth, это не реализованная часть данного контракта.

## Передача

Координатор/разработчики реализуют SEC-01..12 в соответствующих областях;
после этого новый независимый проверяющий сверяет hashes и исполняет
затронутые security tests. Этот документ задаёт требования, не снимает
блокирующие замечания будущего ревью и не меняет продуктовый объём молча.

## Журнал

- 2026-10-18T14:20:36Z — started; зафиксированы база и отсутствие infra outputs.
- 2026-10-18T14:20:36Z — source/read-only анализ; время получения evidence,
  окончание фиксируется отдельно фактическими часами.
- 2026-10-18T14:22:55Z — completed; уточнён обязательный offline recoveryReset:
  revokeall и инвалидизация восстановленных сессий/аренд с сохранением
  generation, до открытия сетевого доступа. Координатор передаёт реализацию
  следующему developer, независимое ревью ещё требуется.
