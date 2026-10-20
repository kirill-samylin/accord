# WORK-129 — ограничения времени отправки форм

Родитель TASK-007, координатор/автор root. Дата20.10.2026.
Область записи: `web/src/features/mutations/deadline.ts`,
`deadline.test.ts`, подключение в `useAction.ts`; общие документы root.
Существующая область126 передана root после завершения его реализации;
API signal добавляет125, независимое review выполняет127. Runtime и QA пишет128.

## Причина и контракт

При зависшем fetch прежний новый клиент мог остаться в `sending` без возможности
повтора. Ограничиваем весь Promise отправки, включая чтение JSON: обычная форма
15секунд, upload `/web/designs`130секунд. Upload сохраняет Go deadline120секунд
и границу50МиБ. Таймаут не доказывает отказ транзакции: существующий catch
переводит действие в uncertain, сохраняет прежние action ID, session и payload.
Нового автоматического POST, нового ID или самовольного heartbeat нет.
Ручной retry снова отправляет тот же snapshot с новым локальным таймаутом.

## Проверки

PASS:4 авторских unit на Node24.19.0 с управляемыми таймерами: ordinary15s
без автоматического retry, upload не прерывается ранее130s, успешное завершение
снимает таймер. Реальный `submitAction` с mock fetch отдельно получает headers200,
затем зависает на JSON body: abort даёт UNKNOWN_RESULT/retry_same_action,
автоматического POST нет, явный retry передаёт тот же body/action ID.
Полный клиентский suite31/31 и TypeScript PASS на промежуточной дельте.
Независимый127 scoped review APPROVE; productionbuild и реальный browser
timeout/recovery ожидаются после остальных обязательных исправлений125.

Не смешивать этот mutation deadline с read-only dashboard8s: отдельную регрессию
polling/last-good/draft persistence исправляет125 в своей области.
