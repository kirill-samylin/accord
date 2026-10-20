"use client";
import { DropdownMenu as Primitive } from "radix-ui";
import type { ComponentProps } from "react";
import { cn } from "@/lib/utils";
export const DropdownMenu = Primitive.Root;
export const DropdownMenuTrigger = Primitive.Trigger;
export function DropdownMenuContent({
  className,
  ...props
}: ComponentProps<typeof Primitive.Content>) {
  return (
    <Primitive.Portal>
      <Primitive.Content
        className={cn("popover menu-content", className)}
        sideOffset={5}
        {...props}
      />
    </Primitive.Portal>
  );
}
export function DropdownMenuItem(props: ComponentProps<typeof Primitive.Item>) {
  return <Primitive.Item className="menu-item" {...props} />;
}
export const DropdownMenuSeparator = Primitive.Separator;
