# TASK019 — единственный интерфейс и понятный выбор файла токена

Поручение25.10.2026: полностью удалить из кода упрощённый HTML/no-JS интерфейс,
включая вход `/web/login`, и исправить непонятный нативный выбор файла токена.
Публикация и стенд продолжают ранее порученный цикл в исходном origin
`https://github.com/kirill-samylin/accord.git`, ветка `codex/native-preflight`.
Root — координатор; WORK290 backend removal, WORK291 token UI, WORK292 review.

Единственный UI — Next `/app`. JSON `/web` endpoints, cookie login, CSRF/action-ID,
leases/epochs/ACL/capabilities и scoped downloads/preview сохраняются. Известные
legacy GET/HEAD pages перенаправляются на corresponding fixed relative `/app`
адрес без выдачи page action nonce; неизвестные страницы/static404. NonJSON POST
отклоняется406 до body/мутации, foreign Origin по-прежнему403. Native REST/MCP
и файлы не становятся HTML или публичными redirects.

Полностью удаляются internal/webui templates/assets/JS/CSS/fonts и Renderer/
Static wiring. Next fallback links/menu/noscript/error fallback удалены. Старые
SSR-only browser/control/refinement тесты удаляются, shared Next/lifecycle/upload
coverage сохраняется; meaningful backend/security assertions мигрируются на JSON.
Исходный TZ и исторические evidence документы не переписываются.

Выбор токена: custom accessible кнопка, отдельное имя/статус, hint, замена/очистка;
содержимое файла не читается JS, не сохраняется и не выводится. Multipart4096byte
граница, CSRF и точный повтор uncertain запроса остаются. Independent token UI
source review APPROVE; typecheck/109tests/format PASS на промежуточном дереве,
actual browser/candidate/build/deployment пока NOT-RUN.

Baseline production — bdcabbd, schema6. Только coherent capable all-project
Go+Next update с format2backup, preservation/history/ACL/floors и roll-forward.
Новый кандидат не устанавливается до independent source/tests/ops approvals.

## Source готов к local candidate commit

Независимое source review — APPROVE/PASS review_changes;720members и51deletion
зафиксированы отдельно от evolving docs. Новые Go regressions проверяют actual
single/registry HTTP routes, no-cookie/no-nonce stateless redirects, canonical
scope/unknown404, unread body nonJSON406/foreign403, no logout mutation и binary
anonymous/revoked401. Прежние JSON auth/CSRF/replay/ACL/download byte assertions
сохранены. Full go test ./... PASS12testpackages, web109/type/format PASS,
infra174 PASS. Полный race/vet/build и actual process/HTTPS приёмка ещё идут.

При полной infra проверке обнаружены3 прежних stale fixture ожидания: unknown
schema6 после появления6 и fake native ACK без реальной миграции/reset в4/5
copy tests. Исправлены только tests: future7/history6 и проверка actual rejection
с неизменными copied bytes/boot guard/no RESET_COMPLETE. Production infra не
изменялась; successful native restore6 этим тестом не заявлен. Первоначальные
FAIL логи сохранены, independent review тестовых исправлений APPROVE.

Независимый tester после product approve мигрировал upload-process.cjs на
JSON/Accept и406, сохранив50MiB/+1byte/31s stream/120s timeout cleanup. Изменено
только stale heading «Войти» в существующем next-security. New test-source
review APPROVE до actual runtime; общая починка старого Next E2E harness не
входит в этот выпуск. Actual native Windows/Claude execution/второйПК NOT-RUN.
