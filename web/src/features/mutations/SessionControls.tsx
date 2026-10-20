"use client";

import { actionBody, actionID, type MutationProps } from "./contracts";
import { useAction } from "./useAction";
import { ActionNotice } from "./ActionNotice";

export function SessionControls({ page, onSuccess }: MutationProps) {
  const action = useAction(page, onSuccess);
  function submit(operation: "continue" | "logout") {
    void action.send(
      operation === "continue" ? "/web/session/continue" : "/web/logout",
      actionBody(action.page, operation),
    );
  }
  return (
    <div className="session-controls">
      {action.page.can_continue ? (
        <div className="session-continuation">
          <p className="form-help">
            Рабочая сессия завершилась. Продолжите её явно, чтобы выполнять
            новые действия.
          </p>
          <button
            className="button"
            disabled={action.locked || !actionID(action.page, "continue")}
            onClick={() => submit("continue")}
          >
            Продолжить работу
          </button>
        </div>
      ) : null}
      <button
        type="button"
        className="button button-secondary"
        disabled={action.locked || !actionID(action.page, "logout")}
        onClick={() => submit("logout")}
      >
        Выйти
      </button>
      <ActionNotice
        message={action.message}
        uncertain={action.phase === "uncertain"}
        busy={action.busy}
        onRetry={() => void action.retry()}
      />
    </div>
  );
}
