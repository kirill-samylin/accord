"use client";
import { Tabs as Primitive } from "radix-ui";
import type { ComponentProps } from "react";
import { cn } from "@/lib/utils";
export const Tabs = Primitive.Root;
export function TabsList({
  className,
  ...props
}: ComponentProps<typeof Primitive.List>) {
  return <Primitive.List className={cn("tabs", className)} {...props} />;
}
// Radix moves focus to the activated trigger (roving focus), and browsers may
// treat that as :focus-visible even after a mouse click. The trigger records
// pointer activation in data-pointer-focus (cleared by any key or on blur) so
// CSS shows the ring for keyboard focus only. Handlers never preventDefault:
// Radix skips its own activation when an earlier handler does.
export function TabsTrigger({
  onPointerDown,
  onKeyDown,
  onBlur,
  ...props
}: ComponentProps<typeof Primitive.Trigger>) {
  return (
    <Primitive.Trigger
      className="tab"
      {...props}
      onPointerDown={(event) => {
        event.currentTarget.dataset.pointerFocus = "";
        onPointerDown?.(event);
      }}
      onKeyDown={(event) => {
        delete event.currentTarget.dataset.pointerFocus;
        onKeyDown?.(event);
      }}
      onBlur={(event) => {
        delete event.currentTarget.dataset.pointerFocus;
        onBlur?.(event);
      }}
    />
  );
}
export function TabsContent({
  className,
  ...props
}: ComponentProps<typeof Primitive.Content>) {
  return (
    <Primitive.Content className={cn("tab-content", className)} {...props} />
  );
}
