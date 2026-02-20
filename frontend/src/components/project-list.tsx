import { ProjectCard } from "@/components/project-card";
import { EmptyState } from "@/components/empty-state";
import type { ProcessStatus, Project, ServiceHealth, TunnelStatus } from "@/types";

interface ProjectListProps {
  projects: Record<string, Project>;
  healthLookup: (project: string, service: string) => ServiceHealth | undefined;
  processLookup: (project: string, service: string) => ProcessStatus | undefined;
  onToggle: (name: string) => void;
  onEdit: (name: string) => void;
  onDelete: (name: string) => void;
  onAdd: () => void;
  tunnelLookup: (project: string, service: string) => TunnelStatus | undefined;
  onProcessRefresh: () => void;
  onTunnelRefresh: () => void;
}

export function ProjectList({
  projects,
  healthLookup,
  processLookup,
  onToggle,
  onEdit,
  onDelete,
  onAdd,
  tunnelLookup,
  onProcessRefresh,
  onTunnelRefresh,
}: ProjectListProps) {
  const entries = Object.entries(projects);

  if (entries.length === 0) {
    return <EmptyState onAdd={onAdd} />;
  }

  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
      {entries.map(([name, project]) => (
        <ProjectCard
          key={name}
          name={name}
          project={project}
          healthLookup={healthLookup}
          processLookup={processLookup}
          onToggle={onToggle}
          onEdit={onEdit}
          onDelete={onDelete}
          tunnelLookup={tunnelLookup}
          onProcessRefresh={onProcessRefresh}
          onTunnelRefresh={onTunnelRefresh}
        />
      ))}
    </div>
  );
}
