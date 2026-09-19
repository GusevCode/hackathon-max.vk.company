import { describe, expect, it, vi } from "vitest";
import { checkApiHealth } from "./health";

describe("checkApiHealth", () => {
  it("accepts a healthy response", async () => {
    const fetcher = vi.fn(async () => new Response("ok\n", { status: 200 }));

    await expect(checkApiHealth(undefined, fetcher)).resolves.toBeUndefined();
    expect(fetcher).toHaveBeenCalledWith("/healthz", {
      headers: { Accept: "text/plain" },
      signal: undefined,
    });
  });

  it("rejects an unhealthy response", async () => {
    const fetcher = vi.fn(async () => new Response("unavailable", { status: 503 }));

    await expect(checkApiHealth(undefined, fetcher)).rejects.toThrow(
      "healthcheck returned 503",
    );
  });
});
