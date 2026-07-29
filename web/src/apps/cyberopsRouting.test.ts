import { describe, expect, it } from "vitest";
import { createMemoryHistory, createRouter } from "@tanstack/react-router";
import { router } from "../router";

describe("CyberOps router registration", () => {
  it("matches the traffic dashboard route", async () => {
    expect(router.routesById["/cyberops/dashboard"]).toBeDefined();
    const memoryRouter = createRouter({
      routeTree: router.routeTree,
      history: createMemoryHistory({ initialEntries: ["/cyberops/dashboard"] }),
    });
    await memoryRouter.load();
    expect(memoryRouter.state.matches.map((match) => match.routeId)).toContain("/cyberops/dashboard");
  });
});
