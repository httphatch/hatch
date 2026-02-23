import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Eye, EyeOff } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { useSettings } from "@/hooks/use-settings";
import type { Settings } from "@/types";

const TLD_OPTIONS = ["test", "localhost", "local", "dev"];
const LOG_LEVEL_OPTIONS = ["debug", "info", "warn", "error"];

export function SettingsForm() {
  const { settings, loading, saving, error, save } = useSettings();
  const [form, setForm] = useState<Settings | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [showToken, setShowToken] = useState(false);

  useEffect(() => {
    if (settings) {
      setForm({ ...settings });
    }
  }, [settings]);

  if (loading) {
    return (
      <p className="py-8 text-center text-muted-foreground">Loading settings...</p>
    );
  }

  if (!form) {
    return (
      <p className="py-8 text-center text-destructive">
        {error || "Failed to load settings"}
      </p>
    );
  }

  function set<K extends keyof Settings>(key: K, value: Settings[K]) {
    setForm((prev) => (prev ? { ...prev, [key]: value } : prev));
  }

  function parsePort(value: string): number {
    const n = parseInt(value, 10);
    return isNaN(n) ? 0 : n;
  }

  async function handleSave() {
    if (!form) return;
    setSaveError(null);

    if (form.http_port < 1 || form.http_port > 65535) {
      setSaveError("HTTP port must be between 1 and 65535");
      return;
    }
    if (form.https_port < 1 || form.https_port > 65535) {
      setSaveError("HTTPS port must be between 1 and 65535");
      return;
    }
    if (form.http_port === form.https_port) {
      setSaveError("HTTP and HTTPS ports must be different");
      return;
    }

    try {
      await save(form);
      toast.success("Settings saved. Daemon reloaded.");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to save settings"
      );
    }
  }

  return (
    <div>
      <div className="border-b border-border">
        <div className="flex items-center border-b border-border bg-surface/50 px-4 py-2">
          <h3 className="text-sm font-semibold text-foreground">Network</h3>
        </div>
        <div className="space-y-4 px-4 py-4">
          <div className="grid grid-cols-3 gap-4">
            <div className="space-y-2">
              <Label htmlFor="tld">TLD</Label>
              <Select value={form.tld} onValueChange={(v) => set("tld", v)}>
                <SelectTrigger id="tld" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {TLD_OPTIONS.map((tld) => (
                    <SelectItem key={tld} value={tld}>
                      .{tld}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="http-port">HTTP Port</Label>
              <Input
                id="http-port"
                type="number"
                min={1}
                max={65535}
                value={form.http_port}
                onChange={(e) => set("http_port", parsePort(e.target.value))}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="https-port">HTTPS Port</Label>
              <Input
                id="https-port"
                type="number"
                min={1}
                max={65535}
                value={form.https_port}
                onChange={(e) => set("https_port", parsePort(e.target.value))}
              />
            </div>
          </div>
        </div>
      </div>

      <div className="border-b border-border">
        <div className="flex items-center border-b border-border bg-surface/50 px-4 py-2">
          <h3 className="text-sm font-semibold text-foreground">General</h3>
        </div>
        <div className="space-y-4 px-4 py-4">
          <div className="flex items-center justify-between">
            <Label htmlFor="auto-start">Auto Start</Label>
            <Switch
              id="auto-start"
              checked={form.auto_start}
              onCheckedChange={(v) => set("auto_start", v)}
            />
          </div>
          <div className="flex items-center justify-between">
            <Label htmlFor="tray-icon">Tray Icon</Label>
            <Switch
              id="tray-icon"
              checked={form.tray_icon}
              onCheckedChange={(v) => set("tray_icon", v)}
            />
          </div>
          <div className="flex items-center justify-between">
            <Label htmlFor="log-level">Log Level</Label>
            <Select
              value={form.log_level}
              onValueChange={(v) => set("log_level", v)}
            >
              <SelectTrigger id="log-level">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {LOG_LEVEL_OPTIONS.map((level) => (
                  <SelectItem key={level} value={level}>
                    {level}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
      </div>

      <div className="border-b border-border">
        <div className="flex items-center border-b border-border bg-surface/50 px-4 py-2">
          <h3 className="text-sm font-semibold text-foreground">Integrations</h3>
        </div>
        <div className="space-y-4 px-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="cf-token">Cloudflare Token</Label>
            <div className="relative">
              <Input
                id="cf-token"
                type={showToken ? "text" : "password"}
                value={form.cloudflare_token ?? ""}
                onChange={(e) => set("cloudflare_token", e.target.value)}
                placeholder="Optional"
                className="pr-10"
              />
              <button
                type="button"
                onClick={() => setShowToken((v) => !v)}
                className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              >
                {showToken ? (
                  <EyeOff className="size-4" />
                ) : (
                  <Eye className="size-4" />
                )}
              </button>
            </div>
          </div>
          <div className="space-y-2">
            <Label htmlFor="cf-account-id">Account ID</Label>
            <Input
              id="cf-account-id"
              value={form.cloudflare_account_id ?? ""}
              onChange={(e) => set("cloudflare_account_id", e.target.value)}
              placeholder="Optional (auto-detected from token)"
            />
          </div>
        </div>
      </div>

      {(error || saveError) && (
        <p className="px-4 pt-4 text-sm text-destructive">{saveError || error}</p>
      )}
      <div className="flex justify-end px-4 py-4">
        <Button onClick={handleSave} disabled={saving}>
          {saving ? "Saving..." : "Save & Reload"}
        </Button>
      </div>
    </div>
  );
}
