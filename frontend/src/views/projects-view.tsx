import { useState } from "react";
import { ProjectList } from "@/components/project-list";
import { AddProjectDialog } from "@/components/add-project-dialog";
import { EditProjectDialog } from "@/components/edit-project-dialog";
import { Button } from "@/components/ui/button";
import { useProjects } from "@/hooks/use-projects";
import { useHealth } from "@/hooks/use-health";
import { useProcesses } from "@/hooks/use-processes";
import { useTunnels } from "@/hooks/use-tunnels";
import { Plus } from "lucide-react";

export function ProjectsView() {
  const { projects, add, update, remove, toggle, loading, error } =
    useProjects();
  const { lookup } = useHealth();
  const { lookup: processLookup, refresh: refreshProcesses } = useProcesses();
  const { lookup: tunnelLookup, refresh: refreshTunnels } = useTunnels();

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
      <div className="flex items-center justify-between border-b border-border bg-surface/50 px-4 py-2">
        <h2 className="text-lg font-semibold text-foreground">Projects</h2>
        <Button size="sm" onClick={() => setAddOpen(true)}>
          <Plus />
          Add Project
        </Button>
      </div>
      {loading && (
        <p className="py-16 text-center text-muted-foreground">Loading...</p>
      )}
      {error && (
        <p className="py-16 text-center text-destructive">{error}</p>
      )}
      {!loading && !error && (
        <ProjectList
          projects={projects}
          healthLookup={lookup}
          processLookup={processLookup}
          onToggle={toggle}
          onEdit={setEditTarget}
          onDelete={handleDelete}
          onAdd={() => setAddOpen(true)}
          tunnelLookup={tunnelLookup}
          onProcessRefresh={refreshProcesses}
          onTunnelRefresh={refreshTunnels}
        />
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
