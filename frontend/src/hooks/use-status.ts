import { useCallback, useEffect, useState } from "react";
import * as api from "@/api";
import type { DaemonStatus } from "@/types";

export function useStatus() {
  const [status, setStatus] = useState<DaemonStatus | null>(null);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(() => {
    api
      .getStatus()
      .then((s) => {
        setStatus(s);
        setError(null);
      })
      .catch((err) => {
        setStatus(null);
        setError(err instanceof Error ? err.message : "Failed to get status");
      });
  }, []);

  useEffect(() => {
    refresh();
    const id = setInterval(refresh, 60_000);
    return () => clearInterval(id);
  }, [refresh]);

  return { status, error, refresh };
}
