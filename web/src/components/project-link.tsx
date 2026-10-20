"use client";
import Link from "next/link";
import type { ComponentProps } from "react";
import { useProjectScope } from "@/features/projects/project-context";
export function ProjectLink({
  href,
  onClick,
  ...props
}: Omit<ComponentProps<typeof Link>, "href"> & { href: string }) {
  const scope = useProjectScope();
  return (
    <Link
      {...props}
      prefetch={false}
      href={scope.app(href)}
      onClick={(event) => {
        if (scope.locked) {
          event.preventDefault();
          return;
        }
        onClick?.(event);
      }}
    />
  );
}
