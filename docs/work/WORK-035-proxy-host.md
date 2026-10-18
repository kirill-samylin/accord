# WORK-035 — явный Host для частного TLS proxy

- Роль: developer; исполнитель `/root/proxyfix003`, запуск 18.10.2026.
- Статус: выполнено; этап: передача на независимое ревью.
- Основа: native `99e722c44f83b1808120bba1fabb04cf2f68a5b5` в изолированной
  `.local/worker-source-003`, ветка `codex/native-proxy`.
- Зависимость: root подтвердил реальный MCP403 через частный TLS, REST200.
- Границы записи: native transport/serve, installer/unit, тесты, связанные
  инструкции; без схемы/зависимостей, main WIP, live-операций, commit/push.
- Процесс и developer profile прочитаны из соседнего проекта; модель роли
  Claude здесь не закреплена, используется модель родительского агента.

## Причина и решение

Official Go SDK v1.8.0 `mcp/streamable.go:174,317` защищает loopback listener
от внешнего Host. Tailscale Serve передаёт настоящее частное DNS:port, поэтому
транспорт SDK возвращал403 до initialize. Отдельный TLS/REST health этого
дефекта не выявлял. Источник API — зафиксированный исходник official SDK,
не неподдержанная модификация библиотеки.

Согласованный с координатором контракт: `HandlerOptions.ExternalHost`,
`serve --external-host` либо `ACCORD_EXTERNAL_HOST`. Только одна canonical
полная lower-case ASCII DNS authority с explicit портом. Запрещены wildcard,
URL/userinfo/path/query, пробелы, неcanonical IP/port и отсутствующий порт.
Все пути проверяют Host до auth: разрешены canonical literal loopback,
localhost и точная внешняя authority. Forwarded headers не учитываются;
дубли Header.Host и конфликт absolute-form URL authority отвергаются.
Неверный HandlerOptions возвращает HTTP500 без создания SDK handler;
CLI прекращает старт до открытия listener/БД. SDK guard отключается только
при непустой validated внешней настройке за этим фильтром. Без неё остаётся
включённым. Origin/bearer/admission не ослаблены.

Installer получает явный `--external-host`, сохраняет одну managed строку в
`/etc/accord/server.env` root0600/каталог0700, атомарно и с file+directory fsync.
Не выводит имя/секреты. Omission сохраняет existing настройку. Чужое содержимое,
symlink, режим/владелец, foreign staging вызывают отказ. Systemd читает
optional EnvironmentFile; адрес клиента в исходники не включён.

## Проверки

PASS: полный `make check` на native99e + текущем diff: fmt, vet, все Go tests,
24 Python tests, race всех пакетов, build. Race: cmd1.682s, bridge12.973s,
client1.346s, store6.762s, transport2.901s, acceptance15.957s.
После узкой правки обработки не-ASCII чужого env-файла повторно пройдены
5 host_config tests и все24 Python tests. `git diff --check` — PASS.
Targeted transport/CLI перед этим — PASS.
Python3.14 выдаёт ResourceWarning в прежних recovery SQLite fixtures;
новые host_config tests проходят. Ошибка первой собственной test fixture
(nil Service вместо authProbe) исправлена; продуктовая ошибка этим не маскируется.
Новые process/E2E не создавались: следующий этап после независимого approve.
Live VPS и настоящий Codex после исправления — NOT-RUN автором.

## Журнал

| Время Europe/Moscow | Событие | Статус | Основание |
| --- | --- | --- | --- |
| 2026-10-18T18:07:22+03:00 | started | в работе | Первая зафиксированная запись native исправления; отдельная рабочая копия |
| 2026-10-18T18:10:22+03:00 | progress | в работе | Код, targeted integration и Python PASS; готовится полная проверка |

| 2026-10-18T18:12:49+03:00 | completed / handed_off | выполнено | Source freeze, make check PASS, manifest; адресат root/reviewer |

## Версия передачи

Без commit/push. 15 файлов source/tests/instructions (WORK исключена) в manifest:
`/Users/shagi/Desktop/Project/agent-board/.local/worker-source-003/.local/evidence/proxy-host-003/manifest.sha256`.
SHA256 manifest: `78684c73e660c511142a6b644b9ba1adc2fe868554391835eba1c8f0b8413cc5`.
Полный make check log SHA256: `4eb4bf673d94634f192bcdb74880aa38f78f8f3b603227a5611bb4964385a03c`.
Финальный Python24 log SHA256: `26f0c3a0178096d9312e6cc1319358b73a93e05cdccafd2644d4e1ffd5b64226`.

Следующие этапы: независимый code/security review; затем отдельный tester
создаёт process/E2E, тесты проходят review; root фиксирует и развёртывает
проверенную native ревизию, повторяет настоящий HTTPS MCP/Codex smoke.
Существующие source gate/lease/idempotency и схема1 не менялись.

