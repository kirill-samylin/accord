# WORK180 — независимая проверка доступа третьего участника

23.10.2026. Отдельный QA; root ведёт WORK178, reviewer — WORK179.
Область — только эта карточка и ignored private QA evidence под
`.local/test-internet-connection/`. Новый permanent test harness не создаётся.
Прежний actual native GUI preparation FAIL/cleanup из WORK172 сохранён;
эта проверка не повторяет и не закрывает native GUI сценарий.

## Gate и процедура

До отдельного root **setup GO** никаких live запросов новым credential.
План использует existing reviewed `.local/gui-connection/client.py`, обычный
HTTPS login с CSRF и отдельные in-memory CookieJar, без shared cookies.

1. Проверить private token/connection files принадлежат пользователю, regular,
   не symlink, mode0600; не читать/печатать literal token вне Client.login.
   Connection project ID/name и identity — expected178 scope, не OS identity.
2. Новый participant normal login → read-only catalog: ровно один active проект
   `Test_internet`, роль editor; association с exact credential-free
   `https://github.com/DenisShagi/Test_internet.git`; ветка не выдумана для
   repository без подтверждённого default branch.
3. Read-only запросы catalog detail и explicit project route для GUI/default
   по known owner metadata:403/404 denial, без switch/project mutations.
4. Отдельный owner login/catalog: прежний GUI и новый Test_internet видны,
   roles совпадают с expected owner baseline/WORK178 proof. Никаких membership
   изменений или нового credential этим QA не создаётся.
5. Normal logout обоих sessions. Private report содержит только безопасные
   результаты/counts/roles и hashes; origin/token/CSRF/cookies не выводятся.

No model/native CLI actions, host enrollment/controller grant, task creation,
изменения GUI/Test_internet source или общих настроек. Editor не означает
управление чужими процессами. OS/Claude Code/Codex третьего участника и его
физический компьютер остаются NOT-RUN.

Процедура передана179 до execution; source setup178 проходит отдельный review.
Actual QA status: **NOT-RUN**, ожидается setup GO.

## Фактический результат

Reviewer179 одобрил процедуру и узкое normal project-open действие только для
создания browser binding; root передал setup GO после создания проекта и
индивидуального credential. Выполнен один проход через existing Client, без
нового permanent harness, отдельных моделей или worker.

**PASS5**, оба normal logout успешны. Полный private report:
`.local/test-internet-connection/qa180-1792759319126156000/results-private.json`.

- Новый credential соответствует новому индивидуальному identity/machine;
  token/connection files regular, принадлежат пользователю, защищённый mode.
- Catalog содержит ровно один active `Test_internet` с ролью editor, can_write
  true/can_manage false. Repository association exact credential-free
  `https://github.com/DenisShagi/Test_internet.git`, branch пустая/не выдумана.
- GUI catalog detail и explicit GUI managed route, а также default managed
  route вернули строго404 до и после открытия разрешённого проекта. Ни401,
  ни409PROJECT_SESSION_REQUIRED, redirect или network error за denial не приняты.
- Normal CSRF/action open разрешённого проекта работает; его managed read
  показывает0 hosts/0 runs. Editor не получил OS control/enrollment.
- Прежний owner видит GUI с прежней owner ролью и новый Test_internet как owner;
  его catalog snapshot до/после проверки неизменен. Membership и business state
  тест не менял. Оба browser sessions завершены normal CSRF logout, cookie
  files не создавались, credential contents/CSRF/cookies/origin не выводились.

Evidence передано179 для независимого review. ОС/native Claude Code/Codex и
физический третий компьютер **NOT-RUN**. Прежний WORK172 native GUI preparation
**FAIL** остаётся отдельным незакрытым результатом и не заменён этим PASS.
