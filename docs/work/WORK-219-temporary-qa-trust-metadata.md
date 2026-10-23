# WORK219 — точечная trust metadata только временного QA checkout

23.10.2026, root. CONTRACT DRAFT; реализация и новый runtime NOT-RUN.

## Основание и граница

Пользователь ответил: **«Разрешить только тестовые папки (рекомендуется)»**
на вопрос о сохранении Codex отметки trusted только для наших временных тестовых
папок, с проверкой остальных байтов настроек. Это будущее ограниченное исключение
для acceptance QA; прежние WORK209B FAIL и WORK218 strict PASS не переоцениваются.
Авторство прежней вставки не доказано. Codex не получает новых OS/MCP/network
grants, writable roots, approval overrides либо разрешения менять настройки.

Разрешение не относится к Давинчи, Test_internet, чужому checkout, произвольному
пути, глобальному доверию либо существующей записи untrusted. Parent/harness
никогда не добавляет, не редактирует, не удаляет и не восстанавливает trust/config.
Допускается только наблюдать результат настоящего consented native QA запуска.

## Единственное допустимое отличие

До любого native child сохраняются private0600 raw config bytes и metadata;
полный baseline фиксируется до первого turn, без промежуточного adoption.
requirements.toml и managed_config.toml сохраняются побайтно и по identity/mode.
config.toml должен быть существующим regular no-link файлом ожидаемого владельца;
его owner/mode не меняются. Возможная atomic replacement identity отдельно
фиксируется в evidence, не скрывается. Ошибки чтения/парсинга — FAIL.

Run заранее фиксирует **один** canonical QA workspace: новый собственный checkout
под private0700 QA HOME prefix, с обычным внутренним .git, без ссылок/пользовательских
файлов/локальных Codex configs. Проверить root ownership, canonical ancestors и
отсутствие неожиданных config файлов до модели. Exact canonical path хранится
только private. Этот workspace не может быть назначен из ответа модели или diff.

Допустимы два результата:

1. Все config bytes идентичны: STRICT_UNCHANGED.
2. В config.toml появилась ровно одна новая запись projects[exactWorkspace],
   которой не было в baseline; её единственное поле trust_level равно строке
   trusted. Все остальные разобранные значения/типы/ключи строго одинаковы.
   Никакой замены старого trust, изменения иных projects или дополнительных полей.
   Raw after получается из raw before **одной непрерывной вставкой ≤512bytes**;
   удаление этой вставки возвращает исходные bytes точно. Вставка содержит только
   таблицу exactWorkspace, trust_level и необходимое ASCII whitespace; без comments,
   других таблиц/ключей/control bytes. Корректность подтверждается TOML parser и
   отдельной exact-shape проверкой вставки. Результат QA_TRUST_ADDITION_ALLOWED.

Если baseline уже содержит exactWorkspace, только вариант1. Любое другое
отличие — FAIL; raw before/after и finite structural diff сохраняются без restore.
Нельзя игнорировать весь projects раздел, нормализовать/пересериализовать baseline
для byte comparison либо объявлять arbitrary include/settings metadata безопасными.
Private evidence хранит file SHA/length/owner/mode/identity, insertion offset/length/
SHA, exact-workspace binding SHA, parsed-diff acceptance и historical config SHA.
Публичный отчёт — только enum/counts/hashes, без путей, raw config или секретов.

## Реализация и review gates

Один небольшой общий QA checker с pure unit cases; потребители используют exact
reviewed SHA, без divergent copies. Node harness может вызвать pinned Python
tomllib helper с bounded input/output/deadline через существующий owned supervisor;
либо checker реализуется целиком в одном runtime с полноценным TOML parser.
Новые зависимости не устанавливать. Успех helper не заменяет ownership/terminal
proof и первоначальный filesystem binding в harness.

Обязательные pure cases: unchanged; exact insertion; insertion at legal table
boundary; existing trusted unchanged; missing workspace plus other key; changed
old trust; extra field/table/comment; second workspace; deletion/reformatting;
malformed TOML/duplicate key; prefix collision; symlink/wrong owner/mode; changed
requirements/managed; multi-insertion; too-large insertion. Cases не читают и
не меняют реальные настройки. Проверяется не только semantic equality.

Сначала независимый CONTRACT APPROVE202. Затем отдельный author SOURCE pass и
SOURCE APPROVE202. Изменения descriptor203/210: exact a32 candidate/build/source
pins + A/C-v2/218 receipts, без новых grants; actual fixture-specific proof всё
равно выполняется самим product worker. Historic8c и closed gates не снимаются
одним редактированием manifest. Каждый runtime требует отдельный root GO после
review, бюджета/cleanup и full source freeze. F6/C2/H1/B2 не получают автоматический
GO из 203. Production apply/enrollment/Windows/второй ПК этим документом не разрешены.

Root owns этот контракт/TASK. Автор общей QA реализации и adapters — отдельный
исполнитель gui_connection_qa; независимый reviewer202. Существующие source/evidence
freezes сохраняются. Общий статус TASK014 остаётся IN PROGRESS/NOT READY.

## CONTRACT APPROVE202 и SOURCE GO

Независимый reviewer202 подтвердил контракт без замечаний: exact predefined
private checkout/new-entry-only/semantic + single-insertion byte checks, прочие
configs strict, прежние результаты immutable. Root выдал author gui_connection_qa
SOURCE GO общего checker и нового203 adapter после narrow C2 correction.
Нужен также genuine browser human session для human-only revoke/disable операций:
унаследованный MCP coordinator session не удовлетворяет контракту этих методов.
Product authorization не меняется. Исторические source freezes сохраняются;
пока разрешены только source/hash/syntax/AST, не helper/native/SSH/UI runtime.
