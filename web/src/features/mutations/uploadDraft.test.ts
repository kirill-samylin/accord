import assert from "node:assert/strict";
import test from "node:test";
import type { DesignAsset, PageData } from "../../lib/types.ts";
import { uploadDraft } from "./uploadDraft.ts";
import { uploadBody } from "./contracts.ts";

const taskAsset = {
  design_id: "a".repeat(48),
  scope: "task",
  scope_key: "T-7",
  title: "Existing task design",
} as DesignAsset;
const epicAsset = {
  design_id: "b".repeat(48),
  scope: "epic",
  scope_key: "E-2",
  title: "Existing epic design",
} as DesignAsset;

test("unfiltered and invalid list scopes become a real project upload with no scope key", () => {
  for (const scope of ["", "not-a-scope", "<script>"]) {
    assert.deepEqual(
      uploadDraft({ scope, scope_key: "T-foreign", epic_key: "E-foreign" }),
      {
        title: "",
        scope: "project",
        scope_key: "",
        epic_key: "",
        design_id: "",
      },
    );
  }
});

test("task and epic version links inherit the existing material scope, key and title", () => {
  for (const asset of [taskAsset, epicAsset]) {
    const metadata = uploadDraft(
      { scope: "", scope_key: "", design_id: asset.design_id },
      [asset],
    );
    assert.equal(metadata.scope, asset.scope);
    assert.equal(metadata.scope_key, asset.scope_key);
    assert.equal(metadata.title, asset.title);
    assert.equal(metadata.design_id, asset.design_id);
  }
});

test("confirmed failed-upload metadata stays editable including an explicitly empty title", () => {
  const metadata = uploadDraft(
    {
      design_id: taskAsset.design_id,
      scope: "task",
      scope_key: "T-9",
      title: "",
      epic_key: "E-3",
    },
    [taskAsset],
  );
  assert.equal(metadata.title, "");
  assert.equal(metadata.scope_key, "T-9");
  assert.equal(metadata.epic_key, "E-3");
});

test("upload metadata excludes query capabilities and multipart uses current protected IDs", () => {
  const metadata = uploadDraft({
    scope: "",
    csrf: "stale-csrf",
    action_id: "stale-action",
    cursor: "query-cursor",
    owner: "query-owner",
    title: "Design",
  });
  assert.deepEqual(Object.keys(metadata), [
    "title",
    "scope",
    "scope_key",
    "epic_key",
    "design_id",
  ]);
  const body = uploadBody(
    {
      csrf: "fresh-csrf",
      actions: { upload: "fresh-upload" },
    } as unknown as PageData,
    metadata,
    new File(["fixture"], "design.svg"),
  );
  assert.equal(body.get("scope"), "project");
  assert.deepEqual(body.getAll("csrf"), ["fresh-csrf"]);
  assert.deepEqual(body.getAll("action_id"), ["fresh-upload"]);
  assert.equal([...body.keys()].at(-1), "file");
});
