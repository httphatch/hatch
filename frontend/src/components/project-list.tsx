import { ProjectCard } from "@/components/project-card";
import { EmptyState } from "@/components/empty-state";
import type { Project } from "@/types";

interface ProjectListProps {
  projects: Record<string, Project>;
  onToggle: (name: string) => void;
  onEdit: (name: string) => void;
  onDelete: (name: string) => void;
  onAdd: () => void;
}

export function ProjectList({
  projects,
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
    <div className="space-y-4">
      {entries.map(([name, project]) => (
        <ProjectCard
          key={name}
          name={name}
          project={project}
          onToggle={onToggle}
          onEdit={onEdit}
          onDelete={onDelete}
        />
      ))}
    </div>
  );
}
