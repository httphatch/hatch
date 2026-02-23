import { useMemo } from "react";
import { HealthDot } from "@/components/health-dot";
import { serviceUrl } from "@/lib/service-url";
import { useProjects } from "@/hooks/use-projects";
import { useHealth } from "@/hooks/use-health";
import type { Project } from "@/types";
import { safeOpenURL } from "@/lib/utils";

interface RouteEntry {
  domain: string;
  route: string;
  target: string;
  url: string;
  project: string;
  service: string;
}

function buildRoutes(projects: Record<string, Project>): RouteEntry[] {
  const routes: RouteEntry[] = [];
  for (const [name, proj] of Object.entries(projects)) {
    if (!proj.enabled) continue;
    for (const [svcName, svc] of Object.entries(proj.services)) {
      const url = serviceUrl(proj.domain, svc);
      if (!url) continue;
      const domain = svc.subdomain
        ? `${svc.subdomain}.${proj.domain}`
        : proj.domain;
      routes.push({
        domain,
        route: svc.route || "/",
        target: svc.proxy ?? "",
        url,
        project: name,
        service: svcName,
      });
    }
  }
  return routes.sort((a, b) => a.domain.localeCompare(b.domain));
}

export function RoutesView() {
  const { projects } = useProjects();
  const { lookup } = useHealth();
  const routes = useMemo(() => buildRoutes(projects), [projects]);

  return (
    <div className="mx-auto w-full max-w-5xl p-4">
      <h2 className="mb-4 text-lg font-semibold text-foreground">Routes</h2>
      {routes.length === 0 ? (
        <p className="py-16 text-center text-muted-foreground">
          No active routes. Enable a project to see its routes here.
        </p>
      ) : (
        <div className="overflow-x-auto border border-border bg-card">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border text-left text-xs text-muted-foreground">
                <th className="px-3 py-2 font-medium">Domain</th>
                <th className="px-3 py-2 font-medium">Route</th>
                <th className="px-3 py-2 font-medium">Target</th>
                <th className="px-3 py-2 font-medium w-8">Health</th>
              </tr>
            </thead>
            <tbody>
              {routes.map((r) => (
                <tr
                  key={`${r.domain}-${r.route}-${r.service}`}
                  className="border-b border-border last:border-0"
                >
                  <td className="px-3 py-2 font-mono">
                    <button
                      type="button"
                      onClick={() => safeOpenURL(r.url)}
                      className="hover:text-primary hover:underline"
                    >
                      {r.domain}
                    </button>
                  </td>
                  <td className="px-3 py-2 font-mono text-muted-foreground">
                    {r.route}
                  </td>
                  <td className="px-3 py-2 font-mono text-muted-foreground">
                    {r.target}
                  </td>
                  <td className="px-3 py-2">
                    <HealthDot health={lookup(r.project, r.service)} />
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
