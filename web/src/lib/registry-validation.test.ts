import assert from "node:assert/strict";
import { test } from "node:test";
import {
  memberIDError,
  projectNameError,
  repositoryURLError,
  validateMember,
  validateProjectFields,
  validateRepository,
} from "./registry-validation.ts";

test("project name is trimmed before validation and sending", () => {
  const values = new URLSearchParams({ name: "   " });
  assert.ok(validateProjectFields(values).name);
  const padded = new URLSearchParams({ name: "  Accord  " });
  assert.deepEqual(validateProjectFields(padded), {});
  assert.equal(padded.get("name"), "Accord");
  assert.ok(projectNameError("я".repeat(61)), "Go limit is 120 bytes");
  assert.equal(projectNameError("я".repeat(60)), "");
});

test("member id accepts only i_<32hex> and legacy_<64hex>", () => {
  assert.equal(memberIDError("i_" + "a".repeat(32)), "");
  assert.equal(memberIDError("legacy_" + "0".repeat(64)), "");
  assert.ok(memberIDError("i_123"));
  assert.ok(memberIDError("bogus"));
  const values = new URLSearchParams({ identity_id: " i_" + "b".repeat(32) });
  assert.deepEqual(validateMember(values), {});
  assert.equal(values.get("identity_id"), "i_" + "b".repeat(32));
});

test("repository url mirrors the server rules", () => {
  assert.equal(repositoryURLError("https://github.com/o/r.git"), "");
  assert.equal(repositoryURLError("ssh://git@github.com/o/r.git"), "");
  assert.ok(repositoryURLError("https://user:pw@github.com/o/r.git"));
  assert.ok(repositoryURLError("https://github.com"));
  assert.ok(repositoryURLError("git@github.com:o/r.git"));
  assert.ok(repositoryURLError("https://github.com/o/r?x=1"));
  assert.ok(repositoryURLError(""));
  assert.equal(repositoryURLError("", false), "");
  assert.ok(
    validateRepository(
      new URLSearchParams({ url: "https://h/o/r", branch: "a..b" }),
    ).branch,
  );
});
