# WORK260 — одна запись доверия Test_internet, SOURCE ONLY

`ops_validation_qa`, 24.10.2026. По явному пользовательскому разрешению root
поручил подготовить применение доверия **только**
`/Users/shagi/Developer/Test_internet`. Actual config пока **не изменён**;
Codex/Claude/helper/native/model запусков не было.

Применён OpenAI Docs skill. Поддерживаемый ключ
`projects.<path>.trust_level = "trusted"` проверен по
[официальному configuration reference](https://learn.chatgpt.com/docs/config-file/config-reference).
Current own config прочитан приватно только для parser/metadata/hash binding;
содержимое и секреты не выводились, token/auth/profile файлы не читались.
Exact project entry отсутствует; mode600/owner/single-link подтверждены.

Подготовлен private пакет `.local/task014/trust260/`: helper, pure transform
fixtures, план с точным snapshot и root GO, README и source-preparation receipt.
Преобразование добавляет только одну TOML table/строку и сохраняет все исходные
байты как prefix; parsed остальная структура должна совпасть. Exact trusted —
verified noop, иной/неоднозначный existing entry, duplicate/invalid TOML или
неподдерживаемая sealed inline table — stop. Other project trust/approval/model/
MCP/profile/auth настройки не меняются.

Будущий apply требует source approval и exact root GO. Проверяет собственные
NOFOLLOW/single-link fd, mode/uid/gid/inode/bytes, nonblocking advisory lock и
отсутствие конкурирующего writer; создаёт private600 original backup с fsync file
и directories, затем private same-directory temp/atomic replace/fsync/readback.
Повторная проверка исходного inode+bytes выполняется непосредственно перед
replace. Новый inode/mtime ожидаемы; original config не chmod. Не заявляется
универсальный CAS против noncooperating concurrent editor — root обеспечивает
quiescence. Unknown/ошибка сохраняют private evidence, без auto retry/rollback/delete.

По уточнению root разрешено не сохранять только информационный
`com.apple.provenance`: значение не читалось/не копируется. Readonly Darwin
metadata guard отклоняет ACL/flags/иной или security xattr до replace; uid/gid/
mode600 сохраняются. Root подтвердил noACL и sole provenance; собственный
fd metadata probe подтвердил ACL NULL/ENOENT,21bytes xattr-name list и flags0.
Никакого изменения TCC/системных прав нет.

Helper171строка (дополнительные строки — metadata и durable failure/backup guards,
без нового framework). JSON/Python AST/modes700/600 и unchanged config bytes —
PASS static. Pure fixtures8append/3noop/11reject автором **не выполнялись**;
переданы independent202 вместе с полным source review. Source apply NOT-RUN.

Manifest5members SHA256
`ff3ed0fc9caaa28672541375a867099dd91802f72a6fc19e0e8291da4be246c9`.
Plan SHA256 `040aaf4af0a41effa290c88e8c3eb9236012f86d35f2905582eb9e3b32648e63`.
Helper SHA256 `001dc075339771fa64aab2002a0eaf5c9e1b7b1deb86f2975c7a1b3d6dd79fc5`.
Pure fixtures SHA256 `4bd98d694cfebda531e8b0e0bb7b414510e1d461c6e1504e869727c3c7d3588f`.
Global config snapshot и будущая original backup остаются только private ignored
evidence; содержимое не переносится в WORK/Git.

Independent202: полный SOURCE **APPROVE** exact ff3ed0fc…/5members, findings нет.
Pinned Python AST-only synthetic transform8append/3noop/11reject — PASS;
helper apply/current config/native reviewer не выполнял. Approval receipt
`.local/task014/root-go/trust260-source202.json`, SHA256
`be8fd7320c231608e2828cf98b2291d4f86be520a8252c1c382c623c5a3218ca`.
Source не менялся; actual apply требует отдельного root GO.
