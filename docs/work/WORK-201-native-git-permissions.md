# WORK201 — локальное разрешение Git для автоматического разработчика

23.10.2026, root. CONTRACT DRAFT, не разрешение активировать новый режим.
Продолжение TASK014; исходный source candidate3e6ac81 прошёл SOURCE APPROVE189.
Новая реализация требует отдельного contract/source review и QA.

## Основание

WORK198v2: настоящий Codex изменил fixture-файл и завершил turn с положительным
owned exit, но не создал commit. `git add/commit` не вызывались; сообщение агента
об отказе не доказывает OS-denial. Полный workflow поэтому не прошёл prerequisite.
Отдельно обнаружен hash drift пользовательского конфига; его исходные bytes не
сохранены, автор изменений не установлен, конфиг не восстанавливался.

Readonly проверка текущего user config показала отсутствие `sandbox_mode`,
`sandbox_workspace_write`/`default_permissions` и named profiles. Есть trusted
entry текущего synthetic checkout. Это текущий snapshot, не реконструкция drift.

## Выбранный контур

Default-off local `AutomationWorkspace` opt-in **codex_git_metadata_writes**.
Применяется только к automation developer с Codex и отдельно разрешённым
workspace-write. Manual runs, analyst/reviewer/tester и прежние конфиги сохраняют
прежнее поведение. Server policy/profile/command не включают opt-in. Флаг входит
в local preset consent digest; его изменение требует обычного local consent
update и явного обновления server policy, без автоматического regrant.

Для этого режима не передаётся legacy `sandbox_mode`. Передаются временные
CLI config overrides выбранного named permission profile, который расширяет
`:workspace` и добавляет write только для **точного canonical checkout/.git**.
Его `.git/config` и `.git/hooks` остаются read-only, как и унаследованные
`.agents/.codex/.aws`. Command network выключен, approvals never; managed
requirements не ослабляются. Поддерживается обычная `.git` directory внутри
checkout с проверенной identity, не symlink/gitdir outside или linked worktree.

Это разрешение писать Git-метаданные в данном checkout, а не whitelist только
команды commit. Push/merge/deploy не разрешаются новым контрактом. Полный HEAD,
source manifest, clean index и независимые review/QA gates остаются прежними.
Same-OS-user/native configuration ownership — прежняя trusted-owner граница.
Наследуемый `:workspace` сохраняет прежние системные temporary write paths;
новый grant — только exact `.git`. Нельзя обещать общий запрет всех записей
вне checkout, уже разрешённых baseline profile.

Ни `--approve-for-me`, ни отключение sandbox, ни `--ignore-user-config`, ни
правка global config, ни fallback на более широкие permissions не применяются.
Профиль не должен молча игнорироваться из-за старых настроек Codex.

## Проверка применяемой конфигурации

Используется supported local stdio `codex app-server`: только initialize,
initialized, **config/read{cwd,includeLayers:true}** и **configRequirements/read**.
Без thread/start, model/list, turns, auth login, config write, tool/MCP calls.
Local config proof не является выполнением работы. Прочие MCP connectors
отключаются на inspection invocation; существующий exact managed MCP inspection
перед native запуском сохраняется отдельно.

Текущий process API передаёт только one-shot prompt и закрывает child stdin;
он не подходит для интерактивного handshake. Допускается узкий фиксированный
config-only dialogue внутри supervised helper с отдельным child stdin pipe:
initialize → ожидаемый ответ → initialized → config/read и requirements/read →
оба ожидаемых ответа → закрытие stdin и positive owned exit. Parent lifeline
остаётся отдельным. Произвольные RPC/команды или клиентский pass-through не
добавляются. Неожиданные server requests, ошибки, malformed/oversized output,
неизвестные ID и timeout дают fail-closed с остановкой только owned child и
достоверным proof; raw responses не выводятся в логи. Batch/посылка всех строк
с немедленным EOF не считаются доказанным протоколом.

Документированный API подтверждён схемой установленного CLI
0.163.0-alpha.11.1: ConfigReadParams.cwd/includeLayers, ConfigReadResponse.layers
с config/name/version/disabledReason, requirements.allowedPermissionProfiles и
defaultPermissions доступны. Readonly generation схемы выполнена root в ignored
`.local/task014/codex-schema201`; config RPC/model в этом шаге не выполнялись.

Автор сверяет фактическую форму response до выбора окончательного parser:

- все active config layers и effective selected profile однозначно прочитаны;
  active legacy sandbox keys, unknown/unreadable shape или противоречащие
  managed requirements блокируют режим с понятной причиной;
- exact selected profile не содержит дополнительных paths/roots/parent/network
  grants, effective approval policy never; managed allowlists не обходятся;
- source/config/permission inspection вне global worker mutex, bounded, с
  positive owned exit. Ownership uncertainty сохраняется как uncertainty;
- текущая authority, path/executable identity и согласованный proof проверяются
  перед spawn, stop/revoke при подготовке сохраняют прежние fencing/barriers;
- сохраняется только безопасный digest/описание permission proof, не raw layers:
  config response может содержать secrets, никогда не идёт в server/log/output.
  Не относящиеся к permissions изменения вроде project trust metadata не
  объявляются ни расширением прав, ни доказанным авторством изменения конфига.

Если actual CLI/API не позволяет доказать этот режим, он остаётся blocked.
Нельзя компенсировать это более широким preset или текстовым обещанием модели.
Схема Config допускает additionalProperties, но не типизирует permission tree:
его фактическое присутствие и exact grants должны быть доказаны config-only QA.
Configured grants proof не заменяет отдельную отрицательную проверку OS enforcement.

## Порядок и приёмка

1. Independent contract review201 до product реализации; author193 — worker/IPC
   и авторские unit/integration. Root — contract/checkpoint/integration docs.
2. Unit проверяют opt-in/default/manual/role separation, digest/version binding,
   exact profile/path generation, config layers/managed constraints/malformed
   responses/secrets, stop/ownership failures и отсутствие fallback.
3. Независимое source review до новых process sources; отдельный QA после
   approve пишет config-only и actual native positive/negative harnesses.
4. Сначала config-only proof установленного CLI, затем один actual Codex
   file+single local commit в fresh private HOME fixture. Проверяются exact
   bytes/tree/parent/HEAD/clean state/real UUID/positive owned exit.
5. Отдельная отрицательная проверка доказывает запрет записи в sentinel вне
   checkout и baseline temporary paths, а также в
   read-only config/hooks. Не считать отказ в тексте агента OS enforcement proof.
   Модель не получает задачу атаковать пользовательские файлы: только собственные
   synthetic sentinel paths внутри private fixture parent.
6. После этого обязательны real full pipeline/fix/control/UI и capable Linux
   migration/restore, затем отдельная production activation. Windows native
   остаётся NOT-RUN до настоящего Windows ПК.

Официальные источники: [permission profiles](https://learn.chatgpt.com/docs/permissions),
[advanced config](https://learn.chatgpt.com/docs/config-file/config-advanced),
[app-server config/read](https://learn.chatgpt.com/docs/app-server),
[noninteractive CLI](https://learn.chatgpt.com/docs/non-interactive-mode).
