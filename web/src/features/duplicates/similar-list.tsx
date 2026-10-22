"use client";
import { ProjectLink as Link } from "@/components/project-link";
import { PersonChip } from "@/components/people";
import { Status } from "@/components/common";
import { percent, reasonLabels } from "@/lib/people";
import type { SimilarTask } from "@/lib/types";
import { cn } from "@/lib/utils";

export function ScoreBadge({ score }: { score: number }) {
  const level = score >= 0.8 ? "high" : score >= 0.6 ? "mid" : "low";
  return (
    <span
      className={cn("score-badge", `score-${level}`)}
      title="Насколько похожи формулировки"
    >
      {percent(score)}
    </span>
  );
}

export function SimilarItem({
  item,
  compact = false,
}: {
  item: SimilarTask;
  compact?: boolean;
}) {
  return (
    <div className={cn("similar-item", compact && "similar-item-compact")}>
      <ScoreBadge score={item.score} />
      <div className="similar-main">
        <div className="similar-title">
          <Link
            prefetch={false}
            className="mono task-key"
            href={`/app/tasks/${item.key}`}
            target={compact ? "_blank" : undefined}
          >
            {item.key}
          </Link>
          <Link
            prefetch={false}
            className="truncate"
            href={`/app/tasks/${item.key}`}
            target={compact ? "_blank" : undefined}
            title={item.title}
          >
            {item.title}
          </Link>
        </div>
        <div className="similar-meta">
          <Status value={item.state} />
          {item.holder_owner && item.live_lease ? (
            <PersonChip owner={item.holder_owner} prefix="делает" />
          ) : item.created_by ? (
            <PersonChip owner={item.created_by} prefix="создал" />
          ) : null}
          {item.finished_recent ? (
            <span className="muted">уже сделано недавно</span>
          ) : null}
          {item.reasons.length ? (
            <span className="muted truncate">
              совпадает:{" "}
              {item.reasons.map((r) => reasonLabels[r] ?? r).join(", ")}
            </span>
          ) : null}
        </div>
      </div>
    </div>
  );
}

export function SimilarList({
  items,
  compact = false,
}: {
  items: SimilarTask[];
  compact?: boolean;
}) {
  return (
    <div className="similar-list">
      {items.map((item) => (
        <SimilarItem key={item.key} item={item} compact={compact} />
      ))}
    </div>
  );
}
