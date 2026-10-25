"use client";
import { useQuery } from "@tanstack/react-query";
import { readPage } from "@/lib/api";
import { isCatalogPage, type CatalogPage } from "@/lib/projects";
import type { PageData } from "@/lib/types";
import { useProjectScope } from "../projects/project-context";
// Whether the signed-in identity is a global administrator. The projects
// catalog carries registry.identity.admin on every catalog page; this shares
// the project switcher's query so the shell never issues a second request
// for the same scope.
export function useAdminAccess(): boolean {
  const scope = useProjectScope();
  const catalog = useQuery({
    queryKey: ["project-catalog", scope.key],
    queryFn: ({ signal }) =>
      readPage<PageData | CatalogPage>("/app/projects", signal, scope.key),
    retry: false,
    gcTime: 0,
    staleTime: 60_000,
  });
  return (
    isCatalogPage(catalog.data) && catalog.data.registry.identity.admin === true
  );
}
