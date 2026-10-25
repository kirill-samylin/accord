// Fixed worker diagnostics (internal/worker) explained for the owner. The
// original text stays visible; this only adds what to do next. Unknown text
// gets no hint rather than a guess.
export const consentCommand =
  "agent-board worker consent --config <файл worker.json>";

export const recoverCommand =
  "agent-board worker recover --config <файл worker.json> --run <ID запуска>";

const hints: Array<[RegExp, string]> = [
  [
    /workspace_dirty/,
    "В рабочей папке есть незакоммиченные изменения, которые worker не может приписать прерванному шагу. Владелец проверяет папку (git status), сохраняет или убирает изменения и продолжает цикл.",
  ],
  [
    /interrupted_work_stashed/,
    "Частичная работа прерванного шага сохранена в git stash (ссылка в отчёте запуска); вернуть её можно командой «git stash apply <sha>».",
  ],
  [
    /(?:workspace_quarantined|needs_recovery)/,
    "Владение процессом на компьютере не подтверждено. Владелец проверяет, что агент остановлен, и выполняет «" +
      recoverCommand +
      "», затем продолжает или отменяет цикл.",
  ],
  [
    /local_budget_exhausted/,
    "Исчерпан локальный предел запусков или срок на компьютере. Владелец увеличивает пределы в worker.json и обновляет согласие (worker consent).",
  ],
  [
    /(?:local workspace or provider identity changed|local path identity changed|local executable\/workspace identity changed)/i,
    "На компьютере изменился исполняемый файл агента или рабочая папка — например, обновились Codex или Claude. Worker не запускает изменённую программу без согласия владельца. Владелец на этом компьютере останавливает worker, выполняет «" +
      consentCommand +
      "», запускает worker снова и продолжает цикл.",
  ],
];

export function workerErrorHint(message: string | undefined | null) {
  if (!message) return undefined;
  return hints.find(([pattern]) => pattern.test(message))?.[1];
}
