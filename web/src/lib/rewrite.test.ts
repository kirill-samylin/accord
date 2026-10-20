import { test } from "node:test";
import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { browserRewrites } from "./upstream.ts";
const require = createRequire(import.meta.url);
// Exercise the exact compiler used by the pinned Next runtime. Checking only
// our destination string missed its removal of the slash for an empty :path*.
const { prepareDestination } =
  require("next/dist/shared/lib/router/utils/prepare-destination.js") as {
    prepareDestination: (args: {
      appendParamsToQuery: boolean;
      destination: string;
      params: Record<string, string | string[]>;
      query: Record<string, string | string[]>;
    }) => {
      parsedDestination: {
        pathname: string;
        query: Record<string, string | string[]>;
      };
    };
  };
test("literal browser root rewrite retains Go slash and all legacy view query fields", () => {
  const routes = browserRewrites("http://127.0.0.1:3101");
  assert.equal(routes[0].source, "/web/");
  const queries: Record<string, string | string[]>[] = [
    {},
    { view: "questions", state: "ready" },
    { view: "tasks", q: "fixture" },
    { view: ["tasks", "questions"] },
    { view: "unknown" },
  ];
  for (const query of queries) {
    const actual = prepareDestination({
      appendParamsToQuery: true,
      destination: routes[0].destination,
      params: {},
      query,
    });
    assert.equal(actual.parsedDestination.pathname, "/web/");
    assert.deepEqual(actual.parsedDestination.query, query);
  }
  const wildcard = prepareDestination({
    appendParamsToQuery: true,
    destination: routes[1].destination,
    params: {},
    query: {},
  });
  assert.equal(wildcard.parsedDestination.pathname, "/web");
});
test("nonroot browser paths keep the existing fixed upstream and actual wildcard segments", () => {
  const routes = browserRewrites("http://127.0.0.1:3101");
  const actual = prepareDestination({
    appendParamsToQuery: true,
    destination: routes[1].destination,
    params: { path: ["tasks", "T-1"] },
    query: {},
  });
  assert.equal(actual.parsedDestination.pathname, "/web/tasks/T-1");
  assert.throws(() => browserRewrites("https://remote.invalid:3101"));
});
