"use client";
import { CopyX, GitMerge, Info } from "lucide-react";
import { ProjectLink as Link } from "@/components/project-link";
import type { MutationResult, PageData, Task } from "@/lib/types";
import { actionBody, actionID } from "../mutations/contracts";
import { ActionNotice } from "../mutations/ActionNotice";
import { useAction } from "../mutations/useAction";
import { SimilarItem } from "./similar-list";

export function DuplicateBanners({ task }: { task: Task }) {
  return (
    <>
      {task.duplicate_of ? (
        <div className="duplicate-banner" role="note">
          <CopyX size={16} aria-hidden />
          <span>
            Эта задача отмечена дублем{" "}
            <Link prefetch={false} href={`/app/tasks/${task.duplicate_of}`}>
              {task.duplicate_of}
            </Link>
            . Агенты её не возьмут; работа ведётся в основной задаче.
          </span>
        </div>
      ) : null}
      {task.duplicate_override ? (
        <div className="duplicate-banner" role="note">
          <Info size={16} aria-hidden />
          <span>
            Создана, несмотря на похожие:{" "}
            {task.duplicate_override.similar.map((key, i) => (
              <span key={key}>
                {i ? ", " : ""}
                <Link prefetch={false} href={`/app/tasks/${key}`}>
                  {key}
                </Link>
              </span>
            ))}
            . Причина: {task.duplicate_override.reason}
          </span>
        </div>
      ) : null}
    </>
  );
}

// SimilarTasks lists live similar work for this task and lets a human or
// coordinator close this task as a duplicate of one of them.
export function SimilarTasks({
  page,
  onSuccess,
  readOnly = false,
}: {
  page: PageData;
  onSuccess: (result: MutationResult) => void;
  readOnly?: boolean;
}) {
  const action = useAction(page, onSuccess);
  const task = page.task!;
  const items = page.similar ?? [];
  if (!items.length || task.state === "cancelled") return null;
  const free =
    !task.lease &&
    (task.state === "ready" ||
      task.state === "created" ||
      task.state === "open");
  const permitted =
    !readOnly &&
    free &&
    page.can_coordinate === true &&
    !page.can_continue &&
    page.project?.can_write !== false &&
    Boolean(actionID(action.page, "coord:duplicate"));
  function close(canonical: string) {
    void action.send(
      `/web/tasks/${task.key}/duplicate`,
      actionBody(action.page, "coord:duplicate", {
        canonical_key: canonical,
        version: String(task.version),
        reason: `Дубль ${canonical}: отмечено в карточке задачи`,
      }),
    );
  }
  return (
    <section className="similar-callout" aria-labelledby="similar-heading">
      <div className="similar-callout-head">
        <CopyX size={15} aria-hidden />
        <span id="similar-heading">Похожие задачи</span>
      </div>
      <p className="muted">
        Перед началом работы проверьте, не делает ли это уже кто-то другой.
      </p>
      <ActionNotice
        message={action.message}
        uncertain={action.phase === "uncertain"}
        busy={action.busy}
        onRetry={() => void action.retry()}
      />
      <div className="similar-list">
        {items.map((item) => (
          <div className="similar-row" key={item.key}>
            <SimilarItem item={item} />
            {permitted && item.state !== "cancelled" ? (
              <button
                className="button button-secondary button-sm"
                disabled={action.locked}
                onClick={() => close(item.key)}
                title={`${task.key} будет отменена со ссылкой на ${item.key}`}
              >
                <GitMerge size={14} aria-hidden />
                Закрыть как дубль {item.key}
              </button>
            ) : null}
          </div>
        ))}
      </div>
    </section>
  );
}
