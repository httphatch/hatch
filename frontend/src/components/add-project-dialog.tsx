import { useEffect, useState } from "react";
import { toast } from "sonner";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import type { Project } from "@/types";

interface AddProjectDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onAdd: (name: string, project: Project) => Promise<void>;
}

function emptyForm() {
  return {
    name: "",
    domain: "",
    path: "",
    serviceName: "web",
    proxy: "localhost:3000",
    command: "",
    envFile: "",
    route: "",
    subdomain: "",
    websocket: false,
  };
}

export function AddProjectDialog({
  open,
  onOpenChange,
  onAdd,
}: AddProjectDialogProps) {
  const [form, setForm] = useState(emptyForm);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (open) {
      setForm(emptyForm());
      setError(null);
    }
  }, [open]);

  function set<K extends keyof typeof form>(key: K, value: (typeof form)[K]) {
    setForm((prev) => {
      const next = { ...prev, [key]: value };
      if (key === "name" && !prev.domain) {
        next.domain = `${String(value)}.test`;
      }
      return next;
    });
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setError(null);

    const domain = form.domain || `${form.name}.test`;
    if (!/^[a-z0-9.-]+$/i.test(domain)) {
      setError("Domain must contain only letters, numbers, dots, and hyphens");
      setSubmitting(false);
      return;
    }
    if (!form.proxy && !form.command) {
      setError("Each service must have at least one of proxy or command");
      setSubmitting(false);
      return;
    }
    if (form.route && !form.route.startsWith("/")) {
      setError("Route must start with /");
      setSubmitting(false);
      return;
    }
    if (form.subdomain && !/^[a-z0-9-]+$/i.test(form.subdomain)) {
      setError("Subdomain must contain only letters, numbers, and hyphens");
      setSubmitting(false);
      return;
    }

    const project: Project = {
      domain,
      path: form.path,
      enabled: true,
      services: {
        [form.serviceName]: {
          ...(form.proxy ? { proxy: form.proxy } : {}),
          ...(form.command ? { command: form.command } : {}),
          ...(form.envFile ? { env_file: form.envFile } : {}),
          ...(form.route ? { route: form.route } : {}),
          ...(form.subdomain ? { subdomain: form.subdomain } : {}),
          ...(form.websocket ? { websocket: true } : {}),
        },
      },
    };

    try {
      await onAdd(form.name, project);
      setForm(emptyForm());
      onOpenChange(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to add project");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Add Project</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">Project name</Label>
            <Input
              id="name"
              value={form.name}
              onChange={(e) => set("name", e.target.value)}
              placeholder="my-app"
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="domain">Domain</Label>
            <Input
              id="domain"
              value={form.domain}
              onChange={(e) => set("domain", e.target.value)}
              placeholder={form.name ? `${form.name}.test` : "my-app.test"}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="path">Project path</Label>
            <Input
              id="path"
              value={form.path}
              onChange={(e) => set("path", e.target.value)}
              placeholder="/Users/you/projects/my-app"
              required
            />
          </div>
          <fieldset className="space-y-3 border border-border p-3">
            <legend className="px-1 text-sm font-medium">
              Initial service
            </legend>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-2">
                <Label htmlFor="svc-name">Name</Label>
                <Input
                  id="svc-name"
                  value={form.serviceName}
                  onChange={(e) => set("serviceName", e.target.value)}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="proxy">Proxy</Label>
                <Input
                  id="proxy"
                  value={form.proxy}
                  onChange={(e) => set("proxy", e.target.value)}
                  placeholder="localhost:3000"
                />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-2">
                <Label htmlFor="command">Command</Label>
                <Input
                  id="command"
                  value={form.command}
                  onChange={(e) => set("command", e.target.value)}
                  placeholder="npm start"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="env-file">Env file</Label>
                <Input
                  id="env-file"
                  value={form.envFile}
                  onChange={(e) => set("envFile", e.target.value)}
                  placeholder=".env"
                />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-2">
                <Label htmlFor="route">Route</Label>
                <Input
                  id="route"
                  value={form.route}
                  onChange={(e) => set("route", e.target.value)}
                  placeholder="/api/*"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="subdomain">Subdomain</Label>
                <Input
                  id="subdomain"
                  value={form.subdomain}
                  onChange={(e) => set("subdomain", e.target.value)}
                  placeholder="api"
                />
              </div>
            </div>
            <div className="flex items-center gap-2">
              <Switch
                id="ws"
                checked={form.websocket}
                onCheckedChange={(v) => set("websocket", v)}
              />
              <Label htmlFor="ws">WebSocket</Label>
            </div>
          </fieldset>
          {error && <p className="text-sm text-destructive">{error}</p>}
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={submitting}>
              {submitting ? "Adding..." : "Add Project"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
