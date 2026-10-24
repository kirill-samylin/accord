"use client";
import { useState } from "react";
import { PageHeading, EmptyState, Fact } from "@/components/common";
import { ProjectLink as Link } from "@/components/project-link";
import { date } from "@/lib/format";
import {
  commandLabels,
  deliveryLabels,
  hostOnline,
  providerName,
  runLabels,
} from "@/lib/managed-agents";
import type { MutationResult, PageData } from "@/lib/types";
import { AgentNavigation } from "../profiles/agent-navigation";
import { RunControls } from "./run-controls";
import { RequirementDialog } from "./requirement-dialog";
import { useProjectScope } from "../projects/project-context";
export function ManagedDetail({
  page,
  onSuccess,
}: {
  page: PageData;
  onSuccess: (result: MutationResult) => void;
}) {
  const [requirement, setRequirement] = useState(false);
  const scope = useProjectScope();
  const detail = page.managed_run;
  if (!detail)
    return (
      <div className="page">
        <EmptyState title="Работа не найдена" />
      </div>
    );
  const { run, commands } = detail;
  const host = page.managed?.hosts.find((entry) => entry.id === run.host_id);
  const online = hostOnline(host, page.managed?.observed_at);
  return (
    <div className="page managed-page">
      <AgentNavigation current="managed" />
      <Link href="/app/managed" className="inline-link" prefetch={false}>
        ← Вся работа агентов
      </Link>
      <PageHeading
        title={run.profile_snapshot?.name || providerName(run.provider)}
        description={`${providerName(run.provider)} · ${host?.name || "Компьютер недоступен"} · ${run.workspace}`}
      />
      {page.managed?.observed_at ? (
        <p className="form-help">
          Последние данные сервера: {date(page.managed.observed_at)}. Состояние
          могло измениться после получения данных.
        </p>
      ) : null}
      <div className="managed-summary list-surface">
        <div>
          <span className="muted">Последнее состояние процесса</span>
          <strong>{runLabels[run.state] ?? run.state}</strong>
          <small className="muted">Подтверждение: {date(run.updated_at)}</small>
        </div>
        <div>
          <span className="muted">Связь с компьютером</span>
          <strong className={online ? "managed-online" : "muted"}>
            {online
              ? "На связи по последним данным"
              : "Не на связи по последним данным"}
          </strong>
          <small className="muted">
            {host ? date(host.last_seen_at) : "Нет данных"}
          </small>
        </div>
        <div>
          <span className="muted">Задача Accord</span>
          {run.task_key ? (
            <Link
              className="inline-link"
              href={`/app/tasks/${run.task_key}`}
              prefetch={false}
            >
              {run.task_key}
            </Link>
          ) : (
            <strong>Не указана</strong>
          )}
          <small className="muted">
            Завершение шага агента не завершает задачу.
          </small>
        </div>
      </div>
      {run.last_error ? (
        <div className="notice error-notice break" role="alert">
          {run.last_error}
        </div>
      ) : null}
      {!run.task_key ? (
        <section className="notice screen-stack">
          <p>
            Эта работа запущена без задачи. Изменение файлов требует задачи и
            аренды. Завершённый шаг процесса сам по себе не подтверждает
            результат.
          </p>
          <p>
            Чтобы выполнить поручение с созданием задач, разработкой,
            независимым ревью и проверкой, отправьте его команде.
          </p>
          <button
            className="button"
            disabled={page.project?.can_write === false || scope.locked}
            onClick={() => setRequirement(true)}
          >
            Выполнить исходное требование
          </button>
        </section>
      ) : null}
      <div className="detail-layout">
        <div className="screen-stack">
          <RunControls page={page} run={run} onSuccess={onSuccess} />
          <section
            className="settings-section screen-stack"
            aria-labelledby="managed-commands-title"
          >
            <h2 id="managed-commands-title">Команды и уточнения</h2>
            <p className="form-help">
              «Получено компьютером» подтверждает сохранение команды.
              «Применено» подтверждает действие агента, но не согласие с
              просьбой или выполнение задачи.
            </p>
            {detail.commands_truncated ? (
              <p className="notice">
                Показаны последние команды. Более ранние записи сохранены в
                истории Accord.
              </p>
            ) : null}
            {commands.length ? (
              <ol className="managed-commands">
                {commands.map((command) => (
                  <li key={command.id} className="managed-command">
                    <div className="managed-row">
                      <strong>
                        {commandLabels[command.kind] ?? command.kind}
                      </strong>
                      <span
                        className={
                          command.status === "failed" ||
                          command.status === "delivery_unknown"
                            ? "managed-warning"
                            : "muted"
                        }
                      >
                        {deliveryLabels[command.status] ?? command.status}
                      </span>
                      <time className="muted" dateTime={command.created_at}>
                        {date(command.created_at)}
                      </time>
                    </div>
                    {command.text ? (
                      <p className="profile-text">{command.text}</p>
                    ) : null}
                    {command.forced ? (
                      <p className="managed-warning">
                        Процесс остановлен принудительно.
                      </p>
                    ) : null}
                    {command.error ? (
                      <p className="field-error break">{command.error}</p>
                    ) : null}
                    {command.kind === "clarify" &&
                    command.status === "queued" ? (
                      <p className="form-help">
                        {["stopped", "completed"].includes(run.state)
                          ? "Сохранено. Нажмите «Продолжить», чтобы передать агенту."
                          : "Будет передано после текущего шага агента."}
                      </p>
                    ) : null}
                    {command.status === "delivery_unknown" ? (
                      <p className="form-help">
                        Результат требует проверки на компьютере.
                        Автоматического повторного запуска нет.
                      </p>
                    ) : null}
                  </li>
                ))}
              </ol>
            ) : !detail.commands_truncated ? (
              <p className="muted">Команд пока нет.</p>
            ) : null}
          </section>
          <section
            className="settings-section screen-stack"
            aria-labelledby="managed-output-title"
          >
            <h2 id="managed-output-title">Последние сообщения агента</h2>
            <p className="form-help">
              Краткие сообщения. Полная история остаётся на компьютере
              владельца.
            </p>
            {run.output?.length ? (
              <div className="managed-output">
                {run.output.map((line, index) => (
                  <p key={index}>{line}</p>
                ))}
              </div>
            ) : (
              <p className="muted">Агент ещё не передал сообщений.</p>
            )}
            {run.output_truncated ? (
              <p className="notice">
                Показана последняя часть сообщений. Более ранний текст сокращён.
              </p>
            ) : null}
          </section>
        </div>
        <aside className="properties-panel">
          <h3>Контекст работы</h3>
          <dl>
            <Fact name="Компьютер" value={host?.name} />
            <Fact name="Рабочая папка" value={run.workspace} />
            <Fact name="Агент" value={providerName(run.provider)} />
            <Fact
              name="Профиль"
              value={
                run.profile_snapshot
                  ? `${run.profile_snapshot.name} · версия ${run.profile_snapshot.version}`
                  : "Без профиля"
              }
            />
            <Fact name="Создана" value={date(run.created_at)} />
          </dl>
          <h3>Первоначальное задание</h3>
          <p className="profile-text">{run.prompt}</p>
          <details className="managed-identifiers">
            <summary>Сведения о диалоге</summary>
            <dl>
              <Fact
                name="Диалог агента"
                value={
                  <span className="mono break">
                    {run.provider_session_id || "Ещё не подтверждён"}
                  </span>
                }
              />
              <Fact
                name="ID работы"
                value={<span className="mono break">{run.id}</span>}
              />
            </dl>
            <p className="form-help">
              Диалог агента и рабочая сессия Accord — разные сущности. Право на
              задачу отдельно подтверждается её арендой.
            </p>
          </details>
        </aside>
      </div>
      {requirement ? (
        <RequirementDialog
          key={scope.key}
          prompt={run.prompt}
          onClose={() => setRequirement(false)}
          onSuccess={onSuccess}
        />
      ) : null}
    </div>
  );
}
