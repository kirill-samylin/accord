# Подготовить локальный компьютер для автоматического цикла

Инструкция для TASK014 и команды `worker automation-preview` из WORK205.
Точная установленная версия и runtime-проверки — в [checkpoint](TASK-014-automation.md).
До обновления клиента эта команда отсутствует. Windows native требует отдельной
проверки; cross-build не означает, что инструкция уже проверена на Windows.

Сначала подготовьте обычный [worker config](managed-worker-guide.md): свой
project ID, собственный токен, реальную папку проекта, native CLI и управляющих.
Один проект использует свой state. Для изменения согласия работающий worker
сначала штатно остановите; второй процесс с тем же state не запускайте.

## Выбрать реальные проверки проекта

Владелец задаёт фиксированные локальные проверки. Агент выбирает только их ID;
передать произвольную команду из браузера нельзя. Обязательна хотя бы одна
проверка типа test с настоящим JUnit/TAP отчётом и исполненными тестами.

Пример ниже предназначен для проекта, где уже есть `tools/run_checks.py`,
который действительно запускает тесты и сохраняет JUnit в `.artifacts/junit.xml`.
Сам Accord этот файл не создаёт. Выберите команду и формат своего проекта.
Пустой репозиторий требует сначала определить требование, стек и способ проверки.
Нельзя заменять тесты командой, которая просто печатает «PASS».

Добавьте к своему private config следующие поля, заменив alias `repo`, пути,
UTC deadline и пределы на согласованные значения:

```json
{
  "codex_sandbox": "workspace-write",
  "native_mcp_approved_tools": {
    "codex": ["describe", "next", "get", "transition", "workflow_question", "workflow_plan", "workflow_run_check"]
  },
  "automation": {
    "version": 1,
    "workspaces": {
      "repo": {
        "version": 1,
        "enabled": true,
        "expires_at": "REPLACE_WITH_UTC_DEADLINE",
        "roles": {
          "analyst": ["codex"],
          "developer": ["codex"],
          "reviewer": ["codex"],
          "tester": ["codex"]
        },
        "codex_git_metadata_writes": true,
        "allow_tester_artifact_writes": false,
        "max_attempt_seconds": 300,
        "max_workflow_seconds": 1800,
        "max_native_invocations": 12,
        "git_executable": "/ABSOLUTE/CANONICAL/PATH/TO/git",
        "git_sha256": "",
        "source_roots": ["src", "tests", "tools", "README.md", "AGENTS.md", ".gitignore"],
        "artifact_roots": [".artifacts"],
        "dependency_roots": [],
        "checks": {
          "project-tests": {
            "version": 1,
            "enabled": true,
            "kind": "test",
            "required": true,
            "executable": "/ABSOLUTE/CANONICAL/PATH/TO/python3",
            "executable_sha256": "",
            "args": ["-B", "tools/run_checks.py"],
            "directory": ".",
            "timeout_seconds": 120,
            "report_format": "junit",
            "report_path": ".artifacts/junit.xml",
            "isolation": "trusted_owner_process",
            "trusted_owner_consent": ""
          }
        }
      }
    }
  }
}
```

Это фрагмент для объединения с базовым config, а не замена всего файла.
Верхний `workspaces.repo` должен уже содержать реальную папку и provider codex.
Укажите все tracked source paths; исключённые artifact/dependency roots не
могут пересекаться с source roots. Поэтому `source_roots:["."]` вместе с
`.artifacts` не подходит. Dependency roots требуют собственных hashes.
`trusted_owner_process` означает явный запуск проверяемого кода репозитория
от владельца worker; это не системная изоляция произвольного тестового кода.

Git flag отдельно разрешает Codex developer писать метаданные **этой** `.git`;
config/hooks остаются read-only. Нужна обычная `.git` directory в checkout,
без linked worktree/внешнего gitdir. CLI должен поддерживать проверяемый режим
permissions; при конфликтующих legacy/managed настройках запуск блокируется.
Flag не меняет global config и не разрешает push/merge/deploy. Аналитик,
ревьюер, тестировщик и ручные запуски не получают этот дополнительный grant.

`allow_tester_artifact_writes:false` сохраняет обычный native read-only режим
тестировщика; согласованный фиксированный check запускает worker. Если нужны
artifact-only предложения новых тестов, владелец отдельно выбирает true.
Даже тогда изменение reviewed source или HEAD блокирует результат.

Для Claude выберите его native executable/локальный режим в базовом config,
provider `claude` для нужных ролей и соответствующий точный MCP allowlist.
Codex Git flag относится только к Codex developer. Не переносите на Claude
обещание проверенных Mac/Codex permissions; native Claude проверяется отдельно.
Если разработчик использует только Claude, установите
`codex_git_metadata_writes:false`: true требует локальной роли Codex developer.

## Получить и проверить предложение

```text
agent-board worker automation-preview --config PRIVATE_WORKER_JSON
```

На Windows используйте путь к `agent-board.exe`, canonical native `.exe` для
Git/checks и Windows пути в JSON с `\\` (например, `C:\\Projects\\repo`). Private config/state должны иметь
нужные [DACL](managed-worker-guide.md); Unix chmod их не заменяет.

Ожидается JSON с `applied:false`, `requires_local_apply:true` и значениями для
каждого workspace/check. Команда читает draft и файлы для hashes, проверяет
полную допустимость предложения. Она не читает токен, не создаёт state,
не подключается к серверу и ничего не включает. Existing nonempty pin mismatch
даёт отказ: новая версия executable/dependencies требует отдельного решения
владельца, старый pin не подменяется молча.

После проверки перенесите предложенные `git_sha256`, dependency hashes,
`executable_sha256` и `trusted_owner_consent` в соответствующие поля своего файла.
При изменении ранее принятого consent увеличьте local automation/workspace/check
версии по изменённым настройкам, затем заново получите preview. Digest связан
с путём/identity рабочей папки, roots, executable/args/report/check параметрами
и локальными Git/tester flags; его нельзя переносить в другой checkout.

Если поля остаются неправильными или deadline истёк, исправьте draft локально.
Не включайте полный доступ, не отключайте проверки и не меняйте общую политику
CLI ради прохождения preview или native запуска.

## Опубликовать локальное согласие и запустить worker

Для уже подключённого компьютера:

```text
agent-board worker consent --config PRIVATE_WORKER_JSON
agent-board worker run --config PRIVATE_WORKER_JSON
```

Если компьютер новый, сначала выполните обычное enrollment из базовой инструкции.
Preview не заменяет enrollment или consent. В приложении заново выберите host
и workspace, примите актуальное локальное согласие и создайте/обновите policy
через **«Выполнить требование» → «Настроить запуск»**. Старое разрешение с прежними
digests не обновляется автоматически. Команда, роли, controllers, проверки,
сроки и лимиты policy должны укладываться в локально разрешённые.

Если обновился Codex/Claude (новый исполняемый файл), шаг не стартует с кодом
`provider_pin_changed`; повторите `worker consent`. Локальный бюджет считает только
реально запущенные native процессы. Восстановление после сбоя — `worker recover`,
см. [базовую инструкцию](managed-worker-guide.md#сбой-и-восстановление).

Далее — [отправить требование и управлять циклом](automation-guide.md).
Второй человек использует собственный токен и своё локальное согласие; файлы
токенов, state, host proof и вход провайдера между людьми не передают.
