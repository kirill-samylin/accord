# WORK-092 — независимое UI review Accord

Роль: design-reviewer `/root/webdesign092`, отдельный от авторов087/088/089/root.
Статус: **APPROVE** осмотренного функционального UI после исправлений UI092-01/02.
Открытых блокирующих замечаний в осмотренном объёме нет.
Дата: 19.10.2026. Родитель TASK-004; база `f06338089bec3b8c7f17f541c463131f5c98f96c`,
рабочая версия `agent-board/.local/web-service-084`, незакоммиченный diff.

## Объём и основания

Прочитаны самостоятельный Accord AGENTS.md, профиль design-reviewer и общий
процесс/design-first Давинчи только как инструкции роли, WEB-01..07/WEB-AC,
web-guide, web-architecture и фактический Page contract. Accord самостоятельный:
Давинчи/UI-kit/Tandem/Git/deploy не затрагивались. Запись только этого WORK;
временные воспроизведения — private `/private/tmp/accord-web-084/ui-review/`.
Именованные Claude-модели недоступны; модель унаследована, автор и reviewer разные.

Пользователь разрешил функциональный адаптивный SSR без отсутствующего макета
Claude Design. DS/SCR и pixel matching **N/A**; новый стиль/новые требования
не назначались. Проверены реально отрисованные страницы и выборочные сценарии,
а не только templates. Это независимая оценка состояний/удобства, не замена
последующего полноценного browser E2E тестировщика.

Окружение: macOS, установленный Google Chrome через Playwright, headless,
viewport1440×1000 и390×844, deviceScaleFactor1, масштаб100%, русские UI тексты.
Локальный синтетический fixture `http://127.0.0.1:17894`, личный fixture-token
входом через file picker; токен не выводился и не снимался на screenshot.
Реальный HTTPS/VPS не проверялся. Ранний binary отличался nonvisual files delta;
после UI092-01 координатор пересобрал exact текущие исходники, повтор входа PASS.

## Находки и возвраты

| ID | Требование / экран | Наблюдение и влияние | Адресат / повтор |
| --- | --- | --- | --- |
| UI092-01 P1, закрыто | WEB-02, login HTML POST | `Referrer-Policy: no-referrer` приводит в Chrome к `Origin:null`; strict guard отклоняет настоящий вход403. Screenshot01. Все разделы недоступны после входа | Root/backend заменил policy на same-origin без ослабления guard. Повтор настоящий file login → Origin=http://127.0.0.1:17894, redirect303 /web/, четыре раздела доступны: PASS |
| UI092-02 P2, закрыто | WEB-01/03/07, собственный answer editor, ответ из пробелов | Native `EVIDENCE_MISSING` возвращает английское `answer required`/409; JS трактует это как потерю владения и отключает отправку, хотя попытка жива. Пользователь не может исправить ответ без reload; ошибки у поля нет | Root добавил trim/limit validation400 до bind/native, русскую ошибку у поля и обновление live FieldErrors/action ID. Повтор:400, aria-invalid=true, draft сохранён, submit доступен, ID rotated=true; исправленный ответ без reload →done, один POST: PASS. Реальный stale409 остаётся readonly |

Сценарий UI092-02: взять human-вопрос T-2 в fixture → ввести два пробела →
«Ответить и завершить». Наблюдались status `Действие не выполнено answer required …
Обновить состояние вопроса`, disabled submit=true, draft сохранился. Скрипт
`journey.cjs` остановился на disabled-кнопке; остальные состояния продолжены
новым browser context той же сессии через явный reload, без скрытого reacquire.

Неблокирующий текстовый долг: system reason `completed` отображается буквально
в истории попыток. Локализация известного системного значения улучшит WEB-01;
пользовательские свободные причины переводить/переписывать не требуется.

## Матрица фактических состояний

