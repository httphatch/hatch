import { Button } from "@/components/ui/button";
import { FolderPlus } from "lucide-react";

interface EmptyStateProps {
  onAdd: () => void;
}

export function EmptyState({ onAdd }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <FolderPlus className="size-12 text-muted-teal mb-4" />
      <h2 className="text-xl font-semibold text-muted-teal mb-2">
        No projects yet
      </h2>
      <p className="text-text-muted mb-6">
        Add a project to get started with local HTTPS development.
      </p>
      <Button onClick={onAdd}>Add Project</Button>
    </div>
  );
}
