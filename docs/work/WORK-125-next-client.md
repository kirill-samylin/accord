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