| Экран / состояние | Результат и граница |
| --- | --- |
| Login/keyboard | PASS file login после UI092-01; first Tab показывает skip link с outline3px. Password token не фотографировался |
| Questions empty/filter | PASS отдельные empty «Мои вопросы»/«Вопросов…нет», сброс, текущие/завершённые состояния; чтение не изображает claim |
| Tasks empty/create | PASS созданы настоящий fixture epic E-1 и human T-2/T-3, redirect на постоянный адрес; роль/контекст/автор/эпик реальные; агентский запуск не обещается |
| Questions own/foreign | PASS свой T-3 виден в «Мои вопросы». Другая browser session видит чужую live аренду, нет answer/claim кнопок; это две сессии на одном ПК |
| Own editor/heartbeat | PASS expiry и remaining видны; продление выключено исходно, явное включение даёт успешный heartbeat/status, выключение останавливает |
| Answer validation | PASS повтор UI092-02 после исправления; screenshot02 показывает русскую ошибку рядом с полем, исправленный повтор без reload завершён |
| Answer stale conflict | PASS отправка заведомо устаревшей generation999 даёт409, текст остаётся копируемым, submit disabled, явный переход обновления; новое владение не появляется |
| Answer offline | PASS browser offline предотвращает отправку, статус «Действие не отправлено», текст сохраняется, submit доступен после сети; сообщение на screenshot02 |
| Answer normal/history | PASS после явного reload нормальный ответ завершает T-2; история показывает вопрос/ответ/fixture-human/browser-fixture/E-1/серверную дату/попытку1/ссылку T-2, screenshot03 |
| Design empty/invalid | PASS пустой список; invalid txt отклонён «Неподдерживаемый формат», title сохранён, требуется явный перевыбор файла |
| Design versions/download | PASS HTML v1→явно «Добавить версию»→v2; обе карточки доступны, исходное имя/тип/размер/автор/дата/связь реальные. Attachment200, application/octet-stream; img/frame/object для HTML нет |
| Global error404 | PASS «Не найдено», ссылки к вопросам/входу; реальных путей/секрета нет |
| Labels/focus/navigation | PASS выборочный осмотр: у input/select/textarea есть label/aria-label; Tab проходит skip/brand/logout/4nav/filter, видимый outline3px. Summary/form controls клавиатурно нативные; полный WCAG аудит NOT-RUN |
| Mobile/long text | PASS на390 document.scrollWidth=390 у всех4 разделов/own/done; длинный title≈180символов, непрерывное description525символов и длинный principal переносятся, действия сохранены; nav имеет собственную horizontal scroll область |
| Console | PASS pageerror[] в осмотренных journeys; ожидаемые HTTP4xx не считаются runtime JS crash |
| Loading/duplicate | PASS выборочный задержанный400мс ответ: submit disabled=true, aria-busy=true, статус «Отправляем…», подтверждённый ответ один POST. Полный ambiguous/duplicate replay E2E принадлежит tester |
| Disabled/hover/motion | Disabled stale submit реально проверен, фокус проверен; отдельного motion-макета нет, SSR без заданных переходов — N/A. Полный hover/reduced-motion/assistive tech аудит NOT-RUN |
| Истёкший TTL/403/revoke/cancel/quota/restart | Полноценные runtime варианты NOT-RUN этим UI reviewer; authoritative security/process/E2E покрытие отдельно. Stale generation проверяет conflict UI, но не заменяет реальное ожидание expiry |

## Доказательства и версия

Private evidence, без токенов/реальных данных:

- `01-origin-error.png` — найденный и затем исправленный login403.
- `02-mobile-own-editor.png` — mobile390, свой редактор, явное продление,
  исправленная русская validation возле поля; реально просмотрен reviewer.
  Offline draft проверен отдельно driver states.cjs без дополнительного screenshot.
- `03-desktop-history.png` — desktop1440, реальная fixture запись решения;
  реально просмотрен reviewer.
- `review.cjs`, `journey.cjs`, `continue.cjs`, `design.cjs`, `states.cjs`, `recheck.cjs` —
  временные manual browser drivers, не новые process/E2E tests проекта.
- `context.json` — private runtime browser-session state0600, не отчёт и не Git;
  содержимое не печаталось. Каталог0700.

Снимок SHA256 UI/frontend/backend-boundary сохранён в
`/private/tmp/accord-web-084/ui-review/review-manifest.sha256`.
UI CSS `df4ade6fc2e4fd0f3551ddb5484bf82b9ebda90fe7ea20d5a92ea6bddc208445`;
UI JS до исправления UI092-02 `3f9468faf5adf5dbec4d8478145d366720e28398f67562a7e287af061d4fea13`;
актуальные финальные хеши в manifest.
UI092-01 fixed http.go `8cbbafd28385acac39c87955b38f8ac7ef38eea98f0e582025a516b6a3652368`.
Финальный повтор UI092-02 выполнялся на rebuilt fixture после исправления;
расширение/новое изменение UI требует соответствующего повтора. Отдельные code/security approve,
полный E2E, HTTPS/live/backup/rollback, второй физический ПК здесь не объявлены PASS.

## Итоговая передача

APPROVE ограничен осмотренными функциональными состояниями и адаптивностью.
Оба возврата исправлены авторами и повторно проверены отдельным reviewer.
Дальше — tester на exact проверенной версии для полного WEB-AC, затем code review
его новых тестов; live/VPS выпускает координатор только в порученных границах.
Исходники приложения reviewer не редактировал. Единственный изменённый repo-файл — этот WORK.
