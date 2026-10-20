"use client";
import { useEffect, useRef } from "react";
export function useRouteHeadingFocus(path: string, ready: boolean) {
  const readyPath = useRef<string | null>(null);
  useEffect(() => {
    if (!ready || readyPath.current === path) return;
    const heading = document.querySelector<HTMLElement>("#main-content h1");
    if (!heading) return;
    // A ready route is handled once even when focus belongs to a live field
    // or dialog. Query changes and background reads must never take it later.
    readyPath.current = path;
    if (document.activeElement !== document.body) return;
    heading.tabIndex = -1;
    heading.focus({ preventScroll: true });
  }, [path, ready]);
}
