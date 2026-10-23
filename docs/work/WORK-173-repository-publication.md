# WORK173 — публикация Accord и объединение истории

Координатор root. Пользователь 23.10.2026 поручил сначала перенести проект в `https://github.com/DenisShagi/accord.git`, объединив все ветки в одну, затем продолжить Mac native diagnosis. Отдельным ответом выбрал текущий код + всю историю, без возвращения экспериментальных функций. Review174, independent branch audit175.

## Подготовка

Target remote read и push dry-run PASS, target heads/tags отсутствуют. Прежний origin сохранён; полного mirror/push внутренних снимков не будет. До изменений создан private all-refs Git bundle и inventory в ignored `.local/repository-publication/`.

Inventory24 local heads +7 old remote branch heads, divergent12local +1remote spec, coverage5tips. Автор175 и reviewer174 подтвердили реальные isolated D/E features, несовместимые с current schema5; выбор пользователя не выдаёт их за функционально интегрированные. Публичный disposition manifest — docs/repository-consolidation.md.

Current production/native checkpoint исправлен честно: server273/schema5 active, Mac enrolled, actual GUI native preparation FAIL/no session/claim, config-only probe PASS без установленной причины; finalization pending. GUI files/config/worker/service во время публикации не меняются.

## Gate

История, MIT LICENSE и product tree сохраняются; source of truth — current273 + только documentation/checkpoint corrections. Старые ветки входят ancestry через explicit ours merge. Все branch tips должны быть ancestors итоговой main, push только main и remote ref readback exact SHA. Existing local branches/worktrees не удаляются.

Known real credential scan и tracked private key patterns выполняются без вывода значений. Historical public HTTPS links preserved in old commits; currentdoc copies generalized. deploy-task012.command, .local/runtime, refs/codex snapshots excluded. Документы GUI158–161 сохраняют dated scope, current guide указывает schema5/TASK013.

Publication ещё NOT-RUN; final review174 и ancestry/tree/secret proof pending. После публикации продолжается narrow native preparation diagnosis без ослабления ownership guards.
