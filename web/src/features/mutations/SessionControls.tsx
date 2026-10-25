"use client";

import { useRef } from "react";
import { actionBody, actionID, type MutationProps } from "./contracts";
import { useAction } from "./useAction";
import { ActionNotice } from "./ActionNotice";

// Continuing an expired session keeps the person on the page they were on:
// the server's generic redirect (project overview) is dropped unless it leads
// to sign-in, so the caller refreshes the current deep link instead.
export function SessionControls({ page, onSuccess }: MutationProps) {
  const continuing = useRef(false);
  const action = useAction(page, (result) => {
    if (
      continuing.current &&
      result.redirect &&
      !/\/login(?:[/?]|$)/.test(result.redirect)
    )
      onSuccess({ ...result, redirect: undefined });
    else onSuccess(result);
  });
  function submit(operation: "continue" | "logout") {
    continuing.current = operation === "continue";
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
            Рабочая сессия истекла из-за бездействия. Продолжите её, чтобы снова
            видеть проект и выполнять действия.
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
