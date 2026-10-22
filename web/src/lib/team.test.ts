import assert from "node:assert/strict";
import test from "node:test";
import { ago, percent, personTone, since } from "./people.ts";
import { similarQuery, validSimilar, validTeam } from "./team.ts";
import { appToWebRoute, webToAppRoute } from "./project-routes.ts";

test("person tone is stable and case-insensitive", () => {
  assert.equal(personTone("Denis"), personTone(" denis "));
  assert.ok(personTone("kirill") >= 0 && personTone("kirill") < 8);
  assert.equal(personTone(undefined), 0);
});

test("relative time is compact and never negative", () => {
  const now = Date.parse("2026-10-22T12:00:00Z");
  assert.equal(since("2026-10-22T11:59:40Z", now), "<1 мин");
  assert.equal(since("2026-10-22T11:48:00Z", now), "12 мин");
  assert.equal(since("2026-10-22T09:30:00Z", now), "2 ч 30 мин");
  assert.equal(since("2026-10-22T12:05:00Z", now), "<1 мин");
  assert.equal(ago("2026-10-22T11:59:59Z", now), "только что");
  assert.equal(since("bad", now), "");
  assert.equal(percent(0.834), "83%");
  assert.equal(percent(1.4), "100%");
});

test("similar query is bounded and omits role for epics", () => {
  const query = new URLSearchParams(
    similarQuery({ title: "x".repeat(300), type: "epic", role: "developer" }),
  );
  assert.equal(query.get("title")?.length, 200);
  assert.equal(query.get("role"), null);
  assert.equal(
    new URLSearchParams(similarQuery({ title: "A", role: "developer" })).get(
      "role",
    ),
    "developer",
  );
});

test("team and similar payloads are validated", () => {
  const team = {
    observed_at: "2026-10-22T12:00:00Z",
    members: [{ owner: "denis", agents: [], machines: [] }],
    unclaimed: [],
    duplicates: [],
    duplicates_total: 0,
    duplicate_mode: "warn",
  };
  assert.ok(validTeam(team));
  assert.ok(!validTeam({ ...team, members: [{ owner: 1 }] }));
  assert.ok(
    validSimilar({
      mode: "warn",
      similar: [{ key: "T-3", title: "a", score: 0.7 }],
    }),
  );
  assert.ok(
    !validSimilar({
      mode: "warn",
      similar: [{ key: "../x", title: "a", score: 1 }],
    }),
  );
});

test("team screen routes between app and web", () => {
  assert.equal(appToWebRoute("/app/team"), "/web/team");
  assert.equal(webToAppRoute("/web/team"), "/app/team");
  assert.equal(
    appToWebRoute("/app/p/p_0123456789abcdef0123456789abcdef/team"),
    "/web/p/p_0123456789abcdef0123456789abcdef/team",
  );
});
