"use client";
import { ProjectLink as Link } from "@/components/project-link";
export function AgentNavigation({
  current,
}: {
  current: "profiles" | "sessions";
}) {
  return (
    <nav className="agent-navigation" aria-label="Разделы агентов">
      <Link
        prefetch={false}
        href="/app/profiles"
        aria-current={current === "profiles" ? "page" : undefined}
      >
        Профили
      </Link>
      <Link
        prefetch={false}
        href="/app/agents"
        aria-current={current === "sessions" ? "page" : undefined}
      >
        Сессии
      </Link>
    </nav>
  );
}
