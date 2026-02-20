import { useCallback, useEffect, useRef, useState } from "react";
import * as api from "@/api";
import type { TunnelStatus } from "@/types";

export function useTunnels() {
  const [tunnels, setTunnels] = useState<TunnelStatus[]>([]);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const refresh = useCallback(async () => {
    try {
      const data = await api.getTunnels();
      setTunnels(data ?? []);
    } catch {
      // silently ignore tunnel poll failures
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
    (project: string, service: string): TunnelStatus | undefined => {
      return tunnels.find(
        (t) => t.project === project && t.service === service
      );
    },
    [tunnels]
  );

  return { tunnels, lookup, refresh };
}
