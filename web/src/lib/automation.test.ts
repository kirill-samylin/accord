import { afterEach, test } from "node:test";
import assert from "node:assert/strict";
import {
  automationPollInterval,
  automationRoles,
  requirementError,
  workflowControlReason,
  type AutomationPolicy,
  resumeNeedsPolicy,
  type WorkflowGrant,
  type WorkflowPage,
} from "./automation.ts";
import { newPolicy, policyErrors } from "./automation-policy.ts";
import { APIError, readPage, submitAction } from "./api.ts";
import {
  appToWebRoute,
  projectAppPath,
  webToAppRoute,
} from "./project-routes.ts";
import type { ManagedHost } from "./managed-agents.ts";
import type { ProfileSummary } from "./agent-profiles.ts";
const now = "2026-10-23T12:00:00Z",
  expiry = "2026-10-24T12:00:00Z";
const project = `p_${"a".repeat(32)}`,
  other = `p_${"b".repeat(32)}`;
const preset = {
  preset_id: "tests",
  local_version: 1,
  local_digest: "a".repeat(64),
  kind: "test",
  required: true,
  isolation: "trusted_owner_process",
};
const scope = {
  version: 1,
  expires_at: expiry,
  roles: Object.fromEntries(automationRoles.map((r) => [r, ["codex"]])),
  checks: [preset],
  max_step_seconds: 1800,
  max_workflow_seconds: 28800,
  max_native_invocations: 40,
};
const host: ManagedHost = {
  id: "mh_demo",
  owner_principal_id: "owner",
  name: "Mac",
  version: 1,
  state: "active",
  controllers: ["owner"],
  workspaces: ["main", "other"],
  providers: ["codex", "claude"],
  last_seen_at: now,
  worker_epoch: 1,
  lease_expires_at: expiry,
  can_control: true,
  worker_capabilities: { automation: 1 },
  automation: {
    scopes: { main: scope },
    enabled: true,
    version: 1,
    expires_at: expiry,
    workspaces: ["main", "other"],
    roles: [...automationRoles],
    providers: ["codex", "claude"],
    checks: [preset],
    max_step_seconds: 1800,
  },
};
const profiles: ProfileSummary[] = automationRoles.map((role, index) => ({
  id: `ap_${String(index).repeat(32)}`,
  version: 1,
  name: role,
  description: "",
  role,
  provider: "unconfigured",
  model: "",
  state: "active",
  created_at: now,
  updated_at: now,
  created_by: "owner",
  updated_by: "owner",
}));
function settings() {
  return {
    ...newPolicy(now, "owner"),
    host_id: host.id,
    host_consent_version: 1,
    workspace: "main",
    checks: [preset],
    role_bindings: Object.fromEntries(
      profiles.map((p) => [
        p.role,
        { provider: "codex", profile_id: p.id, profile_version: p.version },
      ]),
    ),
  };
}
const original = globalThis.fetch;
afterEach(() => {
  globalThis.fetch = original;
});
test("policy selects only exact workspace consent, never flat cross-product", () => {
  const draft = settings();
  assert.deepEqual(policyErrors(draft, [host], profiles, now), {});
  assert.ok(
    policyErrors({ ...draft, workspace: "other" }, [host], profiles, now)
      .workspace,
  );
  assert.ok(
    policyErrors(
      {
        ...draft,
        role_bindings: {
          ...draft.role_bindings,
          analyst: { ...draft.role_bindings.analyst, provider: "claude" },
        },
      },
      [host],
      profiles,
      now,
    ).analyst,
  );
  assert.ok(
    policyErrors(draft, [{ ...host, worker_capabilities: {} }], profiles, now)
      .host_id,
  );
});
test("policy rejects missing tests, changed consent/profiles, invalid limits and expired local scope", () => {
  const draft = settings();
  assert.ok(
    policyErrors({ ...draft, checks: [] }, [host], profiles, now).checks,
  );
  assert.ok(
    policyErrors(
      { ...draft, checks: [{ ...preset, kind: "lint" }] },
      [host],
      profiles,
      now,
    ).checks,
  );
  assert.ok(
    policyErrors({ ...draft, host_consent_version: 2 }, [host], profiles, now)
      .host_id,
  );
  assert.ok(
    policyErrors(
      draft,
      [host],
      profiles.map((p) => ({ ...p, version: 2 })),
      now,
    ).developer,
  );
  assert.ok(
    policyErrors(
      { ...draft, limits: { ...draft.limits, native_invocations: 41 } },
      [host],
      profiles,
      now,
    ).limits,
  );
  assert.ok(
    policyErrors(
      { ...draft, limits: { ...draft.limits, plan_tasks: 0 } },
      [host],
      profiles,
      now,
    ).limits,
  );
  const restricted = {
    ...host,
    automation: {
      ...host.automation!,
      scopes: { main: { ...scope, expires_at: "2026-10-23T13:00:00Z" } },
    },
  };
  assert.ok(policyErrors(draft, [restricted], profiles, now).expires_at);
});
test("new policy keeps client intake off and never silently replaces saved revisions", () => {
  const fresh = newPolicy(now, "owner");
  assert.equal(fresh.client_intake, false);
  assert.equal(fresh.limits.plan_tasks, 8);
  assert.deepEqual(fresh.role_bindings, {});
  const saved = {
    ...settings(),
    id: "apolicy_saved",
    version: 3,
    state: "active",
    issuer_principal_id: "owner",
    issuer_epoch: 1,
    host_owner_epoch: 1,
    created_at: now,
    updated_at: now,
  } satisfies AutomationPolicy;
  assert.deepEqual(
    newPolicy(now, "another", saved).role_bindings,
    saved.role_bindings,
  );
  assert.equal(newPolicy(now, "another", saved).expires_at, saved.expires_at);
});
test("control never resumes working or ambiguous stopping process and denies readonly", () => {
  const detail = {
    workflow: { state: "running" },
    can_control: true,
  } as WorkflowPage;
  assert.ok(workflowControlReason("resume", detail, true));
  assert.equal(workflowControlReason("pause", detail, true), "");
  assert.ok(workflowControlReason("cancel", detail, false));
  assert.ok(
    workflowControlReason(
      "resume",
      { ...detail, workflow: { ...detail.workflow, state: "stopping" } },
      true,
    ),
  );
  const grant = { expires_at: "2999-01-01T00:00:00Z" } as WorkflowGrant;
  assert.equal(
    workflowControlReason(
      "resume",
      { ...detail, workflow: { ...detail.workflow, state: "paused", grant } },
      true,
    ),
    "",
  );
});
test("resume of a draft or ungranted requirement needs an explicit policy", () => {
  const draft = {
    workflow: { state: "awaiting_authorization" },
    can_control: true,
  } as WorkflowPage;
  assert.ok(resumeNeedsPolicy(draft));
  assert.ok(workflowControlReason("resume", draft, true));
  assert.equal(workflowControlReason("resume", draft, true, true), "");
  const paused = { ...draft, workflow: { state: "paused" } } as WorkflowPage;
  assert.ok(resumeNeedsPolicy(paused), "no grant");
  assert.equal(resumeNeedsPolicy({ ...draft, can_resume: true }), false);
  assert.equal(resumeNeedsPolicy({ ...paused, can_resume: false }), true);
});
test("intake bounds use bytes; automation polling respects project locks and errors", () => {
  assert.ok(requirementError("  "));
  assert.ok(requirementError("я".repeat(8193)));
  assert.equal(requirementError("Сделать страницу"), "");
  assert.equal(
    automationPollInterval(`/app/p/${project}/automation`, false, false),
    5000,
  );
  assert.equal(
    automationPollInterval(`/app/p/${project}/automation`, true, false),
    false,
  );
  assert.equal(
    automationPollInterval(`/app/p/${project}/automation`, false, false, 403),
    false,
  );
  assert.equal(
    automationPollInterval(
      `/app/p/${project}/automation`,
      false,
      false,
      503,
      3,
    ),
    30000,
  );
});
test("workflow and policy links preserve project scope including pagination", () => {
  for (const tail of [
    "/automation",
    "/automation/workflows/wf_demo?cursor=opaque",
    "/automation/policies/apolicy_demo",
  ]) {
    assert.equal(
      appToWebRoute(`/app/p/${project}${tail}`),
      `/web/p/${project}${tail}`,
    );
    assert.equal(
      webToAppRoute(`/web/p/${project}${tail}`),
      `/app/p/${project}${tail}`,
    );
  }
  assert.throws(() => projectAppPath(project, `/app/p/${other}/automation`));
});
test("automation actions are allowlisted; cross-project redirects remain unknown and cannot navigate", async () => {
  let called = 0;
  globalThis.fetch = async () => {
    called++;
    return new Response(
      JSON.stringify({
        ok: true,
        redirect: `/web/p/${other}/automation/workflows/wf_demo`,
      }),
      { status: 200 },
    );
  };
  await assert.rejects(
    submitAction(
      `/web/p/${project}/automation/submit`,
      new URLSearchParams({ text: "do" }),
    ),
    (error: unknown) =>
      error instanceof APIError && error.result.retry_same_action === true,
  );
  assert.equal(called, 1);
  await assert.rejects(
    submitAction(
      `/web/p/${project}/automation/admin-bypass`,
      new URLSearchParams(),
    ),
  );
  assert.equal(called, 1);
});
test("readonly project projection removes automation control despite stale server flag", async () => {
  globalThis.fetch = async () =>
    new Response(
      JSON.stringify({
        ok: true,
        page: {
          name: "automation",
          session: {},
          project: { id: project, can_write: false },
          workflow: { can_control: true },
          can_manage_automation: true,
        },
      }),
      { status: 200 },
    );
  const page = await readPage(`/app/p/${project}/automation`);
  assert.equal(page.can_manage_automation, false);
  assert.equal(page.workflow?.can_control, false);
});
