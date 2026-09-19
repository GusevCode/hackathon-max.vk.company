import { useCallback, useEffect, useState } from "react";
import { checkApiHealth } from "../../../shared/api/health";
import type { ApiState } from "../../../entities/service-status/model/types";

export function useApiHealth() {
  const [state, setState] = useState<ApiState>("checking");
  const [isRefreshing, setIsRefreshing] = useState(false);

  const refresh = useCallback(async (signal?: AbortSignal) => {
    setIsRefreshing(true);
    setState("checking");
    try {
      await checkApiHealth(signal);
      setState("online");
    } catch {
      if (!signal?.aborted) setState("offline");
    } finally {
      if (!signal?.aborted) setIsRefreshing(false);
    }
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    void refresh(controller.signal);
    return () => controller.abort();
  }, [refresh]);

  return { state, isRefreshing, refresh };
}
