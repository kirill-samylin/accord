import { afterEach, test } from "node:test";
import assert from "node:assert/strict";
import {
  LatestHostScopeRead,
  hostScopeResult,
} from "./automation-host-scope.ts";
import { readPage } from "./api.ts";
import { appToWebRoute } from "./project-routes.ts";
import type { PageData } from "./types.ts";
const project = `p_${"a".repeat(32)}`,
  other = `p_${"b".repeat(32)}`;
function page(hostID = "mh_one", workspace = "main", version = 2): PageData {
  return {
    project: { id: project, can_write: true },
    can_continue: false,
    managed: {
      observed_at: "2026-10-23T12:00:00Z",
      hosts: [
        {
          id: hostID,
          state: "active",
          can_control: true,
          workspaces: [workspace],
          worker_capabilities: { automation: 1 },
          automation: {
            enabled: true,
            version,
            expires_at: "2026-10-24T12:00:00Z",
            scopes: {
              [workspace]: {
                version,
                expires_at: "2026-10-24T12:00:00Z",
                roles: {},
                checks: [],
              },
            },
            scopes_truncated: false,
          },
        },
      ],
      runs: [],
    },
  } as unknown as PageData;
}
function deferred<T>() {
  let resolve!: (value: T) => void, reject!: (error: unknown) => void;
  const promise = new Promise<T>((yes, no) => {
    resolve = yes;
    reject = no;
  });
  return { resolve, reject, promise };
}
const original = globalThis.fetch;
afterEach(() => {
  globalThis.fetch = original;
});
test("changing the selected workspace aborts and fences late successful responses", async () => {
  const reader = new LatestHostScopeRead(),
    old = deferred<PageData>(),
    fresh = deferred<PageData>();
  let oldSignal: AbortSignal | undefined;
  const first = reader.read(
    (signal) => {
      oldSignal = signal;
      return old.promise;
    },
    "mh_one",
    "main",
  );
  const next = reader.read(() => fresh.promise, "mh_one", "other");
  assert.equal(oldSignal?.aborted, true);
  fresh.resolve(page("mh_one", "other", 3));
  const selected = await next;
  assert.equal(selected?.host.automation?.version, 3);
  assert.ok(selected?.host.automation?.scopes.other);
  old.resolve(page());
  assert.equal(await first, undefined);
});
test("selection and project cleanup suppress stale failures and never adopt previous consent", async () => {
  const reader = new LatestHostScopeRead(),
    old = deferred<PageData>();
  const first = reader.read(() => old.promise, "mh_one", "main");
  reader.cancel();
  old.reject(new Error("old project permission"));
  assert.equal(await first, undefined);
  const current = await reader.read(
    async () => page("mh_two", "new", 4),
    "mh_two",
    "new",
  );
  assert.equal(current?.host.id, "mh_two");
  await assert.rejects(
    reader.read(
      async () => {
        throw new Error("current permission revoked");
      },
      "mh_two",
      "new",
    ),
    /current permission revoked/,
  );
});
test("scope reads require exact full consent and current control, never a list summary", () => {
  assert.equal(
    hostScopeResult(page(), "mh_one", "main").host.automation?.version,
    2,
  );
  const summary = page();
  summary.managed!.hosts[0].automation!.scopes_truncated = true;
  summary.managed!.hosts[0].automation!.scopes = {};
  assert.throws(
    () => hostScopeResult(summary, "mh_one", "main"),
    /полное локальное согласие/,
  );
  assert.throws(
    () => hostScopeResult(page(), "mh_one", "other"),
    /другой папке/,
  );
  const denied = page();
  denied.managed!.hosts[0].can_control = false;
  assert.throws(() => hostScopeResult(denied, "mh_one", "main"), /не разрешил/);
  const viewer = page();
  viewer.project!.can_write = false;
  assert.throws(() => hostScopeResult(viewer, "mh_one", "main"), /не разрешил/);
  const expired = page();
  expired.managed!.hosts[0].automation!.scopes.main.expires_at =
    "2026-10-23T11:59:59Z";
  assert.throws(() => hostScopeResult(expired, "mh_one", "main"), /истекло/);
  const invalid = page();
  invalid.managed!.hosts[0].automation!.expires_at = "invalid";
  assert.throws(() => hostScopeResult(invalid, "mh_one", "main"), /истекло/);
});
test("host scope read routes retain project and reject a response from another project", async () => {
  const path = `/app/p/${project}/automation/host-scope?host_id=mh_one&workspace=main`;
  assert.equal(
    appToWebRoute(path),
    `/web/p/${project}/automation/host-scope?host_id=mh_one&workspace=main`,
  );
  const wrong = page();
  wrong.project!.id = other;
  globalThis.fetch = async () =>
    new Response(JSON.stringify({ ok: true, page: wrong }), { status: 200 });
  await assert.rejects(readPage(path), /другому проекту/);
});
