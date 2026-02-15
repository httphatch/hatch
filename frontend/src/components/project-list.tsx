import { ProjectCard } from "@/components/project-card";
import { EmptyState } from "@/components/empty-state";
import type { Project, ServiceHealth } from "@/types";

interface ProjectListProps {
  projects: Record<string, Project>;
  healthLookup: (project: string, service: string) => ServiceHealth | undefined;
  onToggle: (name: string) => void;
  onEdit: (name: string) => void;
  onDelete: (name: string) => void;
  onAdd: () => void;
}

export function ProjectList({
  projects,
  healthLookup,
  onToggle,
  onEdit,
  onDelete,
  onAdd,
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
          onToggle={onToggle}
          onEdit={onEdit}
          onDelete={onDelete}
        />
      ))}
    </div>
  );
}
