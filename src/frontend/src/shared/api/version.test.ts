import { describe, expect, it, vi } from "vitest";
import { fetchAppVersion } from "./version";

describe("fetchAppVersion", () => {
  it("returns the deployed version", async () => {
    const fetcher = vi.fn(async () =>
      new Response(JSON.stringify({ version: "abc123" }), { status: 200 }),
    );

    await expect(fetchAppVersion(undefined, fetcher)).resolves.toEqual({ version: "abc123" });
    expect(fetcher).toHaveBeenCalledWith("/api/version", {
      headers: { Accept: "application/json" },
      signal: undefined,
    });
  });

  it("rejects an invalid response", async () => {
    const fetcher = vi.fn(async () => new Response(JSON.stringify({}), { status: 200 }));

    await expect(fetchAppVersion(undefined, fetcher)).rejects.toThrow("unexpected version response");
  });
});
