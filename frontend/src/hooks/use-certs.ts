import { useCallback, useEffect, useState } from "react";
import * as api from "@/api";
import type { CertStatus } from "@/types";

export function useCerts() {
  const [certs, setCerts] = useState<CertStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    try {
      const data = await api.getCerts();
      setCerts(data);
      setError(null);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to load certificates"
      );
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return { certs, loading, error, refresh };
}
