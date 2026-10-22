"use client";
import { CopyX, GitMerge, Split } from "lucide-react";
import { ProjectLink as Link } from "@/components/project-link";
import { PersonChip } from "@/components/people";
import { Status } from "@/components/common";
import type {
  DuplicatePair,
  MutationResult,
  PageData,
  SimilarTask,
} from "@/lib/types";
import { reasonLabels } from "@/lib/people";
import { actionBody, actionID } from "../mutations/contracts";
import { ActionNotice } from "../mutations/ActionNotice";
import { useAction } from "../mutations/useAction";
import { ScoreBadge } from "./similar-list";

// free work can be cancelled as a duplicate; leased or finished work cannot.
const free = (t: SimilarTask) =>
  !t.live_lease &&
  (t.state === "ready" || t.state === "created" || t.state === "open");

// resolution picks the item to keep: the one already in progress or finished,
// otherwise the older one. Returns null when neither may be closed.
export function resolution(pair: DuplicatePair) {
  const { older, newer } = pair;
  if (free(newer)) return { keep: older, close: newer };
  if (free(older)) return { keep: newer, close: older };
  return null;
}

function Side({ task }: { task: SimilarTask }) {
  return (
    <div className="pair-side">
      <div className="pair-side-head">
        <Link
          prefetch={false}
          className="mono task-key"
          href={`/app/tasks/${task.key}`}
        >
          {task.key}
        </Link>
        <Status value={task.state} />
      </div>
      <Link
        prefetch={false}
        className="pair-title"
        href={`/app/tasks/${task.key}`}
        title={task.title}
      >
        {task.title}
      </Link>
      <div className="pair-people">
        {task.created_by ? (
          <PersonChip owner={task.created_by} prefix="создал" />
        ) : null}
        {task.holder_owner && task.live_lease ? (
          <PersonChip owner={task.holder_owner} prefix="делает" />
        ) : null}
      </div>
    </div>
  );
}

export function DuplicatePairs({
  pairs,
  total,
  page,
  onSuccess,
}: {
  pairs: DuplicatePair[];
  total: number;
  page: PageData;
  onSuccess: (result: MutationResult) => void;
}) {
  // Stay on the team screen after resolving a pair; the page and live team
  // data are refreshed by the shared success handler.
  const action = useAction(page, (result) =>
    onSuccess({ ...result, redirect: undefined }),
  );
  const permitted =
    page.can_coordinate === true &&
    !page.can_continue &&
    page.project?.can_write !== false;
  function markDuplicate(keep: SimilarTask, close: SimilarTask) {
    void action.send(
      `/web/tasks/${close.key}/duplicate`,
      actionBody(
        action.page,
        "coord:duplicate",
        {
          canonical_key: keep.key,
          reason: `Дубль ${keep.key}: отмечено на экране «Команда»`,
        },
        close.key,
      ),
    );
  }
  function dismiss(pair: DuplicatePair) {
    // The link is stored on an unleased item; a live lease belongs to its holder.
    const target = free(pair.newer) ? pair.newer : pair.older;
    const from = target === pair.newer ? pair.older : pair.newer;
    void action.send(
      `/web/tasks/${target.key}/link`,
      actionBody(
        action.page,
        "coord:link",
        { from: from.key, type: "related", generation: "0" },
        target.key,
      ),
    );
  }
  return (
    <section className="dup-panel" aria-labelledby="dup-heading">
      <div className="dup-panel-head">
        <CopyX size={16} aria-hidden />
        <div>
          <h2 id="dup-heading">Возможные дубли</h2>
          <p className="muted">
            Похожие незавершённые задачи. Оставьте одну, чтобы агенты не делали
            работу дважды, или отметьте, что это разные задачи.
          </p>
        </div>
        <span className="dup-count">{total}</span>
      </div>
      <ActionNotice
        message={action.message}
        uncertain={action.phase === "uncertain"}
        busy={action.busy}
        onRetry={() => void action.retry()}
      />
      <div className="dup-list">
        {pairs.map((pair) => {
          const plan = resolution(pair);
          const canMark =
            permitted &&
            plan &&
            actionID(action.page, "coord:duplicate", plan.close.key);
          const linkTarget = free(pair.newer)
            ? pair.newer
            : free(pair.older)
              ? pair.older
              : null;
          const canDismiss =
            permitted &&
            linkTarget &&
            actionID(action.page, "coord:link", linkTarget.key);
          return (
            <article
              className="dup-row"
              key={`${pair.older.key}-${pair.newer.key}`}
            >
              <div className="dup-score">
                <ScoreBadge score={pair.score} />
                <span className="muted">
                  {pair.reasons.map((r) => reasonLabels[r] ?? r).join(", ")}
                </span>
              </div>
              <div className="dup-sides">
                <Side task={pair.older} />
                <span className="dup-equals" aria-label="похожа на">
                  ≈
                </span>
                <Side task={pair.newer} />
              </div>
              <div className="dup-actions">
                {plan ? (
                  <button
                    className="button button-sm"
                    disabled={!canMark || action.locked}
                    onClick={() => markDuplicate(plan.keep, plan.close)}
                    title={`${plan.close.key} будет отменена со ссылкой на ${plan.keep.key}`}
                  >
                    <GitMerge size={14} aria-hidden />
                    Оставить {plan.keep.key}
                  </button>
                ) : (
                  <span className="muted dup-hint">
                    Обе задачи уже в работе — договоритесь, кто продолжит
                  </span>
                )}
                {linkTarget ? (
                  <button
                    className="button button-secondary button-sm"
                    disabled={!canDismiss || action.locked}
                    onClick={() => dismiss(pair)}
                  >
                    <Split size={14} aria-hidden />
                    Это разные
                  </button>
                ) : null}
              </div>
            </article>
          );
        })}
      </div>
      {total > pairs.length ? (
        <p className="muted dup-more">
          Показаны {pairs.length} из {total}. Остальные появятся после разбора.
        </p>
      ) : null}
    </section>
  );
}
