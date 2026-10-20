# WORK-142 — Next UI постоянных профилей

20.10.2026. Автор: отдельный frontend исполнитель TASK010. Область записи:
`web/**` и этот WORK. Backend/operations/общую TASK не менял; коммиты не создавал.
Исходный checkpoint `867f905d301a508df61f018f6ecda1d17f81539f`, ветка
`codex/agent-profiles`; source изменён совместно с другими владельцами областей.

## Контракт и реализация

Прочитаны AGENTS, agent-profiles и независимый WORK140; сохранены инварианты
requirements/architecture/API/multi-project/Next contracts. Применён
`vercel-react-best-practices` без новых библиотек и обновления зависимостей.
Поля и пути согласованы непосредственно с backend автором WORK141.

- Боковой пункт «Агенты» открывает постоянные профили. Навигация
  «Профили / Сессии» сохраняет прежний экран сессий и его маршруты.
- `/app/profiles` и явные scoped paths показывают реальные backend profiles,
  фильтры действующие/архив/все, добавление и краткие карточки. Инструкции
  в списке не отображаются. Имя не используется как ключ или идентификатор.
- `/app/profiles/{id}` показывает описание, инструкции, сохранённые
  предпочтения provider/model, ID/version/state и метаданные. Изменение,
  архив/восстановление отправляются в Go с CSRF/issued action capability/version.
- История версий открывает неизменяемую ревизию через `?version=N`.
  Историческое представление явно подписано и не предлагает изменения.
- Страница сессии показывает исходный snapshot профиля только при его наличии
  в серверной записи; прежние сессии подписаны «Без профиля». Текущая версия
  профиля не выдаётся за использованную сессией. Snapshot содержит исходные
  instructions/settings и отделён от состояния сессии/аренды.
- Нет кнопки запуска, fake online/model/cost/provider connection. Настройки
  обозначены сохранённым предпочтением; инструкции читаются участниками проекта,
  форма сообщает не добавлять секреты.

## Границы мутаций и черновика

Существующий `useAction` сохраняет захваченные project URL/body/action_id при
неизвестном исходе. Редактор/проект нельзя закрыть/переключить в sending/uncertain;
повтор отправляет те же значения. После окончательного rejection черновик
остаётся доступен, новые capabilities берутся только из подтверждённого ответа.

Черновик фиксирует исходную version отдельно от обновляемой PageData. Фоновое
чтение, конфликт409 и явная проверка состояния не переписывают введённый текст
или version. Если серверная version изменилась, submit отключён до явной кнопки
«Применить черновик к версии…». Эта кнопка меняет только основание версии,
сохраняя текст пользователя. Версия и поля формируются из одного draft snapshot.

Project QueryClient/mount boundaries TASK009 сохранены; query keys включают
полный scoped path и revision query, aborted late reads отвергаются API. Новые
page/mutation routes добавлены строгими allowlists с canonical profile IDs.
Viewer/архив проекта не получают mutation controls; `readPage` дополнительно
сбрасывает `can_manage_profiles`, когда project.can_write=false. Сервер остаётся
единственным владельцем ACL и проверяет прямые POST/replay.

## Файлы

- `web/src/features/profiles/`: screens, dialog, fields, agent navigation,
  session snapshot. Каждый файл до300строк после Prettier.
- `web/src/lib/agent-profiles.ts`: DTO/settings labels, bounded validation,
  draft/version/payload helpers; `profile-contracts.test.ts` —7 regression units.
- `lib/types.ts`, `api.ts`, `project-routes.ts`: новые типы/capability/allowlists.
- `features/board-screens.tsx`, `navigation.ts`, `screens/agents.tsx`:
  подключение экранов, навигация и session profile snapshot.
- `app/styles/profiles.css` и import `globals.css`: текущие tokens/компоненты,
  responsive карточки, формы/история/заголовки и wrapping длинных ID/инструкций.

## Проверки

Node24.19.0, Next16.4.2, существующий lockfile. `make web-check`:
Prettier PASS, TypeScript PASS,60/60 client unit PASS (53 прежних+7 новых),
production build PASS. Исходная проверка обнаружила только TS union return
annotation в новом test-helper; исправлена явным Record<string,string>, повтор PASS.
Известное прежнее Node предупреждение MODULE_TYPELESS_PACKAGE_JSON не стало
причиной менять package.json или зависимости.

Новые units проверяют canonical scoped/revision URLs, разные cache keys,
UTF-8/codepoint bounds, невозможность human/provider prototype injection,
сохранение draft при новой версии, exact retry body/action/version/path,
cross-project success rejection,409 capabilities, viewer suppression и abort
late response. Это units, а не проверка браузера/процессов.

Source review, реальные process/E2E, визуальная390/1440 QA и production HTTPS
этим исполнителем NOT-RUN; отдельный tester допускается только после source
APPROVE. Physical second PC/provider workers/full V1 не заявляются.
