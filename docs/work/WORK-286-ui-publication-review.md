# WORK286 — independent review нового web пакета

Reviewer: review_changes. Review исходного HEAD `3062a11d84b61d3a1d2541f81028ffd852ba77cf`
и текущих web/QA изменений перед TASK018 publication. Общие TASK, web source,
production и чужие карточки reviewer не изменяет.

Прочитаны AGENTS.md, browser/registry isolation контракт и
`vercel-react-best-practices/SKILL.md`. Проверены текущие diff формы задачи,
auth lifecycle, project create/switch, draft storage, HelpDialog и source
HEAD3062 (CSS/layout). Source review не является browser/production PASS.

## Findings и решения

- **P2 / требуется исправить до approve:** новые task/pending черновики содержат
  несохранённые title/description и привязаны только к project scope; logout и
  unauthorized очищали лишь QueryClient. В одной вкладке пользователь A выходит,
  пользователь B входит и в общем доступном проекте открывает создание задачи:
  форма восстанавливает private draft A. Аналогично глобальное название проекта.
  Исправлено отдельным исполнителем validate_candidate в central api.ts +
  session-drafts.ts. Независимый final re-review PASS: очистка только owned keys
  на successful login/logout, 401 и fallback login; unknown logout сохраняет
  черновик до подтверждения. Пять regression tests повторены reviewer: PASS.
- `deploy-task012.command` не соответствует текущему schema6 release: hardcoded
  codex/dedup/TASK012, прежние private adapters. Не запускался; исключить из новой
  поставки, сохранить отдельно в ignored local storage.
- `qa/first-users/start-local.command` выполняет широкие pkill и kill listeners
  3100/3101 без process ownership. Не запускался; исключить из публичной поставки,
  сохранить original ignored. Исторический REPORT нужно уточнить, не выдавая
  local прогоны старого UI за новые проверки HEAD3062.

## Проверки

- `git diff --check` — PASS на исходной рабочей копии.
- Shell syntax обоих .command — PASS; никаких runtime действий с ними.
- Текстовый scan QA/new sources по private key/token patterns — совпадений с
  секретами не найдено. Все33 PNG просмотрены через6 private contact sheets +3 исходные выборки:
  только synthetic qa-newcomer identities, локальные test projects/tasks и
  замаскированные password поля; raw credentials/реальных пользовательских
  данных не найдено. Все33 PNG без metadata chunks в PIL info. OCR Vision
  завис до результата и остановлен; OCR PASS не заявляется.
- Source review: server authorization, action ID/CSRF, project mismatch guards,
  uncertain exact retry и dedup consent не ослаблены diff. Unit/typecheck/build
  выполняются отдельно; здесь их результат до actual evidence не присваивается.
- Форма CreateTaskDialog выросла выше 300 строк; это изменение существующего
  файла, не новый framework. Причина — сохранение текущих mutation/dedup/profile
  gates совместно с новым epic-first recovery. Файл требует последующего
  отдельного split, который не нужен для узкой публикации.

## Final source acceptance

**APPROVE / PASS** для текущего source package, включая независимый auth fix.
Новых неустранённых source blockers не найдено. Обе исторические .command
исключаются из публикации с сохранением ignored originals; QA REPORT содержит
описание локального стенда и не используется как exact candidate production QA.

Reviewer собственноручно повторил5 новых auth regression tests и diff-check PASS;
WORK288 отдельно сообщает typecheck/109unit/format PASS. Root сообщает Next build
PASS; immutable committed rebuild/release provenance остаются отдельным gate.
Final reviewed source hashes/approval receipt сохранены в ignored
`.local/task018-ui-release/review/`. Exact candidate SHA будет привязан после
commit без нового изменения tracked карточки.
Production/browser/native Windows/Claude/second physical PC reviewer не запускал.
