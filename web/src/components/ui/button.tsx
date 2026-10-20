import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { Slot } from "radix-ui";
import { cn } from "@/lib/utils";
const buttonVariants = cva("button", {
  variants: {
    variant: {
      default: "",
      outline: "button-secondary",
      secondary: "button-secondary",
      ghost: "button-ghost",
      destructive: "button-destructive",
      link: "button-link",
    },
    size: {
      default: "",
      sm: "button-sm",
      icon: "icon-button",
      "icon-sm": "icon-button",
      xs: "button-xs",
    },
  },
  defaultVariants: { variant: "default", size: "default" },
});
export const Button = React.forwardRef<
  HTMLButtonElement,
  React.ComponentProps<"button"> &
    VariantProps<typeof buttonVariants> & { asChild?: boolean }
>(function Button({ className, variant, size, asChild, ...props }, ref) {
  const Component = asChild ? Slot.Root : "button";
  return (
    <Component
      ref={ref}
      className={cn(buttonVariants({ variant, size }), className)}
      {...props}
    />
  );
});
export { buttonVariants };
