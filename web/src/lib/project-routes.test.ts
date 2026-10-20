import { test } from "node:test";
import assert from "node:assert/strict";
import {
  appToWebRoute,
  webToAppRoute,
  projectAppPath,
  projectWebPath,
  routeScope,
  safeLoginReturn,
  screenPath,
} from "./project-routes.ts";
import { safeAssetURL } from "./project-assets.ts";
const a = "p_" + "a".repeat(32),
  b = "p_" + "b".repeat(32);
test("identical task keys keep their explicit project across page, redirect and mutation routes", () => {
  for (const id of [a, b]) {
    assert.equal(
      appToWebRoute(`/app/p/${id}/tasks/T-1?tab=details`),
      `/web/p/${id}/tasks/T-1?tab=details`,
    );
    assert.equal(
      webToAppRoute(`/web/p/${id}/tasks/T-1`),
      `/app/p/${id}/tasks/T-1`,
    );
    assert.equal(
      projectWebPath(id, "/web/tasks/T-1/answer"),
      `/web/p/${id}/tasks/T-1/answer`,
    );
    assert.equal(
      projectAppPath(id, "/app/tasks/T-1"),
      `/app/p/${id}/tasks/T-1`,
    );
    assert.equal(routeScope(`/app/p/${id}/tasks/T-1`), id);
    assert.equal(screenPath(`/app/p/${id}/tasks/T-1`), "/app/tasks/T-1");
  }
  assert.equal(appToWebRoute("/app"), "/web/");
  assert.equal(
    appToWebRoute(`/app/projects/${b}`),
    `/web/catalog/projects/${b}`,
  );
  assert.equal(projectAppPath(a, "/app/projects"), "/app/projects");
  assert.notEqual(
    routeScope(`/app/projects/${a}`),
    routeScope(`/app/projects/${b}`),
  );
  assert.throws(() => projectAppPath(a, `/app/p/${b}/tasks/T-1`));
  assert.throws(() => projectWebPath(a, `/web/p/${b}/tasks/T-1/answer`));
});
test("project routes reject traversal, noncanonical IDs, binaries, credentials and external addresses", () => {
  for (const path of [
    `/app/p/${a}/../tasks`,
    `/app/p/${a}/%2e%2e/tasks`,
    `/app/p/${a}/tasks%2fT-1`,
    "/app/p/name/tasks",
    `/app/p/${a.toUpperCase()}/tasks`,
    `https://evil.invalid/app/p/${a}`,
    `//evil.invalid/app/p/${a}`,
    `/app/p/${a}/tasks/T-1/answer`,
    `/app/p/${a}/designs/asset/download`,
    `/app/p/${a}/login`,
  ])
    assert.throws(() => appToWebRoute(path), path);
  assert.equal(
    safeLoginReturn(`/app/p/${a}/tasks/T-1`),
    `/app/p/${a}/tasks/T-1`,
  );
  assert.equal(safeLoginReturn("https://evil.invalid"), undefined);
  assert.equal(safeLoginReturn("/app/login?next=other"), undefined);
  assert.equal(
    appToWebRoute(`/app/login?next=${encodeURIComponent(`/app/p/${a}`)}`),
    "/web/login",
  );
});
test("binary URLs must belong to exactly the visible project; aliases never fetch another board", () => {
  const scoped = `/web/p/${a}/designs/asset_1/preview`;
  assert.equal(safeAssetURL(scoped, "preview", a), scoped);
  assert.equal(safeAssetURL(scoped, "preview", b), undefined);
  assert.equal(
    safeAssetURL("/web/designs/asset_1/preview", "preview", a),
    undefined,
  );
  assert.equal(
    safeAssetURL("https://evil.invalid/preview", "preview", a),
    undefined,
  );
  assert.equal(
    safeAssetURL("/web/designs/asset_1/download", "download"),
    "/web/designs/asset_1/download",
  );
});
