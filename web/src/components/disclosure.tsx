"use client";
// Disclosure replaces native <details>/<summary>: the panel animates open
// and closed (styles/disclosure.css) and its content stays mounted, so forms
// inside keep their state across toggles. While closed the panel is `inert`,
// which removes it from the tab order and the accessibility tree.
import { ChevronDown } from "lucide-react";
import { useId, useState, type ReactNode } from "react";
import { cn } from "@/lib/utils";

export function Disclosure({
  summary,
  children,
  defaultOpen = false,
  open,
  onOpenChange,
  className,
  id,
}: {
  // Trigger label; typography follows the block it sits in.
  summary: ReactNode;
  children: ReactNode;
  // Uncontrolled initial state. Pass `open` (+ onOpenChange) to control it.
  defaultOpen?: boolean;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  // Extra classes for the root; existing block classes (managed-setup,
  // registry-add, automation-card…) keep their layout through them.
  className?: string;
  id?: string;
}) {
  const [own, setOwn] = useState(defaultOpen);
  const isOpen = open ?? own;
  const generated = useId();
  const panelId = `${id ?? generated}-panel`;
  function toggle() {
    const next = !isOpen;
    if (open === undefined) setOwn(next);
    onOpenChange?.(next);
  }
  return (
    <div
      className={cn("disclosure", className)}
      data-open={isOpen ? "" : undefined}
      id={id}
    >
      <button
        type="button"
        className="disclosure-trigger"
        aria-expanded={isOpen}
        aria-controls={panelId}
        onClick={toggle}
      >
        <span className="disclosure-summary">{summary}</span>
        <ChevronDown size={15} aria-hidden className="disclosure-chevron" />
      </button>
      <div
        className="disclosure-panel"
        id={panelId}
        inert={isOpen ? undefined : true}
      >
        <div className="disclosure-inner">{children}</div>
      </div>
    </div>
  );
}
