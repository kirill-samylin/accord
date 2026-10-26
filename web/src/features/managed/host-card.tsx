"use client";
import { useState } from "react";
import { Monitor } from "lucide-react";
import { Disclosure } from "@/components/disclosure";
import { date } from "@/lib/format";
import {
  hostOnline,
  providerName,
  type ManagedHost,
} from "@/lib/managed-agents";
export function HostCard({
  host,
  observedAt,
}: {
  host: ManagedHost;
  observedAt?: string;
}) {
  const online = hostOnline(host, observedAt);
  return (
    <article className="managed-host settings-section">
      <div className="managed-row">
        <Monitor size={18} />
        <h3>{host.name}</h3>
        <span
          className={`managed-status ${online ? "managed-online" : "muted"}`}
        >
          {host.state === "disabled"
            ? "Отключён"
            : online
              ? "На связи по последним данным"
              : "Нет связи по последним данным"}
        </span>
      </div>
      <p className="form-help">
        Последняя связь:{" "}
        {Date.parse(host.last_seen_at) > 0
          ? date(host.last_seen_at)
          : "ещё не подключался"}
      </p>
      <p>
        {host.providers.map(providerName).join(" · ") || "Агенты не разрешены"}
      </p>
      <p className="muted break">
        Рабочие папки: {host.workspaces.join(", ") || "не разрешены"}
      </p>
      <p className="form-help">
        {host.can_control
          ? "Владелец разрешил вам управление."
          : "Управление доступно с разрешения владельца компьютера."}
      </p>
    </article>
  );
}
export function EnrollmentHelp({
  actorPrincipalID,
}: {
  actorPrincipalID?: string;
}) {
  const [copyStatus, setCopyStatus] = useState("");
  async function copy() {
    if (!actorPrincipalID) return;
    try {
      await navigator.clipboard.writeText(actorPrincipalID);
      setCopyStatus("Код скопирован. Передайте его владельцу компьютера.");
    } catch {
      setCopyStatus(
        "Не удалось скопировать автоматически. Выделите код и скопируйте его вручную.",
      );
    }
  }
  return (
    <Disclosure
      className="settings-section managed-setup"
      summary="Подключить компьютер"
    >
      {actorPrincipalID ? (
        <div className="managed-control-code">
          <p>Ваш код управления для подключения компьютера</p>
          <code className="mono break">{actorPrincipalID}</code>
          <button
            className="button button-secondary button-sm"
            type="button"
            onClick={() => void copy()}
          >
            Скопировать код
          </button>
          <p className="form-help">
            Это код вашей учётной записи в этом проекте. Коллега указывает его
            при локальном разрешении управления своим компьютером. Код не
            является паролем и сам по себе не даёт доступа.
          </p>
          {copyStatus ? (
            <p className="form-help" role="status">
              {copyStatus}
            </p>
          ) : null}
        </div>
      ) : null}
      <p>
        Каждый участник устанавливает локальный worker Accord и входит в Codex,
        Claude Code или корпоративный Gennadiy на своём компьютере. Владелец
        выбирает проект, разрешённую рабочую папку и агента.
      </p>
      <p>
        Чтобы вы могли управлять агентом коллеги, он явно разрешает управление
        вашей учётной записи при настройке worker. Само участие в проекте этого
        права не даёт.
      </p>
      <p>
        После подключения оставьте worker включённым. Здесь появится компьютер и
        время последней связи. Открытые ранее чаты в IDE и терминале не
        подключаются к управлению.
      </p>
      <p className="form-help">
        Секреты доступа передаются владельцу отдельно. Не вставляйте их в
        задания или уточнения. Локальная аварийная остановка доступна владельцу
        через worker, даже без сети.
      </p>
    </Disclosure>
  );
}
