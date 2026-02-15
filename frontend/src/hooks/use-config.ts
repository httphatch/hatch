import { useCallback, useEffect, useState } from "react";
import * as api from "@/api";

export function useConfig() {
  const [yaml, setYaml] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    try {
      const data = await api.getConfig();
      setYaml(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load config");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const save = useCallback(async (content: string) => {
    setSaving(true);
    setError(null);
    try {
      await api.updateConfig(content);
      await api.restartDaemon();
      setYaml(content);
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Failed to save config";
      setError(msg);
      throw err;
    } finally {
      setSaving(false);
    }
  }, []);

  return { yaml, loading, saving, error, save, refresh };
}
