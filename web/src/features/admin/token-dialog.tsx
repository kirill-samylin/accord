"use client";
import { useEffect, useState } from "react";
import { Copy, Download, Eye, EyeOff, TriangleAlert } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { AdminPage, Person } from "@/lib/projects";
import type { CatalogMutationResult, IssuedCredential } from "@/lib/types";
import { date } from "@/lib/format";
import { RegistryForm } from "../projects/registry-form";
import { tokenFileName, validateCredential } from "./admin-model";
// Issues a credential for a person. The returned secret is handed to the
// caller as a plain value: it must stay in component state and never reach
// the query cache, the URL, storage or logs.
export function TokenIssueForm({
  page,
  person,
  onIssued,
  onSuccess,
}: {
  page: AdminPage;
  person: Person;
  onIssued: (issued: IssuedCredential) => void;
  onSuccess: (result: CatalogMutationResult) => void;
}) {
  // A new form after each issue: the previous machine/days are not reused
  // by accident for a second token.
  const [round, setRound] = useState(0);
  return (
    <RegistryForm
      key={round}
      page={page}
      actionKey={`admin:person:${person.id}:credential`}
      endpoint={`/web/catalog/admin/people/${person.id}/credentials`}
      title="Выдать токен"
      validate={validateCredential}
      onSuccess={(result) => {
        if (result.issued) onIssued(result.issued);
        setRound((value) => value + 1);
        onSuccess({ ...result, issued: undefined });
      }}
    >
      <div className="form-grid">
        <label className="form-field">
          <span>
            Компьютер<span className="required-mark"> *</span>
          </span>
          <input
            name="machine"
            maxLength={64}
            autoComplete="off"
            spellCheck={false}
            placeholder="ноутбук, рабочая станция…"
          />
        </label>
        <label className="form-field">
          Срок, дней
          <input
            name="days"
            type="number"
            inputMode="numeric"
            min={1}
            max={365}
            step={1}
            defaultValue={30}
          />
        </label>
      </div>
      <p className="form-help">
        Владелец токена — {person.name}. Секрет будет показан один раз сразу
        после выдачи.
      </p>
    </RegistryForm>
  );
}
function download(name: string, secret: string) {
  const blob = new Blob([`${secret}\n`], { type: "text/plain" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = name;
  link.rel = "noopener";
  document.body.appendChild(link);
  link.click();
  link.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}
export function IssuedSecretDialog({
  issued,
  onClose,
}: {
  issued: IssuedCredential | null;
  onClose: () => void;
}) {
  const [visible, setVisible] = useState(false);
  const [copied, setCopied] = useState<"" | "done" | "failed">("");
  useEffect(() => {
    setVisible(false);
    setCopied("");
  }, [issued]);
  const secret = issued?.secret ?? "";
  const fileName = issued ? tokenFileName(issued.owner, issued.machine) : "";
  async function copy() {
    try {
      await navigator.clipboard.writeText(secret);
      setCopied("done");
    } catch {
      setCopied("failed");
    }
  }
  return (
    <Dialog
      open={Boolean(issued)}
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      {/* A live secret is shown exactly once: Esc, an outside click and the ×
          would silently lose it, so only the explicit button closes. */}
      <DialogContent
        className="secret-dialog"
        aria-describedby="issued-secret-description"
        showCloseButton={!secret}
        onEscapeKeyDown={(event) => {
          if (secret) event.preventDefault();
        }}
        onPointerDownOutside={(event) => {
          if (secret) event.preventDefault();
        }}
        onInteractOutside={(event) => {
          if (secret) event.preventDefault();
        }}
      >
        <DialogHeader>
          <DialogTitle>Токен выдан</DialogTitle>
        </DialogHeader>
        {issued ? (
          <div className="secret-body">
            <p id="issued-secret-description" className="muted">
              {issued.owner} · {issued.machine} · действует до{" "}
              {date(issued.expires_at)}
            </p>
            <div className="notice error-notice secret-warning" role="alert">
              <TriangleAlert size={18} aria-hidden />
              <span>
                Секрет показывается только сейчас и больше не будет доступен.
                Сохраните его прежде чем закрыть окно.
              </span>
            </div>
            {secret ? (
              <>
                <div className="secret-box">
                  <code className="mono break" aria-label="Секрет токена">
                    {visible ? secret : "•".repeat(Math.min(secret.length, 48))}
                  </code>
                </div>
                <div className="secret-actions">
                  <button
                    type="button"
                    className="button button-secondary"
                    aria-pressed={visible}
                    onClick={() => setVisible((value) => !value)}
                  >
                    {visible ? <EyeOff size={16} /> : <Eye size={16} />}
                    {visible ? "Скрыть" : "Показать"}
                  </button>
                  <button
                    type="button"
                    className="button button-secondary"
                    onClick={() => void copy()}
                  >
                    <Copy size={16} />
                    {copied === "done"
                      ? "Скопировано"
                      : copied === "failed"
                        ? "Не удалось скопировать"
                        : "Скопировать"}
                  </button>
                  <button
                    type="button"
                    className="button"
                    onClick={() => download(fileName, secret)}
                  >
                    <Download size={16} />
                    Скачать файл
                  </button>
                </div>
                <p className="form-help">
                  Файл <code className="mono">{fileName}</code> содержит только
                  секрет. Передайте его владельцу: файл токена выбирается на
                  странице входа.
                </p>
              </>
            ) : (
              <p className="notice" role="status">
                Секрет уже был выдан при первой отправке и повторно не
                показывается. Если он не был сохранён — отзовите этот доступ и
                выдайте новый.
              </p>
            )}
            <div className="form-actions">
              <button type="button" className="button" onClick={onClose}>
                Я сохранил секрет, закрыть
              </button>
            </div>
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
