export type AppVersion = {
  version: string;
};

export async function fetchAppVersion(
  signal?: AbortSignal,
  fetcher: typeof fetch = fetch,
): Promise<AppVersion> {
  const response = await fetcher("/api/version", {
    headers: { Accept: "application/json" },
    signal,
  });

  if (!response.ok) {
    throw new Error(`version endpoint returned ${response.status}`);
  }

  const payload = (await response.json()) as Partial<AppVersion>;
  if (!payload.version || typeof payload.version !== "string") {
    throw new Error("unexpected version response");
  }

  return { version: payload.version };
}
