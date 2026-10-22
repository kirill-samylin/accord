import { useEffect, useState } from "react";
import { Bot } from "lucide-react";
import { initial, label } from "@/lib/format";
import { personClass } from "@/lib/people";
import { cn } from "@/lib/utils";

// Re-render relative times ("12 мин") without refetching data.
export function useNow(intervalMs = 30000) {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const id = window.setInterval(() => setNow(Date.now()), intervalMs);
    return () => window.clearInterval(id);
  }, [intervalMs]);
  return now;
}

export function PersonAvatar({
  owner,
  online,
  size = "md",
}: {
  owner: string;
  online?: boolean;
  size?: "sm" | "md" | "lg";
}) {
  return (
    <span
      aria-hidden
      className={cn(
        "person-avatar",
        `person-avatar-${size}`,
        personClass(owner),
      )}
    >
      {initial(owner)}
      {online !== undefined ? (
        <span className={cn("presence", online && "presence-on")} />
      ) : null}
    </span>
  );
}

export function PersonChip({
  owner,
  prefix,
  title,
}: {
  owner?: string;
  prefix?: string;
  title?: string;
}) {
  if (!owner) return null;
  return (
    <span className={cn("person-chip", personClass(owner))} title={title}>
      <span className="person-chip-dot" aria-hidden />
      {prefix ? <span className="person-chip-prefix">{prefix}</span> : null}
      <span className="truncate">{owner}</span>
    </span>
  );
}

export function RoleBadge({ role, name }: { role: string; name?: string }) {
  return (
    <span className={cn("role-badge", `agent-${role}`)}>
      <Bot size={12} aria-hidden />
      <span className="truncate">{name || label(role)}</span>
    </span>
  );
}

export function LivePill({
  paused,
  failed,
  fetching,
}: {
  paused: boolean;
  failed: boolean;
  fetching: boolean;
}) {
  const state = failed ? "failed" : paused ? "paused" : "live";
  return (
    <span className={cn("live-pill", `live-${state}`)} role="status">
      <span className={cn("live-dot", fetching && "live-dot-pulse")} />
      <span className="live-text">
        {failed
          ? "Нет связи"
          : paused
            ? "Обновление на паузе"
            : "Обновляется автоматически"}
      </span>
    </span>
  );
}
