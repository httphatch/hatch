import { useCallback, useEffect, useRef, useState } from "react";
import * as api from "@/api";
import type { ServiceHealth } from "@/types";

export function useHealth() {
  const [health, setHealth] = useState<ServiceHealth[]>([]);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const refresh = useCallback(async () => {
    try {
      const data = await api.getHealth();
      setHealth(data ?? []);
    } catch {
      // silently ignore health poll failures
    }
  }, []);

  useEffect(() => {
    refresh();
    intervalRef.current = setInterval(refresh, 10_000);
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [refresh]);

  const lookup = useCallback(
    (project: string, service: string): ServiceHealth | undefined => {
      return health.find(
        (h) => h.project === project && h.service === service
      );
    },
    [health]
  );

  return { health, lookup, refresh };
}
