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
import { Separator } from "@/components/ui/separator";
import type { Project, Service } from "@/types";
import { Plus, Trash2 } from "lucide-react";

interface EditProjectDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  name: string;
  project: Project;
  onSave: (name: string, project: Project) => Promise<void>;
}

interface ServiceForm {
  name: string;
  proxy: string;
  command: string;
  dir: string;
  envFile: string;
  route: string;
  subdomain: string;
  websocket: boolean;
}

function serviceToForm(name: string, svc: Service): ServiceForm {
  return {
    name,
    proxy: svc.proxy ?? "",
    command: svc.command ?? "",
    dir: svc.dir ?? "",
    envFile: svc.env_file ?? "",
    route: svc.route ?? "",
    subdomain: svc.subdomain ?? "",
    websocket: svc.websocket ?? false,
  };
}

export function EditProjectDialog({
  open,
  onOpenChange,
  name,
  project,
  onSave,
}: EditProjectDialogProps) {
  const [domain, setDomain] = useState(project.domain);
  const [path, setPath] = useState(project.path);
  const [services, setServices] = useState<ServiceForm[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (open) {
      setDomain(project.domain);
      setPath(project.path);
      setServices(
        Object.entries(project.services).map(([n, s]) => serviceToForm(n, s))
      );
      setError(null);
    }
  }, [project, open]);

  function updateService(idx: number, partial: Partial<ServiceForm>) {
    setServices((prev) =>
      prev.map((s, i) => (i === idx ? { ...s, ...partial } : s))
    );
  }

  function addService() {
    setServices((prev) => [
      ...prev,
      { name: "", proxy: "localhost:3000", command: "", dir: "", envFile: "", route: "", subdomain: "", websocket: false },
    ]);
  }

  function removeService(idx: number) {
    setServices((prev) => prev.filter((_, i) => i !== idx));
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setError(null);

    if (!/^[a-z0-9.-]+$/i.test(domain)) {
      setError("Domain must contain only letters, numbers, dots, and hyphens");
      setSubmitting(false);
      return;
    }

    const svcMap: Record<string, Service> = {};
    for (const s of services) {
      if (!s.name) {
        setError("All services must have a name");
        setSubmitting(false);
        return;
      }
      if (!s.proxy && !s.command) {
        setError(`Service "${s.name}" must have at least one of proxy or command`);
        setSubmitting(false);
        return;
      }
      if (s.route && !s.route.startsWith("/")) {
        setError(`Route for "${s.name}" must start with /`);
        setSubmitting(false);
        return;
      }
      if (s.subdomain && !/^[a-z0-9-]+$/i.test(s.subdomain)) {
        setError(`Subdomain for "${s.name}" must contain only letters, numbers, and hyphens`);
        setSubmitting(false);
        return;
      }
      svcMap[s.name] = {
        ...(s.proxy ? { proxy: s.proxy } : {}),
        ...(s.command ? { command: s.command } : {}),
        ...(s.dir ? { dir: s.dir } : {}),
        ...(s.envFile ? { env_file: s.envFile } : {}),
        ...(s.route ? { route: s.route } : {}),
        ...(s.subdomain ? { subdomain: s.subdomain } : {}),
        ...(s.websocket ? { websocket: true } : {}),
      };
    }

    try {
      await onSave(name, {
        domain,
        path,
        enabled: project.enabled,
        services: svcMap,
      });
      onOpenChange(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to save project");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Edit {name}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="edit-domain">Domain</Label>
            <Input
              id="edit-domain"
              value={domain}
              onChange={(e) => setDomain(e.target.value)}
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="edit-path">Path</Label>
            <Input
              id="edit-path"
              value={path}
              onChange={(e) => setPath(e.target.value)}
              required
            />
          </div>

          <Separator />

          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <Label>Services</Label>
              <Button
                type="button"
                variant="outline"
                size="xs"
                onClick={addService}
              >
                <Plus />
                Add
              </Button>
            </div>
            {services.map((svc, idx) => (
              <fieldset
                key={idx}
                className="space-y-3 border border-border p-3"
              >
                <div className="flex items-center justify-between">
                  <Input
                    value={svc.name}
                    onChange={(e) =>
                      updateService(idx, { name: e.target.value })
                    }
                    placeholder="Service name"
                    className="h-7 w-40 text-sm"
                    required
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-xs"
                    onClick={() => removeService(idx)}
                  >
                    <Trash2 className="text-destructive" />
                  </Button>
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-1">
                    <Label className="text-xs">Proxy</Label>
                    <Input
                      value={svc.proxy}
                      onChange={(e) =>
                        updateService(idx, { proxy: e.target.value })
                      }
                      placeholder="localhost:3000"
                    />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs">Command</Label>
                    <Input
                      value={svc.command}
                      onChange={(e) =>
                        updateService(idx, { command: e.target.value })
                      }
                      placeholder="npm start"
                    />
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-1">
                    <Label className="text-xs">Working directory</Label>
                    <Input
                      value={svc.dir}
                      onChange={(e) =>
                        updateService(idx, { dir: e.target.value })
                      }
                      placeholder="packages/api"
                    />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs">Env file</Label>
                    <Input
                      value={svc.envFile}
                      onChange={(e) =>
                        updateService(idx, { envFile: e.target.value })
                      }
                      placeholder=".env"
                    />
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-1">
                    <Label className="text-xs">Route</Label>
                    <Input
                      value={svc.route}
                      onChange={(e) =>
                        updateService(idx, { route: e.target.value })
                      }
                      placeholder="/api/*"
                    />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs">Subdomain</Label>
                    <Input
                      value={svc.subdomain}
                      onChange={(e) =>
                        updateService(idx, { subdomain: e.target.value })
                      }
                      placeholder="api"
                    />
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div className="flex items-end gap-2 pb-1">
                    <Switch
                      checked={svc.websocket}
                      onCheckedChange={(v) =>
                        updateService(idx, { websocket: v })
                      }
                    />
                    <Label className="text-xs">WebSocket</Label>
                  </div>
                </div>
              </fieldset>
            ))}
          </div>

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
              {submitting ? "Saving..." : "Save"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
