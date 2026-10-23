import type { ManagedHost } from "./managed-agents.ts";
import type { PageData } from "./types.ts";
export interface HostScopeResult {
  host: ManagedHost;
  observedAt: string;
}
export function hostScopeResult(
  page: PageData,
  hostID: string,
  workspace: string,
): HostScopeResult {
  const hosts = page.managed?.hosts ?? [],
    host = hosts[0],
    consent = host?.automation;
  if (page.can_continue)
    throw new Error("Сессия истекла. Продолжите работу перед выбором папки.");
  if (
    hosts.length !== 1 ||
    host.id !== hostID ||
    !host.workspaces.includes(workspace)
  )
    throw new Error(
      "Ответ относится к другой папке. Выберите компьютер и папку заново.",
    );
  if (
    page.project?.can_write === false ||
    !host.can_control ||
    host.state !== "active"
  )
    throw new Error(
      "Владелец компьютера не разрешил вам управление этой папкой.",
    );
  if (
    !consent?.enabled ||
    !Number.isSafeInteger(consent.version) ||
    consent.version < 1 ||
    consent.scopes_truncated ||
    host.worker_capabilities?.automation !== 1 ||
    Object.keys(consent.scopes ?? {}).length !== 1 ||
    !consent.scopes[workspace]
  )
    throw new Error(
      "Для этой папки не получено полное локальное согласие. Владелец должен разрешить автоматизацию.",
    );
  const observed = Date.parse(page.managed?.observed_at ?? ""),
    scope = consent.scopes[workspace];
  if (
    !Number.isFinite(observed) ||
    !Number.isFinite(Date.parse(consent.expires_at)) ||
    !Number.isFinite(Date.parse(scope.expires_at)) ||
    Date.parse(consent.expires_at) <= observed ||
    Date.parse(scope.expires_at) <= observed
  )
    throw new Error(
      "Локальное согласие для этой папки истекло. Владелец должен обновить разрешение.",
    );
  return { host, observedAt: page.managed!.observed_at };
}
// A late response can neither replace a newer selection nor surface its error.
// Abort saves transport work; the sequence fence also handles readers ignoring it.
export class LatestHostScopeRead {
  private sequence = 0;
  private controller?: AbortController;
  cancel() {
    this.sequence++;
    this.controller?.abort();
    this.controller = undefined;
  }
  async read(
    loader: (signal: AbortSignal) => Promise<PageData>,
    hostID: string,
    workspace: string,
  ): Promise<HostScopeResult | undefined> {
    this.cancel();
    const sequence = this.sequence,
      controller = new AbortController();
    this.controller = controller;
    try {
      const page = await loader(controller.signal);
      if (sequence !== this.sequence || controller.signal.aborted) return;
      return hostScopeResult(page, hostID, workspace);
    } catch (error) {
      if (sequence !== this.sequence || controller.signal.aborted) return;
      throw error;
    } finally {
      if (sequence === this.sequence) this.controller = undefined;
    }
  }
}
