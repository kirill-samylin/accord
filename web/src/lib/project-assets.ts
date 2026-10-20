import { projectID } from "./projects.ts";
export function safeAssetURL(
  url: string,
  kind: "download" | "preview",
  expectedProject?: string,
) {
  if (expectedProject !== undefined && !projectID.test(expectedProject))
    return undefined;
  const prefix =
    expectedProject && projectID.test(expectedProject)
      ? `/p/${expectedProject}`
      : "";
  return new RegExp(`^/web${prefix}/designs/[A-Za-z0-9_-]+/${kind}$`).test(url)
    ? url
    : undefined;
}
