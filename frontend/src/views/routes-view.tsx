import { useMemo } from "react";
import { HealthDot } from "@/components/health-dot";
import { LoadingState } from "@/components/loading-state";
import { ErrorState } from "@/components/error-state";
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
  const { projects, loading, error } = useProjects();
  const { lookup } = useHealth();
  const routes = useMemo(() => buildRoutes(projects), [projects]);

  if (loading) {
    return <LoadingState message="Loading routes..." />;
  }

  if (error) {
    return <ErrorState message={error} />;
  }

  return routes.length === 0 ? (
    <p className="py-16 text-center text-muted-foreground">
      No active routes. Enable a project to see its routes here.
    </p>
  ) : (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border text-left text-xs text-muted-foreground">
            <th className="px-4 py-2 font-medium">Domain</th>
            <th className="px-4 py-2 font-medium">Route</th>
            <th className="px-4 py-2 font-medium">Target</th>
            <th className="px-4 py-2 font-medium w-8">Health</th>
          </tr>
        </thead>
        <tbody>
          {routes.map((r) => (
            <tr
              key={`${r.project}-${r.service}`}
              className="border-b border-border/50 last:border-0"
            >
              <td className="px-4 py-2 font-mono">
                <button
                  type="button"
                  onClick={() => safeOpenURL(r.url)}
                  className="hover:text-primary hover:underline cursor-pointer"
                >
                  {r.domain}
                </button>
              </td>
              <td className="px-4 py-2 font-mono text-muted-foreground">
                {r.route}
              </td>
              <td className="px-4 py-2 font-mono text-muted-foreground">
                {r.target}
              </td>
              <td className="px-4 py-2">
                <HealthDot health={lookup(r.project, r.service)} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
