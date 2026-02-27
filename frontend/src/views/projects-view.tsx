import { useState } from "react";
import { ProjectList } from "@/components/project-list";
import { AddProjectDialog } from "@/components/add-project-dialog";
import { EditProjectDialog } from "@/components/edit-project-dialog";
import { SectionHeader } from "@/components/section-header";
import { LoadingState } from "@/components/loading-state";
import { ErrorState } from "@/components/error-state";
import { Button } from "@/components/ui/button";
import { useProjects } from "@/hooks/use-projects";
import { ProjectServicesProvider } from "@/hooks/use-project-services";
import { Plus } from "lucide-react";

export function ProjectsView() {
  const { projects, add, update, remove, toggle, loading, error } =
    useProjects();

  const [addOpen, setAddOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<string | null>(null);

  function handleDelete(name: string) {
    if (confirm(`Delete project "${name}"?`)) {
      remove(name);
    }
  }

  const editProject = editTarget ? projects[editTarget] : null;

  return (
    <div className="mx-auto w-full max-w-5xl">
      <SectionHeader title="Projects">
        <Button size="sm" onClick={() => setAddOpen(true)}>
          <Plus />
          Add Project
        </Button>
      </SectionHeader>
      {loading && <LoadingState />}
      {error && <ErrorState message={error} />}
      {!loading && !error && (
        <ProjectServicesProvider>
          <ProjectList
            projects={projects}
            onToggle={toggle}
            onEdit={setEditTarget}
            onDelete={handleDelete}
            onAdd={() => setAddOpen(true)}
          />
        </ProjectServicesProvider>
      )}
      <AddProjectDialog open={addOpen} onOpenChange={setAddOpen} onAdd={add} />
      {editTarget && editProject && (
        <EditProjectDialog
          open={!!editTarget}
          onOpenChange={(open) => {
            if (!open) setEditTarget(null);
          }}
          name={editTarget}
          project={editProject}
          onSave={update}
        />
      )}
    </div>
  );
}
