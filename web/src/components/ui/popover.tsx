"use client";
import { Popover as Primitive } from "radix-ui";
import type { ComponentProps } from "react";
export const Popover = Primitive.Root;
export const PopoverTrigger = Primitive.Trigger;
export function PopoverContent(
  props: ComponentProps<typeof Primitive.Content>,
) {
  return (
    <Primitive.Portal>
      <Primitive.Content
        className="popover filter-popover"
        sideOffset={8}
        align="start"
        {...props}
      />
    </Primitive.Portal>
  );
}
