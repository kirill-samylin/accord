# TASK016 — новое поручение из экрана управления

24.10.2026. Пользователь повторно сообщил о неисполненном поручении создать
`c1.js` в корне Test_internet с `console.log(1)`. Run
`mr_93545b960e7c112be4e7a6e87c37cd22` completed: native процесс успешно
завершился, но отказался писать без task/lease. Это не повтор pin failure
[WORK278](work/WORK-278-test-internet-pin-repair.md) и не выполнение задания.

Пользователь явно выбрал: **новое поручение проходит весь цикл автоматически —
задача, разработка, независимые review и QA**. Используется существующий
[workflow TASK014](automation.md), его разрешения и ограничения сохраняются.

## Изменение поведения

- Основной вход из «Управления» отправляет новое поручение в существующую
  форму полного цикла с явным выбором действующей команды/версии разрешения.
  Одна отправка запускает дальнейший цикл; политика не выбирается и не
  продлевается скрыто. Черновик не выдаётся за запущенную работу.
- Прежний запуск отдельного агента остаётся ручным: для изменения файлов
  нужна задача и отдельно полученная самим агентом актуальная аренда.
  Без задачи допускается только явно выбранное поручение без изменений.
  Указание «только чтение» не является обещанием дополнительной OS-изоляции.
- Исходный текст taskless run можно перенести в форму полного цикла в памяти
  текущего project scope; без текста в URL/глобальном хранилище, автоподтверждения
  либо отправки при переключении проектов. Mutation uncertainty сохраняется.
- Completed — завершение шага процесса. Выполнение требования подтверждается
  задачами, фактическим SHA, review и QA, а не exit0 или анализом слов модели.

## Владельцы и проверки

Реализация/UI unit/integration — [WORK281](work/WORK-281-taskless-launch-author.md).
Независимые architecture/security/source review —
[WORK282](work/WORK-282-taskless-launch-review.md). Отдельная process/UI/native QA
после source approve — WORK283. Capable production release — WORK284.
Root ведёт этот checkpoint, точечную подготовку локального разрешения и операции.

Для точного `c1.js` прежний local source scope недостаточен. Reviewable draft
добавляет только `c1.js`, повышает host/workspace/check versions1→2 и
пересчитывает preset digest. Остальные пути, tools, права, роли, controllers,
сроки, бюджеты и действующая policy сохраняются; policy обновляется как новая
immutable revision, без дубликата разрешения. До независимого approve draft
не применяется. Токены/host proof остаются private runtime вне Git.

Сценарий приёмки: настоящее исходное поручение → analyst plan → один developer
commit с единственным новым `c1.js` → отдельные exact-SHA review/tester и реальные
project checks. Проверка вывода Node допускается отдельно на exact reviewed
файле установленным runtime и явно отличается от workflow preset/JUnit.
Координатор не создаёт файл вместо native разработчика. Старая история сохранена.

Private evidence: игнорируемая `.local/task017-taskless-launch/`.
Production поставка только capable all-project upgrade/roll-forward с полным
backup/историей/ACL, отдельным review и HTTPS QA; старый binary/DB rollback запрещён.
Текущий статус: IN PROGRESS. Новые Windows/Claude execution/второй ПК — NOT-RUN.
