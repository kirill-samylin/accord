import { afterEach, test } from "node:test";
import assert from "node:assert/strict";
import { adminActions, readPage, submitAction } from "./api.ts";
import {
  appToWebRoute,
  projectAppPath,
  routeScope,
  safeLoginReturn,
  webToAppRoute,
} from "./project-routes.ts";
import { isAdminPage, isCatalogPage, type AdminPage } from "./projects.ts";
import {
  auditTime,
  personProjects,
  samePersonName,
  tokenFileName,
  unlinkedIdentities,
  validateCredential,
} from "../features/admin/admin-model.ts";
const a = "p_" + "a".repeat(32);
const u = "u_" + "1".repeat(32),
  i = "i_" + "2".repeat(32),
  legacy = "legacy_" + "3".repeat(64);
const originalFetch = globalThis.fetch;
afterEach(() => {
  globalThis.fetch = originalFetch;
});
test("the administration screen maps to the catalog admin page in both directions", () => {
  assert.equal(appToWebRoute("/app/admin"), "/web/catalog/admin");
  assert.equal(appToWebRoute("/app/admin/"), "/web/catalog/admin");
  assert.equal(webToAppRoute("/web/catalog/admin"), "/app/admin");
  assert.equal(webToAppRoute("/web/catalog/admin/"), "/app/admin");
  assert.equal(routeScope("/app/admin"), "catalog");
  assert.equal(routeScope("/app/admin"), routeScope("/app/projects"));
  assert.equal(projectAppPath(a, "/app/admin"), "/app/admin");
  assert.equal(projectAppPath(undefined, "/app/admin"), "/app/admin");
  assert.equal(safeLoginReturn("/app/admin"), "/app/admin");
  for (const path of [
    "/app/admin/people",
    `/app/p/${a}/admin`,
    "/app/admin/../projects",
    "/web/admin",
  ])
    assert.throws(() => appToWebRoute(path), path);
  assert.throws(() => webToAppRoute("/web/catalog/admin/people"));
  assert.throws(() => webToAppRoute("/web/admin"));
});
test("admin mutation paths are allowed exactly as the contract lists them", async () => {
  for (const path of [
    "/web/catalog/admin/people",
    `/web/catalog/admin/people/${u}`,
    `/web/catalog/admin/people/${u}/credentials`,
    `/web/catalog/admin/identities/${i}`,
    `/web/catalog/admin/identities/${legacy}`,
  ])
    assert.ok(adminActions.test(path), path);
  for (const path of [
    "/web/catalog/admin",
    "/web/catalog/admin/people/",
    "/web/catalog/admin/people/u_abc",
    `/web/catalog/admin/people/${u}/credentials/extra`,
    `/web/catalog/admin/people/${u.toUpperCase()}`,
    "/web/catalog/admin/identities",
    `/web/catalog/admin/identities/${i}/revoke`,
    "/web/catalog/admin/identities/legacy_abc",
    `/web/catalog/admin/identities/${u}`,
    `/web/catalog/admin/audit`,
  ]) {
    assert.equal(adminActions.test(path), false, path);
    await assert.rejects(
      submitAction(path, new URLSearchParams()),
      /Недопустимое действие/,
      path,
    );
  }
  let sent = "";
  globalThis.fetch = async (actual) => {
    sent = String(actual);
    return Response.json({
      ok: true,
      issued: {
        identity_id: i,
        owner: "Анна",
        machine: "laptop",
        expires_at: "2026-11-24T00:00:00Z",
        secret: "s3cret",
      },
    });
  };
  const result = await submitAction(
    `/web/catalog/admin/people/${u}/credentials`,
    new URLSearchParams({ machine: "laptop", days: "30" }),
  );
  assert.equal(sent, `/web/catalog/admin/people/${u}/credentials`);
  assert.equal(result.ok, true);
});
function page(overrides: Partial<AdminPage> = {}): AdminPage {
  return {
    name: "admin",
    registry: {
      identity: { id: i, owner: "Анна", machine: "laptop", admin: true },
      csrf: "csrf",
      actions: {},
    },
    people: [
      {
        id: u,
        name: "Анна",
        note: "",
        role: "admin",
        active: true,
        version: 1,
        created_at: "2026-10-01T00:00:00Z",
        updated_at: "2026-10-01T00:00:00Z",
      },
    ],
    identities: [
      {
        id: i,
        person_id: u,
        owner: "Анна",
        machine: "laptop",
        source: "catalog",
        expires_at: "2026-12-01T00:00:00Z",
        revoked: false,
      },
      {
        id: legacy,
        person_id: "",
        owner: "old",
        machine: "pc",
        source: "legacy",
        expires_at: "2026-12-01T00:00:00Z",
        revoked: false,
      },
    ],
    projects: [{ id: a, name: "Accord", state: "active" }],
    memberships: [
      { project_id: a, identity_id: i, role: "viewer" },
      { project_id: a, identity_id: i, role: "owner" },
      { project_id: a, identity_id: legacy, role: "editor" },
    ],
    audit: [],
    ...overrides,
  };
}
test("the admin page is recognised only with its full shape and reaches the admin screen", async () => {
  const data = page();
  assert.ok(isCatalogPage(data));
  assert.ok(isAdminPage(data));
  assert.equal(isAdminPage({ ...data, name: "projects" }), false);
  assert.equal(isAdminPage({ ...data, audit: undefined }), false);
  globalThis.fetch = async () => Response.json({ ok: true, page: data });
  const loaded = await readPage<AdminPage>("/app/admin");
  assert.equal(loaded.name, "admin");
  assert.equal(loaded.registry.identity.admin, true);
});
test("people see their projects through identities, unlinked identities surface and token files are named by owner and machine", () => {
  const data = page();
  assert.deepEqual(personProjects(data, data.people[0]), [
    { id: a, name: "Accord", state: "active", role: "owner" },
  ]);
  assert.deepEqual(
    unlinkedIdentities(data).map((identity) => identity.id),
    [legacy],
  );
  assert.equal(tokenFileName("Анна", "laptop"), "Анна-laptop.token");
  assert.equal(tokenFileName("a/b", "c\\d"), "a_b-c_d.token");
  assert.deepEqual(
    validateCredential(new URLSearchParams({ machine: "pc", days: "30" })),
    {},
  );
  assert.ok(
    validateCredential(new URLSearchParams({ machine: "", days: "400" })).days,
  );
  assert.ok(
    validateCredential(new URLSearchParams({ machine: "pc", days: "0" })).days,
  );
  assert.equal(auditTime(undefined), "—");
  assert.notEqual(auditTime(1761350400), "—");
  assert.notEqual(auditTime("2026-10-25T00:00:00Z"), "—");
  assert.equal(auditTime(1761350400), auditTime("2026-10-25T00:00:00Z"));
});

test("a person with the same name only warns", () => {
  const people = [{ name: "Анна  Иванова" }];
  assert.equal(samePersonName(people, " анна иванова "), true);
  assert.equal(samePersonName(people, "Анна"), false);
  assert.equal(samePersonName(people, "   "), false);
});
