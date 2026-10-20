# WORK-125 — основной Next.js клиент

Дата: 20.10.2026. Автор frontend125. Область записи: `web/**`, кроме
`web/src/features/mutations/**` (автор126), и эта карточка. TASK-007 ведёт
координатор. Прочитаны требования, архитектура, API, control-web-architecture,
WORK-122/123/124, реальные Go DTO и существующие screen/query контракты.
Применён skill vercel-react-best-practices: независимые запросы, ограниченная
сериализация, запросы с отменой, отсутствие глобального auth store/кэша людей.

## Реализовано

- Next.js App Router + TypeScript, React, TanStack Query, Tailwind4, Lucide,
  Radix primitives, cmdk; точные версии в package.json/package-lock.json.
  Go остаётся владельцем auth, доменных мутаций, транзакций и SQLite.
- Streamlined baseline pinned Paperclip `5739a53f5f5e6cccbf21c7ae491ad06cb4b79b7c`:
  sidebar240, один компактный project menu, New Task/Search обычными строками,
  grouped navigation, agent strip первым, плоские KPI, реальные status bars,
  плотные списки/board260, line tabs, task properties, mobile drawer/bottom nav.
  MIT notice и Inter/OFL сохранены в public. Бренд остаётся Accord.
- Десять основных экранов: обзор, задачи, вопросы, агенты, проект, активность,
  статистика, дизайн, решения, настройки. Task/epic detail, agent detail,
  read-only quick-look, project tabs и material version preview. Командный поиск,
  фильтры/пагинация, светлая/тёмная тема, адаптивность и reduced motion.
- Формы126 интегрированы: вход, создание задачи/эпика, редактирование текста,
  human claim/answer/release/manual heartbeat, продолжение/выход, upload.
  Create dialog лениво читает реальные эпики и свежую capability с tasks screen.
  Description forceMount сохраняет answer draft при переключении вкладок;
  unknown upload блокирует закрытие Dialog. Quick-look read-only, рабочие
  действия ведут в полноценную карточку.
- Typed same-origin JSON bridge. Разрешённые переходы `/web` → `/app` проверяются
  allowlist; файлы остаются cookie-protected `/web/designs/:id/...`.
  Персональные reads no-store, QueryClient создаётся на экземпляр приложения,
  очищается на logout/login/401. Неверный login401 не сбрасывает preauth форму.
- Dashboard обновляет только безопасный `/web/data/dashboard` каждые15секунд
  при видимой/online странице, без выдачи новых action IDs и продления аренд.
  401 очищает доступ, 409 останавливает polling и перечитывает shell для явного
  продолжения. Ошибка оставляет последний хороший снимок с фактической датой.
- Epic open не теряется в task enum; unfamiliar server states видимы.
  Vocabulary activity взят из core.ValidControlEventType. Не показаны
  вымышленные запуски процессов, модели, costs, agent config Save или проекты.

## Runtime и защитные границы

Node24.19.0 проверен из bundled runtime. Next поддерживает Node20.9+, а
package engines>=22.9 задан для локального strip-types test workflow.
@types/node24.19.2 соответствует проверенной major runtime. Системный npm20
устанавливает зависимости с expected engine warning; build/test запускаются24.

`ACCORD_GO_ORIGIN` — только server-owned http literal-loopback origin с явным
портом, без credentials/path/query. Единственный fixed rewrite `/web/:path*`.
Rewrites фиксируются при build: одинаковый origin нужен на build/start; default
Go127.0.0.1:3101, Next127.0.0.1:3100. Go --web-origin равен публичному Next origin.
`experimental.proxyTimeout=130000` сохраняет120секунд Go upload deadline;
API подтверждён установленным Next type/source, boundary processQA отдельная.
Next external rewrite также клонирует body; `proxyClientMaxBodySize=51МиБ`
передаёт полный50МиБ файл+64КиБ envelope и boundary+1 до Go без ослабления
Go filecap. Это ограниченный Node buffer, не обещание zero-buffer streaming.
Production ingress `/web` напрямую Go убирает этот дополнительный buffer.
Nonce Proxy match только `/app`, не file uploads. Production script-src nonce
без unsafe-inline/unsafe-eval; dynamic rendering. Styles допускают inline для
Radix geometry и фактических chart widths. SSR `/web` не менялся.

Frontend build находится `web/.local/next`, runtime/evidence root `.local/`.
`web/node_modules` — conventional ignored dependency installation exception.
Все исходные TS/TSX/CSS читаемо форматированы Prettier3.9.10, разделены по
ответственности до300строк. Исключения: генерируемый package-lock.json и
внешние MIT/OFL license texts; ручного minify/line-count gaming нет.

## Авторские проверки

На текущем workspace source, без Git commit:

- `npm run typecheck` под Node24 — PASS.
- `npm test` — PASS27/27, включая8 frontend API/grouping tests и19 forms126.
- `npm run build` — PASS, Next16.4.2 production dynamic routes и Proxy собраны.
- npm public registry install/audit — PASS,0 vulnerabilities на момент установки.
- `npm run format:check` — PASS после финального форматирования.

API tests проверяют fixed upstream, redirect/binary boundary, protected JSON
read options, domain409, exact body identity, unknown/nonJSON retry contract,
сохранность open epic/unknownstate. Новые process/E2E и browser visual checks
автором не выполнялись: отдельный tester после независимого approve.
Официальные Next API проверены координатором; реализации сверены с типами
установленного Next. Final review/QA sourceversion и evidence ведёт TASK-007.

## Оставшиеся границы

Одна реальная configured board. Multi-project registry, persistent agent
profiles/runner/adapters, start/stop/chat, provider accounting/budgets,
org/goals/routines/integrations остаются отдельными backend этапами TASK-007.
Текущая миграция не закрывает весь Paperclip feature parity или полный V1.
Никаких Git commit/push, deploy, VPS/Tandem либо изменений соседнего проекта.

## Обязательная регрессия надёжности чтения после первого ревью

По находкам127 и браузерной QA128 добавлены только следующие исправления:

- `read-deadline.ts`:8секунд на весь GET, включая body/JSON parsing, объединение
  query AbortSignal и timeout, очистка timer/listener в finally. Обычные screen
  GET используют этот же deadline. `submitAction` принимает optional AbortSignal;
  deadline мутаций реализует координатор в отдельно принадлежащих ему файлах.
- `dashboard-polling.ts` и `use-dashboard.ts`: последовательное чтение через
  TanStack Query, manual refetch с cancelRefetch:false, фоновые обновления только
  при visible+online+active page. Visibility/offline/pagehide немедленно вызывают
  cancelQueries и прерывают fetch/body. Возобновление возможно при возвращении.
  Последний хороший snapshot сохраняется; неполный snapshot отклоняется.
  После ошибок задержки30/60/120секунд с base15; Retry-After429/503 учитывается
  как нижняя граница в диапазоне15..120секунд. Успех сбрасывает backoff.
  401/409 останавливают polling;409 предлагает явное перечитывание shell.
- ControlContent показывает fatal state только без cached PageData. Ошибка
  refetch при сохранённом page отображает notice и сохраняет тот же TaskDetail,
  answer draft и состояние pending action вместо unmount.
- `skipTrailingSlashRedirect:true` в Next config сохраняет Go canonical `/web/`.
  Поддержка подтверждена Next16.4.2 config-shared.d.ts1696 и config-schema.js832.
  Это устраняет конфликт Next `/web/`→`/web` с Go `/web`→`/web/` при redirect:error.

Авторские проверки текущей delta: TypeScript PASS, formatter PASS, unit37/37
PASS вместе с координаторскими mutation deadline tests. Новые unit проверяют
8секунд на зависшее body, parent cancellation, отсутствие start после abort,
снятие timer/listener после успеха, backoff/Retry-After/domain errors и canonical
root JSON URL. Production build этой delta и process/browser recheck автором
**NOT-RUN** по прямому запрету координатора на изменение frozen artifact;
координатор соберёт новую версию после повторного независимого approve.

### Дополнительные два gate127

`validDashboard` теперь сохраняет TASK-006 nested validation: opaque session ID,
agent role/status, строковые owner/machine/run ID, действительные даты,
безопасные task key/title и event cursor/type/at/provenance. Коллекции ограничены
6агентами/20событиями; nil Go collections нормализуются в пустые массивы только
после валидации. Unit с настоящим QueryClient подтверждает: malformed response
отклоняется и не заменяет последний хороший snapshot.

`appToWeb` убирает trailing slash у nonroot экранов, сохраняя canonical Go root
`/web/`; unit проверяет login/tasks/agents/designs и сохранение query.
TypeScript и unit40/40 PASS; formatter и diff-check PASS. Build по-прежнему
NOT-RUN автором; финальный артефакт собирает координатор после approve127.

### Исправление реального root404 из artifact2

Сохранён API root `/app`→`/web/` и прежний Go `view=tasks|questions` контракт.
Next16.4.2 `prepareDestination` компилирует пустой wildcard destination
`/web/:path*` в `/web` без завершающего slash; одного отключения308 было
недостаточно. Литеральный destination `/web/` сохраняет slash. В `browserRewrites`
добавлено правило source `/web/` → fixed upstream `/web/` **перед** wildcard.
Next config использует этот helper. Дополнительного `/web/overview` mapper нет.

Два regression unit вызывают фактический установленный Next prepareDestination:
проверяют literal `/web/`, прежнее stripping wildcard, неизменные task segments
и query (включая valid/unknown/duplicate view, чтобы Go продолжал валидацию).
Все42 unit PASS; TypeScript/formatter PASS. Runtime root login smoke выполняет
координатор после build; автор по поручению build не запускал.
