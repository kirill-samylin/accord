# WORK182 — установка приватной диагностики Mac worker

23.10.2026. Координатор/оператор root; author WORK176, independent reviewer
WORK169, separate QA WORK177. Это prerequisite TASK013/TASK014; автоматический
workflow не реализован, исходный GUI native preparation FAIL не объяснён.

## Проверенный исходник

WORK176 source + author unit/race APPROVE169. WORK177 actual Go config-only
probe прошёл3 discovery/verification пары из обычного процесса и3 пары из
отдельной LaunchAgent: все12 фаз PASS,31–56ms, без ownership uncertainty.
Exact owning363 Go sources, GUI3316 files, persistent config/plist/binary
сохранены; временная служба удалена. FINAL177 EVIDENCE APPROVE169 получен.
Это не воспроизведение ошибки и не доказательство её исправления.

Локальный commit `62e354525b83a2b671ab37ede938ffe7e838e24b` содержит только
reviewed176 diagnostic delta и WORK176. Clean attached build checkout
переиспользован; Darwin arm64/CGO0/trimpath build с exact VCS revision и
`vcs.modified=false` PASS. SHA256 binary:
`10e845ff702e2b1f9d9bcac4ac8cdd278c4aba65e5fd839104dd0cc822c6d429`.
Build artifacts и private operation source/proofs находятся только в `.local/`.
Сервер остаётся прежней версии, схема и данные не обновляются этим действием.

## План и границы

Независимый PLAN APPROVE169: только замена локального worker binary после
idle/pending checks, доказанной остановки принадлежащей службы и отсутствия
её children. Полный private backup old binary/config/plist/state; запись
только нового binary атомарно, прежние config, token, host proof, local consent,
provider flags и plist неизменны. Bootstrap той же user LaunchAgent,
ожидание fresh worker incarnation/epoch и повторная проверка authority.
Новый journal не заменяется старым backup.

Конкретный private installation source передан169 до выполнения. Затем
отдельный QA использует frozen ранее reviewed readonly GUI adapter с narrow
expected-binary-pin/local-worker-revision delta и собственным reapprove.
Единственный bounded retry: настоящее describe/next/README hash, exact native
conversation resume, stop/clarify между двумя контроллерами. Никаких изменений
GUI, commits, глобальных permissions или подключения чужого physical PC.

## Фактический статус

Установка — **PASS**: concrete source получил REQUEST CHANGES169 на fail-path
повторного bootstrap и обязательную запись FAIL proof; оба замечания исправлены
до исполнения. Source SHA256
`42a478595389a91f0740be3ae41b012daabb3b64fde1c15fb134259a1cbcfd44`
получил независимый SOURCE APPROVE169. Единственное исполнение завершилось
exit0/PASS: old worker exit доказан, full private backup создан, binary заменён
атомарно, fresh worker incarnation с большим epoch получила lease. Config/plist
и host authority unchanged; source GUI и сервер не менялись. Финальный
независимый readback installation evidence — **APPROVE169**.

Повтор native GUI integration — **FAIL**, один bounded запуск после exact root GO
на reviewed frozen adapter490d13. Новый diagnostic:
`prepare_discovery_timeout`, без ownership suffix, generation1, provider
conversation не начат. Это локализует отказ в discovery, но пока не root cause.
QA сохранил предыдущие FAIL; QA cards cancelled/profile archived, GUI baseline,
worker binary/config и adapter unchanged, persistent worker running. Повтора нет.

Root нашёл отличие process context: действующий LaunchAgent имеет
`ProcessType=Background`, а временная config-only проба177 оставляла ProcessType
неустановленным (Standard). Её PASS остаётся действительным для заявленных
configured env/cwd, но не доказывает runtime Background. На основании системной
man5 launchd.plist и Apple OSS man источник говорит о разных resource limits;
гипотеза проверяется отдельным QA184 comparison, без изменения постоянной службы
и без model calls. Таймаут и ownership guards не увеличивались/не отключались.

Windows/Claude/другой physical PC — NOT-RUN.
При отказе сохраняются diagnostic stage и все предыдущие FAIL; автоматического
цикла повторных model calls или объявления root cause fixed не предусмотрено.
