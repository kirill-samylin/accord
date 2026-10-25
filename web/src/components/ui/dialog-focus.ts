"use client";
import { useRef } from "react";
// "content": move focus to the dialog/popover container itself (default).
// Radix otherwise focuses the first field, and a focused text field always
// shows the focus ring, so every window opened with a ring on its first input.
// Keyboard users still land inside the trapped scope and Tab to the first
// control. "first" keeps the Radix behaviour for surfaces whose purpose is to
// type immediately (the command palette).
export type InitialFocus = "content" | "first";
type FocusHandlers = {
  onOpenAutoFocus?: (event: Event) => void;
  onCloseAutoFocus?: (event: Event) => void;
  initialFocus?: InitialFocus;
  returnFocus?: { current: HTMLElement | null };
};
function available(element: Element | null): element is HTMLElement {
  return (
    element instanceof HTMLElement &&
    element !== document.body &&
    element.isConnected &&
    element.getClientRects().length > 0 &&
    !element.closest("[inert]") &&
    !element.matches(":disabled") &&
    getComputedStyle(element).visibility !== "hidden"
  );
}
// Return focus to the opener; when it is gone (a menu item, a submit button
// replaced by the result), use an explicit target, then the page heading.
export function restoreDialogFocus(
  opener: Element | null,
  fallback?: HTMLElement | null,
) {
  const heading = document.querySelector<HTMLElement>("#main-content h1");
  if (heading && !heading.hasAttribute("tabindex")) heading.tabIndex = -1;
  const target =
    [opener, fallback ?? null, heading].find(available) ??
    Array.from(
      document.querySelectorAll<HTMLElement>("[data-command-trigger]"),
    ).find(available);
  target?.focus({ preventScroll: true });
}
// Radix dispatches the mount-autofocus event on the focus-scope container,
// which already carries tabIndex=-1.
export function focusContainer(event: Event) {
  event.preventDefault();
  const container = event.currentTarget ?? event.target;
  if (container instanceof HTMLElement)
    container.focus({ preventScroll: true });
}
export function useDialogFocusHandlers(handlers: FocusHandlers = {}) {
  const opener = useRef<Element | null>(null);
  return {
    onOpenAutoFocus(event: Event) {
      // Controlled dialogs have no Radix Trigger. Capture before framework
      // autofocus, including the field from which a keyboard shortcut opened.
      opener.current = document.activeElement;
      handlers.onOpenAutoFocus?.(event);
      if (event.defaultPrevented || handlers.initialFocus === "first") return;
      focusContainer(event);
    },
    onCloseAutoFocus(event: Event) {
      handlers.onCloseAutoFocus?.(event);
      if (event.defaultPrevented) return;
      event.preventDefault();
      restoreDialogFocus(opener.current, handlers.returnFocus?.current);
    },
  };
}
