"use client";
import { useRef } from "react";
type FocusHandlers = {
  onOpenAutoFocus?: (event: Event) => void;
  onCloseAutoFocus?: (event: Event) => void;
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
export function restoreDialogFocus(opener: Element | null) {
  const target = available(opener)
    ? opener
    : Array.from(
        document.querySelectorAll<HTMLElement>("[data-command-trigger]"),
      ).find(available);
  target?.focus({ preventScroll: true });
}
export function useDialogFocusHandlers(handlers: FocusHandlers = {}) {
  const opener = useRef<Element | null>(null);
  return {
    onOpenAutoFocus(event: Event) {
      // Controlled dialogs have no Radix Trigger. Capture before framework
      // autofocus, including the field from which a keyboard shortcut opened.
      opener.current = document.activeElement;
      handlers.onOpenAutoFocus?.(event);
    },
    onCloseAutoFocus(event: Event) {
      handlers.onCloseAutoFocus?.(event);
      if (event.defaultPrevented) return;
      event.preventDefault();
      restoreDialogFocus(opener.current);
    },
  };
}
