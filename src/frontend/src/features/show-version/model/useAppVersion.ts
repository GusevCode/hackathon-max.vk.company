import { useEffect, useState } from "react";
import { fetchAppVersion } from "../../../shared/api/version";

export function useAppVersion() {
  const [version, setVersion] = useState<string | null>(null);

  useEffect(() => {
    const controller = new AbortController();
    void fetchAppVersion(controller.signal)
      .then((payload) => setVersion(payload.version))
      .catch(() => {
        if (!controller.signal.aborted) setVersion("недоступна");
      });

    return () => controller.abort();
  }, []);

  return version ?? "проверяем…";
}
