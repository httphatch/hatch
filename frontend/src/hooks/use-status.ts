import { useEffect, useState } from "react";
import * as api from "@/api";
import type { DaemonStatus } from "@/types";

export function useStatus() {
  const [status, setStatus] = useState<DaemonStatus | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .getStatus()
      .then(setStatus)
      .catch((err) =>
        setError(err instanceof Error ? err.message : "Failed to get status")
      );
  }, []);

  return { status, error };
}
