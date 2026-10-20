import { test } from "node:test";
import assert from "node:assert/strict";
import { QueryClient } from "@tanstack/react-query";
import {
  DashboardReadError,
  readDashboard,
  validDashboard,
} from "./dashboard-polling.ts";
function snapshot() {
  const at = "2026-10-20T12:00:00Z";
  const agent = {
    session_id: "a".repeat(64),
    role: "developer",
    owner: "Fixture",
    machine: "Fixture machine",
    run_id: "fixture-run",
    status: "active",
    expires_at: at,
  };
  const event = {
    cursor: 1,
    type: "task_created",
    at,
    task_key: "T-1",
    session_id: "a".repeat(64),
    data: { owner: "Fixture", role: "developer", new_state: "ready" },
  };
  return {
    overview: {
      project: "Fixture",
      observed_at: at,
      tasks_total: 1,
      tasks_created: 0,
      tasks_ready: 1,
      tasks_in_progress: 0,
      tasks_done: 0,
      tasks_cancelled: 0,
      tasks_blocked: 0,
      open_epics: 0,
      active_agent_sessions: 1,
      active_human_sessions: 0,
      leased_tasks: 0,
      latest_event_cursor: 1,
    },
    agents: { sessions: [agent], observed_at: at },
    activity: { events: [event], observed_at: at },
  };
}
test("dashboard rejects malformed nested rows, dates and over-limit lists", () => {
  const good = snapshot();
  assert.equal(validDashboard(good), true);
  const malformed = [
    { ...good, agents: { ...good.agents, sessions: [{}] } },
    {
      ...good,
      agents: {
        ...good.agents,
        sessions: [{ ...good.agents.sessions[0], owner: 17 }],
      },
    },
    {
      ...good,
      agents: {
        ...good.agents,
        sessions: [{ ...good.agents.sessions[0], session_id: "not-a-session" }],
      },
    },
    {
      ...good,
      agents: {
        ...good.agents,
        sessions: [{ ...good.agents.sessions[0], role: "human" }],
      },
    },
    {
      ...good,
      agents: {
        ...good.agents,
        sessions: [{ ...good.agents.sessions[0], status: "expired" }],
      },
    },
    {
      ...good,
      agents: {
        ...good.agents,
        sessions: [
          { ...good.agents.sessions[0], task: { key: "T-1", title: {} } },
        ],
      },
    },
    { ...good, agents: { ...good.agents, observed_at: "not-a-date" } },
    { ...good, activity: { ...good.activity, observed_at: "not-a-date" } },
    {
      ...good,
      agents: {
        ...good.agents,
        sessions: Array(7).fill(good.agents.sessions[0]),
      },
    },
    {
      ...good,
      activity: {
        ...good.activity,
        events: Array(21).fill(good.activity.events[0]),
      },
    },
    { ...good, activity: { ...good.activity, events: [{}] } },
    {
      ...good,
      activity: {
        ...good.activity,
        events: [{ ...good.activity.events[0], cursor: -1 }],
      },
    },
    {
      ...good,
      activity: {
        ...good.activity,
        events: [{ ...good.activity.events[0], at: "not-a-date" }],
      },
    },
    {
      ...good,
      activity: {
        ...good.activity,
        events: [{ ...good.activity.events[0], task_key: "T-0" }],
      },
    },
    {
      ...good,
      activity: {
        ...good.activity,
        events: [{ ...good.activity.events[0], data: { owner: {} } }],
      },
    },
  ];
  for (const value of malformed) assert.equal(validDashboard(value), false);
});
test("a malformed response cannot overwrite the last good query snapshot", async () => {
  const original = globalThis.fetch;
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const key = ["dashboard-validation"];
  const good = snapshot();
  client.setQueryData(key, good);
  try {
    globalThis.fetch = async () =>
      Response.json({ ...good, agents: { ...good.agents, sessions: [{}] } });
    await assert.rejects(
      client.fetchQuery({
        queryKey: key,
        queryFn: ({ signal }) => readDashboard(signal),
      }),
      (error) =>
        error instanceof DashboardReadError &&
        error.result.error?.code === "INVALID_RESPONSE",
    );
    assert.equal(client.getQueryData(key), good);
    globalThis.fetch = async () =>
      Response.json({
        ...good,
        agents: { ...good.agents, sessions: null },
        activity: { ...good.activity, events: null },
      });
    const empty = await readDashboard(new AbortController().signal);
    assert.deepEqual(empty.agents.sessions, []);
    assert.deepEqual(empty.activity.events, []);
  } finally {
    globalThis.fetch = original;
    client.clear();
  }
});
