import { useMemo, useState } from "react";
import { Button } from "@/components/ui/button";
import { HealthDot } from "@/components/health-dot";
import type { Project, ServiceHealth } from "@/types";
import { ChevronDown, ChevronRight, Map } from "lucide-react";

interface RouteMapProps {
  projects: Record<string, Project>;
  healthLookup: (project: string, service: string) => ServiceHealth | undefined;
}

interface RouteEntry {
  domain: string;
  route: string;
  target: string;
  project: string;
  service: string;
}

function buildRoutes(projects: Record<string, Project>): RouteEntry[] {
  const routes: RouteEntry[] = [];
  for (const [name, proj] of Object.entries(projects)) {
    if (!proj.enabled) continue;
    for (const [svcName, svc] of Object.entries(proj.services)) {
      const domain = svc.subdomain
        ? `${svc.subdomain}.${proj.domain}`
        : proj.domain;
      routes.push({
        domain,
        route: svc.route || "/",
        target: svc.proxy,
        project: name,
        service: svcName,
      });
    }
  }
  return routes.sort((a, b) => a.domain.localeCompare(b.domain));
}

export function RouteMap({ projects, healthLookup }: RouteMapProps) {
  const [open, setOpen] = useState(false);
  const routes = useMemo(() => buildRoutes(projects), [projects]);

  if (routes.length === 0) return null;

  return (
    <div className="mt-6">
      <Button
        variant="ghost"
        size="sm"
        onClick={() => setOpen(!open)}
        className="text-text-muted"
      >
        {open ? <ChevronDown /> : <ChevronRight />}
        <Map className="size-4" />
        Route Map ({routes.length})
      </Button>
      {open && (
        <div className="mt-2 overflow-x-auto rounded-lg border border-border bg-card">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border text-left text-xs text-text-muted">
                <th className="px-3 py-2">Domain</th>
                <th className="px-3 py-2">Route</th>
                <th className="px-3 py-2">Target</th>
                <th className="px-3 py-2 w-8">Health</th>
              </tr>
            </thead>
            <tbody>
              {routes.map((r) => (
                <tr
                  key={`${r.domain}-${r.route}-${r.service}`}
                  className="border-b border-border last:border-0"
                >
                  <td className="px-3 py-2 font-medium">{r.domain}</td>
                  <td className="px-3 py-2 text-text-muted">{r.route}</td>
                  <td className="px-3 py-2 text-text-muted">{r.target}</td>
                  <td className="px-3 py-2">
                    <HealthDot health={healthLookup(r.project, r.service)} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
