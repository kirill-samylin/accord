"use client";

export function ActionNotice({
  message,
  uncertain,
  busy,
  onRetry,
}: {
  message: string;
  uncertain: boolean;
  busy: boolean;
  onRetry: () => void;
}) {
  if (!message) return null;
  return (
    <div className="mutation-notice" role="alert">
      <p>{message}</p>
      {uncertain ? (
        <button
          type="button"
          className="button button-secondary"
          disabled={busy}
          onClick={onRetry}
        >
          Повторить ту же отправку
        </button>
      ) : null}
    </div>
  );
}

export function FieldError({
  name,
  errors,
}: {
  name: string;
  errors: Record<string, string>;
}) {
  return errors[name] ? (
    <span className="field-error" id={`${name}-error`}>
      {errors[name]}
    </span>
  ) : null;
}
