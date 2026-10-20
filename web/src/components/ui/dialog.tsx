"use client";
import * as React from "react";
import { Dialog as Primitive } from "radix-ui";
import { X } from "lucide-react";
import { cn } from "@/lib/utils";
import { useDialogFocusHandlers } from "./dialog-focus";
export const Dialog = Primitive.Root;
export const DialogTrigger = Primitive.Trigger;
export const DialogClose = Primitive.Close;
export function DialogContent({
  className,
  children,
  showCloseButton = true,
  onOpenAutoFocus,
  onCloseAutoFocus,
  ...props
}: React.ComponentProps<typeof Primitive.Content> & {
  showCloseButton?: boolean;
}) {
  const focus = useDialogFocusHandlers({ onOpenAutoFocus, onCloseAutoFocus });
  return (
    <Primitive.Portal>
      <Primitive.Overlay className="dialog-overlay" />
      <Primitive.Content
        className={cn("dialog-content", className)}
        {...props}
        {...focus}
      >
        {children}
        {showCloseButton ? (
          <Primitive.Close
            className="dialog-close icon-button"
            aria-label="Закрыть"
          >
            <X size={16} />
          </Primitive.Close>
        ) : null}
      </Primitive.Content>
    </Primitive.Portal>
  );
}
export function DialogHeader({
  className,
  ...props
}: React.ComponentProps<"div">) {
  return <div className={cn("dialog-header", className)} {...props} />;
}
export function DialogFooter({
  className,
  ...props
}: React.ComponentProps<"div">) {
  return <div className={cn("form-actions", className)} {...props} />;
}
export function DialogTitle(
  props: React.ComponentProps<typeof Primitive.Title>,
) {
  return <Primitive.Title className="dialog-title" {...props} />;
}
export function DialogDescription(
  props: React.ComponentProps<typeof Primitive.Description>,
) {
  return <Primitive.Description className="muted text-sm" {...props} />;
}
