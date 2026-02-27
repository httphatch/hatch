import { createContext, useContext, useMemo, type ReactNode } from "react";
import { useHealth } from "@/hooks/use-health";
import { useProcesses } from "@/hooks/use-processes";
import { useTunnels } from "@/hooks/use-tunnels";
import type { ProcessStatus, ServiceHealth, TunnelStatus } from "@/types";

interface ProjectServicesContextValue {
  healthLookup: (project: string, service: string) => ServiceHealth | undefined;
  processLookup: (project: string, service: string) => ProcessStatus | undefined;
  tunnelLookup: (project: string, service: string) => TunnelStatus | undefined;
  refreshProcesses: () => void;
  refreshTunnels: () => void;
}

const ProjectServicesContext = createContext<ProjectServicesContextValue | null>(null);

export function ProjectServicesProvider({ children }: { children: ReactNode }) {
  const { lookup: healthLookup } = useHealth();
  const { lookup: processLookup, refresh: refreshProcesses } = useProcesses();
  const { lookup: tunnelLookup, refresh: refreshTunnels } = useTunnels();

  const value = useMemo(
    () => ({ healthLookup, processLookup, tunnelLookup, refreshProcesses, refreshTunnels }),
    [healthLookup, processLookup, tunnelLookup, refreshProcesses, refreshTunnels]
  );

  return (
    <ProjectServicesContext.Provider value={value}>
      {children}
    </ProjectServicesContext.Provider>
  );
}

export function useProjectServices(): ProjectServicesContextValue {
  const ctx = useContext(ProjectServicesContext);
  if (!ctx) {
    throw new Error("useProjectServices must be used within a ProjectServicesProvider");
  }
  return ctx;
}
