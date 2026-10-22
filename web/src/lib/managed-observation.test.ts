import { test } from "node:test";
import assert from "node:assert/strict";
import {
  commandReason,
  hostOnline,
  startErrors,
  managedConflictVersion,
} from "./managed-agents.ts";
import {
  observedAt,
  now,
  host,
  run,
  profile,
  draft,
  page,
} from "./managed-fixtures.test-support.ts";
import { applyActionResult } from "../features/mutations/actionState.ts";

test("server snapshot freshness and start/resume remain stable with browser clock behind or ahead", () => {
  const originalNow = Date.now;
  try {
    for (const skew of [-86400000, -2000, 0, 2000, 86400000]) {
      Date.now = () => now + skew;
      assert.equal(
        hostOnline(host(), observedAt),
        true,
        `browser clock skew ${skew}`,
      );
      assert.deepEqual(
        startErrors(
          draft(),
          { observed_at: observedAt, hosts: [host()], runs: [] },
          [profile()],
          true,
        ),
        {},
      );
      assert.equal(
        commandReason(
          "resume",
          { ...run(), state: "stopped" },
          host(),
          true,
          observedAt,
        ),
        "",
      );
      const old = {
        ...host(),
        last_seen_at: new Date(now - 31000).toISOString(),
      };
      assert.equal(hostOnline(old, observedAt), false);
      assert.ok(
        startErrors(
          draft(),
          { observed_at: observedAt, hosts: [old], runs: [] },
          [profile()],
          true,
        ).host_id,
      );
      assert.ok(
        commandReason(
          "resume",
          { ...run(), state: "stopped" },
          old,
          true,
          observedAt,
        ),
      );
    }
  } finally {
    Date.now = originalNow;
  }
});

test("a later server observation ages heartbeat and ownership independently of the browser", () => {
  assert.equal(hostOnline(host(), observedAt), true);
  assert.equal(hostOnline(host(), new Date(now + 31000).toISOString()), false);
  assert.equal(
    hostOnline({ ...host(), lease_expires_at: observedAt }, observedAt),
    false,
  );
  assert.equal(hostOnline(host(), "invalid observation"), false);
});

test("only newer detail of the rejected run can acknowledge a version conflict", () => {
  const old = page();
  const detail = { run: { ...run(), version: 3 }, commands: [] };
  const state = applyActionResult(old, {
    ok: false,
    error: { code: "conflict", message: "Version differs" },
    page: { ...old, managed_run: detail },
    actions: { "managed:command": "fresh" },
  });
  assert.equal(
    managedConflictVersion(
      state.page.view,
      state.page.managed_run,
      run().id,
      2,
    ),
    3,
  );
  assert.equal(
    managedConflictVersion("managed-run", old.managed_run, run().id, 2),
    undefined,
  );
  assert.equal(
    managedConflictVersion("managed-run", detail, "another-run", 2),
    undefined,
  );
  assert.equal(
    managedConflictVersion("managed-run", undefined, run().id, 2),
    undefined,
  );
  const missing = applyActionResult(old, {
    ok: false,
    error: { code: "conflict", message: "Version differs" },
    page: { ...old, view: "managed", managed_run: undefined },
  });
  assert.equal(
    managedConflictVersion(
      missing.page.view,
      missing.page.managed_run,
      run().id,
      2,
    ),
    undefined,
  );
});
