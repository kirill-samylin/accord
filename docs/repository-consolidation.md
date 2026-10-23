# Объединение истории Accord — 23.10.2026

По прямому выбору пользователя итоговая `main` содержит текущий код и полную историю всех локальных и прежних удалённых веток. Экспериментальные функции не возвращаются в рабочую версию. Это объединение истории, а не заявление о функциональной интеграции старых черновиков.

Основой служит `273ccbb15c99dfac91c37f34d2926abf01ca866f` и текущий документированный checkpoint. Независимые проверки — WORK174/175; публикация — WORK173. MIT LICENSE и исходная история сохранены. Новый основной репозиторий: [DenisShagi/accord](https://github.com/DenisShagi/accord).

Для сохранения ancestry используется явное слияние истории со стратегией `ours`: product tree `cmd/`, `internal/`, `infra/`, `web/`, `go.mod`, `go.sum` не меняется. У каждой ветки ниже её исходный commit остаётся достижим из `main`; старый код можно изучить по SHA. Это не squash и не подмена старой истории новым initial commit.

## Что осталось историей

- Config D: настраиваемые роли/scopes/workflow, отличающаяся schema/config semantics. Не включено поверх действующей schema5.
- Performance D4: COW snapshots, индексы и годовые projections/metrics. Не включено; прежние неполные gates не объявлены PASS.
- Legacy backup policy/method: отдельные retention/reserve corrections старой поставки; текущий capable all-project backup не заменяется.
- Integration E: GitHub trust/interchange/notifications primitives и конфигурация; это не реализованный production runtime/provider.
- Native proxy guards перенесены/заменены текущими проверками host/auth/routing; исходная ветка сохранена для сравнения.
- Первоначальное ТЗ уже сохранено побайтно в `docs/source/`; второй устаревший корневой файл не добавляется.

Минимальные дополнительные вершины, покрывающие все расхождения: `f94c95d`, `c8ce77e`, `d1d3fb8`, `dc9c74c`, `25341ba`. Предки остальных веток входят через них и текущий checkpoint.

## Исходные вершины веток

Префикс `origin/` в таблице обозначает прежний upstream до переноса. Достижимость истории не равна включению экспериментальной функции.

| Ветка | Исходный commit | Состояние до объединения |
| --- | --- | --- |
| `codex/agent-profiles` | `343b93c71297c58c972f73fed4d1d4ca3a349740` | Уже предок текущего кода |
| `codex/backup-budget` | `d4337182b1a1353305b06ab9ffcec591433836ba` | История сохранена; dispositions выше |
| `codex/backup-method-fixes` | `c8ce77e039c841e0cb0d67b8924950154c3ea495` | История сохранена; dispositions выше |
| `codex/backup-qa` | `7200cc9e1fdfc544dd274ccb8f3991c849112885` | История сохранена; dispositions выше |
| `codex/config-fixes` | `d5a3a12bcf5a87d854171288139788bff0e2ab9a` | История сохранена; dispositions выше |
| `codex/config-qa` | `71d0ba76f5c3b50a3e5a259ca6cd6c536bfe7b67` | История сохранена; dispositions выше |
| `codex/config-runtime` | `932aa2a3017da0b4fc06032e5134d4ed8fdf2d5b` | История сохранена; dispositions выше |
| `codex/control-refinement` | `55071c83e3e7e5d547c5db245263426312d04bf9` | Уже предок текущего кода |
| `codex/coordination` | `cebcbe5b1955ffc95fb74aabc90c2233d6c83ae3` | Уже предок текущего кода |
| `codex/dedup` | `d9f9eba6fdecb131e36f49907d421a2df43cccbd` | Уже предок текущего кода |
| `codex/integration-config` | `d1d3fb87d4c6d55895a9b0701dd6ae1112121e75` | История сохранена; dispositions выше |
| `codex/integration-libs` | `59ae1a1a8e240e135af155263d01cfd648608c9d` | История сохранена; dispositions выше |
| `codex/managed-agents` | `273ccbb15c99dfac91c37f34d2926abf01ca866f` | Уже предок текущего кода |
| `codex/managed-launch` | `cebcbe5b1955ffc95fb74aabc90c2233d6c83ae3` | Уже предок текущего кода |
| `codex/multi-project` | `867f905d301a508df61f018f6ecda1d17f81539f` | Уже предок текущего кода |
| `codex/native-proxy` | `dc9c74cd692e287173def9fb5743dbf18ffdef21` | История сохранена; dispositions выше |
| `codex/next-vps` | `64e534b7684d238d3e6e243e366844f406b2e61c` | Уже предок текущего кода |
| `codex/next-web` | `7d046d5167e884abfde1ca4a793dcea623d0bd6c` | Уже предок текущего кода |
| `codex/performance` | `b6076eef6bad219bb2137e9e4ad1fd612ada5e55` | История сохранена; dispositions выше |
| `codex/performance-fixes` | `73554ca8d1fac543a2282dcd14a923eb29976aec` | История сохранена; dispositions выше |
| `codex/performance-qa` | `f94c95d19ab7516511e32533f99b1c6430c705b3` | История сохранена; dispositions выше |
| `codex/pilot` | `d542a09648d183034e8cedb6e0ddd05336082fd6` | Уже предок текущего кода |
| `codex/web-control` | `d542a09648d183034e8cedb6e0ddd05336082fd6` | Уже предок текущего кода |
| `codex/web-service` | `a6501d8621abba558606b7fc26d7407fb3b2065e` | Уже предок текущего кода |
| `origin/codex/dedup` | `d9f9eba6fdecb131e36f49907d421a2df43cccbd` | Уже предок текущего кода |
| `origin/codex/native-proxy` | `dc9c74cd692e287173def9fb5743dbf18ffdef21` | История сохранена; dispositions выше |
| `origin/codex/pilot` | `d542a09648d183034e8cedb6e0ddd05336082fd6` | Уже предок текущего кода |
| `origin/codex/web-control` | `d542a09648d183034e8cedb6e0ddd05336082fd6` | Уже предок текущего кода |
| `origin/codex/web-service` | `a6501d8621abba558606b7fc26d7407fb3b2065e` | Уже предок текущего кода |
| `origin/docs/agent-board-spec` | `25341ba5d3890a0273f80ba9cfb66526c1097e3b` | История сохранена; dispositions выше |
| `origin/main` | `9537cb737ee3d1f1e3d63eeccc6a81e2fa681698` | Уже предок текущего кода |

## Границы публикации

Публикуется одна ветка `main`, содержащая всю указанную историю. Существующие локальные ветки/worktrees остаются на месте. Незакоммиченные посторонние файлы не теряются и не публикуются автоматически.

Не публикуются `.local/`, токены, cookies, БД, private runtime/config, Git bundles и внутренние `refs/codex/*` снимки приложения. Старый локальный `deploy-task012.command` не входит в дерево: он ссылается на прежнюю поставку. Не использовать `push --mirror` или старые deployment helpers.

В старых коммитах сохранены ранее опубликованные ссылки на рабочий HTTPS стенд; это публичные ссылки, не credentials. В текущей документации они заменены общими ссылками на процедуру подключения. Проверка известных private credentials и ключей выполняется отдельно до push, без публикации значений.

Перенос репозитория не переключает сервер и не исправляет ошибку native GUI preparation. Production остаётся на проверенном release273/schema5, pending native smoke/finalize; актуальный статус — TASK013.
