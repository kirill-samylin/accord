"use client";
import { Popover as Primitive } from "radix-ui";
import type { ComponentProps } from "react";
import { focusContainer } from "./dialog-focus";
export const Popover = Primitive.Root;
export const PopoverTrigger = Primitive.Trigger;
// Like dialogs, a popover takes focus on its container rather than on the
// first field, so opening it never paints a focus ring on that field.
export function PopoverContent({
  onOpenAutoFocus,
  ...props
}: ComponentProps<typeof Primitive.Content>) {
  return (
    <Primitive.Portal>
      <Primitive.Content
        className="popover filter-popover"
        sideOffset={8}
        align="start"
        onOpenAutoFocus={(event) => {
          onOpenAutoFocus?.(event);
          if (!event.defaultPrevented) focusContainer(event);
        }}
        {...props}
      />
    </Primitive.Portal>
  );
}
