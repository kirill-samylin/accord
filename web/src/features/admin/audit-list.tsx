"use client";
import { useState } from "react";
import { ScrollText } from "lucide-react";
import type { AdminPage } from "@/lib/projects";
import { plural } from "@/lib/format";
import {
  auditDetails,
  actorName,
  auditEvent,
  auditSubject,
  auditTime,
} from "./admin-model";
const pageSize = 40;
export function AuditList({ page }: { page: AdminPage }) {
  const [limit, setLimit] = useState(pageSize);
  const entries = page.audit.slice(0, limit);
  return (
    <section className="admin-section" aria-labelledby="admin-audit-title">
      <div className="admin-section-head">
        <h2 id="admin-audit-title">Журнал</h2>
        <span className="muted">
          {`${plural(page.audit.length, "Последняя", "Последние", "Последние")} ${page.audit.length} ${plural(page.audit.length, "запись", "записи", "записей")} · новые первыми`}
        </span>
      </div>
      {entries.length ? (
        <ol className="admin-audit">
          {entries.map((entry) => (
            <li className="admin-audit-row" key={entry.id}>
              <div className="admin-audit-main">
                <strong>{auditEvent(entry.event)}</strong>
                {auditSubject(page, entry) ? (
                  <span className="admin-audit-subject">
                    {auditSubject(page, entry)}
                  </span>
                ) : null}
                {entry.details ? (
                  <span className="muted break">
                    {auditDetails(entry.details)}
                  </span>
                ) : null}
              </div>
              <div className="admin-audit-meta">
                <span className="truncate" title={entry.actor_id}>
                  {actorName(page, entry.actor_id)}
                </span>
                <time className="entity-date">{auditTime(entry.at)}</time>
              </div>
            </li>
          ))}
        </ol>
      ) : (
        <div className="empty-state">
          <ScrollText size={30} />
          <h3>Журнал пока пуст</h3>
          <p className="muted">
            Здесь появятся выдача и отзыв токенов, изменения ролей и людей.
          </p>
        </div>
      )}
      {page.audit.length > limit ? (
        <button
          type="button"
          className="button button-ghost"
          onClick={() => setLimit((value) => value + pageSize)}
        >
          Показать ещё
        </button>
      ) : null}
    </section>
  );
}
