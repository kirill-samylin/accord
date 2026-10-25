// Fixed worker diagnostics (internal/worker) explained for the owner. The
// original text stays visible; this only adds what to do next. Unknown text
// gets no hint rather than a guess.
export const consentCommand =
  "agent-board worker consent --config <файл worker.json>";

const hints: Array<[RegExp, string]> = [
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
