"use client";
import { useRouter } from "next/navigation";
import { useQueryClient } from "@tanstack/react-query";
import { webToApp, appToWeb } from "@/lib/api";
import type { CatalogPage, Project } from "@/lib/projects";
import { RegistryForm } from "./registry-form";
export function OpenProject({
  page,
  project,
  target,
  title = "Открыть проект",
}: {
  page: CatalogPage;
  project: Project;
  target?: string;
  title?: string;
}) {
  const router = useRouter();
  const client = useQueryClient();
  return (
    <RegistryForm
      page={page}
      actionKey={`switch:${project.id}`}
      endpoint="/web/catalog/switch"
      refreshPath="/app/projects"
      title={title}
      fields={{
        project_id: project.id,
        ...(target ? { target: appToWeb(target) } : {}),
      }}
      onSuccess={(result) => {
        void client.invalidateQueries();
        if (result.redirect) router.replace(webToApp(result.redirect));
      }}
    />
  );
}
