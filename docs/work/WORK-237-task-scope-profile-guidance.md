# WORK237 — точная область чтения и текущая версия вопроса

Автор root, 24.10.2026. SOURCE ONLY; independent review202 требуется до GO.

WORK236 независимо подтвердил F6V10 FAIL: новый developer владел T-3, запросил
упомянутую reviewer T-4 и получил корректный FORBIDDEN_ROLE. Он соблюдал refusal
stop, затем вопрос с устаревшей version1 получил CONFLICT. Исходник/HEAD не менял.
Четыре native закрыты с ACK; pending/leases/reservations0. Failure не переписан.

Контракт202 разрешает узкую правку данных профиля, без новой сборки: замечания
уже доставлены через workflow.Messages в prompt. Developer/analyst GET только
своей задачи; reviewer/tester сохраняют разрешённое чтение explicit собственного
Review.TargetKey. Ссылки на другие reviewer задачи не дают права GET. Для вопроса
прочитать текущий own MCP resource accord://workflow, использовать его version
и живую task lease generation, не managed generation и не константу1. Отказ,
конфликт и lost lease по-прежнему останавливают работу, обходов/hidden retry нет.

Новые замороженные пакеты:

- `.local/task014/pipeline210/f6-v11/source-pins.json`:
  `724a404760d5f936614086168b90007d0ebdd93f8709302c836162c4673ee252`,81members.
  Из inherited source изменены только одноразовое имя gate и profile text.
- `.local/task014/onboard224/v5/source-pins.json`:
  `26e1d1088d602c22e9e7518d2227ce4855fa825fc10f54e33a98a2c78321a5a9`,25members.
  Из inherited source изменены только инструкции четырёх профилей.

Старые пакеты сохранены; runtime не запускался. Binary/server/worker, ACL,
refusal rules, tools, assertions, fixture/runner, budgets6native/1fix/0continuation
не менялись. Target scaffold Test и его bootstrap commit остаются прежними.
Private metadata600/directories700, F6 fixture644; static JSON/SHA/syntax checked.
Только exact independent SOURCE APPROVE и новый root GO разрешают отдельный trial.


Independent SOURCE APPROVE202 получен для обоих пакетов. Receipts
210-f6-v11-source202.json34cc1fb7046e70838f4d83830c793524d5654b87a97e673be5eecb49bc8afac8
и onboarding224-v5-source202.json57459beb36f68561f2cee6d1ba5d5a0c0f3654b0f4a268f5dda24380243efbc0.
Root49CJS/JSON/memberSHA/metadata600/fixture-scaffold644 PASS.
Fresh root input0161fa9167b6dac33cdde7928c01062b129754cd1beb221be09beec87fb108fb,
GO4c5940f9c9197e6309f96430a21f86b454d596dbb9e2f3b6efc3dc45dd414f38,
runtime pipeline210-f6-1792798761726100000. Один6native/1fix/0continuation trial
IN PROGRESS; до exit и независимой приёмки PASS отсутствует.


Промежуточный F6V11 observation: D1 commit49f38bbdbdff721ec1e6149d0a973aaabc10dcbe
и настоящий check exit1; независимый R1 changes_requested. D2 без parent помощи
создал новый commit5506d9d2b5b266776d2a0861ee28972b50259775, actual check exit0,
свой task_generation2 completed. R2 completed, tester queued. Полный harness
ещё жив; этот промежуточный факт не объявляет terminal/native6/QA PASS.


F6V11 actual exit0/PASS, resultsSHA
1ff6815c122b09d54246c5c444bb49bab239b518190f2f0ac23dca05db994dfe.
Шесть настоящих native stages completed, D1bad→R1changes_requested→D2newcommit
→R2approve→Tpassed; harness native_pipeline_proven=true/checks3. Независимый
WORK238 выполняет actualevidence приёмку; source/PASS report сам не заменяет её.
