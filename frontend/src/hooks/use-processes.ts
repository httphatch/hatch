import { useCallback, useEffect, useRef, useState } from "react";
import * as api from "@/api";
import type { ProcessStatus } from "@/types";

export function useProcesses() {
  const [processes, setProcesses] = useState<ProcessStatus[]>([]);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const refresh = useCallback(async () => {
    try {
      const data = await api.getProcesses();
      setProcesses(data ?? []);
    } catch {
      // silently ignore process poll failures
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
    (project: string, service: string): ProcessStatus | undefined => {
      return processes.find(
        (p) => p.project === project && p.service === service
      );
    },
    [processes]
  );

  return { processes, lookup, refresh };
}
