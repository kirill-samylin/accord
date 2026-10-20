# WORK-132 — независимый production smoke Next.js

TASK008, 20.10.2026. Отдельный tester; product/infra/VPS/Git не изменяет.
Владение: эта WORK, `tests/integration/next_vps_*`, private `.local/task008-qa`.
Прочитаны AGENTS, TASK008/007, next-web, API и прежние browser helpers.
Новые process/E2E написаны только после BOUNDED CODE APPROVE131 для WORK130;
до runtime новые tests проходят независимое review131.

## Контракт проверки

`next_vps_go132.py` root запускает через свой pinned SSH transport: Linux
candidate binary сверяется по явному SHA256, отдельный свободный loopback7895,
synthetic DB/token/web-data в приватной временной `/run/accord-qa132-*`.
Проверяются actual executable/readyz, JSON preauth, собственный isolated login
и authenticated overview. В finally завершаются только own process и fixture.
Реальные production DB/config/credentials payload не читает.

`next_vps_smoke.cjs` выполняется с отдельным Chrome context по действующему
HTTPS origin из приватного client config. Единственные mutations на realboard —
собственный browser login/logout. 10 разделов, light/dark,1440/390, Inter,
overflow и navigation, nonce CSP/cookies, static assets, legacy SSR, native401,
authenticated read своей browser board-session, MCP initialize/list12 без
describe/tool calls и без heartbeat. GET не должен продлевать session expiry.
Реальные screenshots/data/diagnostics сохраняются только в ignored private
directory, в tracked отчёте только counts/statuses/hashes.

## Статус

Syntax Python/Node — PASS. Независимый test review131 — **APPROVE**:
Go helper SHA256 `7f25fcdd234940549efc46e4eb30e7979036189135251184edc156f3a455f17d`,
browser SHA256 `d55a809c848868dd828fb3bc0ea0db9d7921c4307a984f3dac6d4fff4189e425`.
Root выполнил approved isolated helper на candidate Linux executable
`b6ae1f26955bf74d43056642ea33a95bd0f65a55673bb8ac45fa54273a641f3c`:
**PASS3/3** readyz, JSON login, cookie overview; own process stopped и fixture
removed. Private `.local/task008/linux-go-proof.json` прочитан tester.

После root apply/explicit GO проведён `production-01` на release
`80486fc472324230a9cda9dbe9906c41998fb19f`: **FAIL**. Root redirect `/app`,
native401 и metrics404 PASS. Первый login HTML200 не стал готовым за15s:
настоящие HTTP429 у Inter font и6 Next static chunks; browser mutations0,
page errors/CSP severe errors пусты. Это runtime delivery defect, root/reviewer
уведомлены сразу. Отчёт сохраняется приватно и не перезаписывается повтором.

`production-02` на `15182480dcfd4823db010423f240415bbc94871a`:
static delivery/CSP PASS, login POST200 и route `/app` достигнуты, sidebar
timeout15s, причина этим старым harness точно не локализована. Узкая diagnostic
delta (private HTTP path/status, failure PNG/body, cleanup message) получила
independent APPROVE131 до `production-03`.

`production-03`: login/cookie, authenticated own-session native GET и MCP12
PASS, static errors0. Дальше настоящий nginx429 при быстрых повторных full
document navigations. Private request statuses показывают RSC viewport prefetch
десятков `/app` страниц, делящих public rate budget с `/web` API. Фактически
осмотренный failure PNG — nginx429. Own logout POST200 PASS, прочих browser
mutations нет. Это не утверждение, что обычный начальный login03 был сломан:
он прошёл; агрессивные повторные matrix navigations усиливают burst.

После отдельного independent product APPROVE131 для отключения28 автоматических
Link prefetch добавлена meaningful regression: после settle и hover нет
запросов с prefetch marker, explicit sidebar click получает настоящий JSON200.
Отчёт хранит только счётчик/paths, не значения headers. Новые tests переданы131
на review до очередного runtime; исторические FAIL не перезаписаны.

Отсутствие действий записи не заменяет
полную функциональную приёмку; synthetic mutations проверены TASK007 отдельно.
Chrome viewport не равен физическому телефону/второму ПК, external Codex/Claude
и полная V1 не заявляются.
