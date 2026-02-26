import { useCallback, useEffect, useState } from "react";
import * as api from "@/api";
import type { Settings } from "@/types";

export function useSettings() {
  const [settings, setSettings] = useState<Settings | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    try {
      const data = await api.getSettings();
      setSettings(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load settings");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const save = useCallback(async (updated: Settings) => {
    setSaving(true);
    setError(null);
    try {
      await api.updateSettings(updated);
      await api.reloadConfig();
      setSettings(updated);
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Failed to save settings";
      setError(msg);
      throw err;
    } finally {
      setSaving(false);
    }
  }, []);

  return { settings, loading, saving, error, save, refresh };
}
