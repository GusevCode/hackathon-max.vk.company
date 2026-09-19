export async function checkApiHealth(
  signal?: AbortSignal,
  fetcher: typeof fetch = fetch,
): Promise<void> {
  const response = await fetcher("/healthz", {
    headers: { Accept: "text/plain" },
    signal,
  });

  if (!response.ok) {
    throw new Error(`healthcheck returned ${response.status}`);
  }

  const body = (await response.text()).trim();
  if (body !== "ok") {
    throw new Error("unexpected healthcheck response");
  }
}
