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
export function TabsTrigger(props: ComponentProps<typeof Primitive.Trigger>) {
  return <Primitive.Trigger className="tab" {...props} />;
}
export function TabsContent({
  className,
  ...props
}: ComponentProps<typeof Primitive.Content>) {
  return (
    <Primitive.Content className={cn("tab-content", className)} {...props} />
  );
}
