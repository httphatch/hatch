import { useState } from "react";
import { Header } from "@/components/header";
import { ProjectList } from "@/components/project-list";
import { RouteMap } from "@/components/route-map";
import { LogViewer } from "@/components/log-viewer";
import { AddProjectDialog } from "@/components/add-project-dialog";
import { EditProjectDialog } from "@/components/edit-project-dialog";
import { SettingsDialog } from "@/components/settings-dialog";
import { useProjects } from "@/hooks/use-projects";
import { useHealth } from "@/hooks/use-health";
import { useStatus } from "@/hooks/use-status";

function App() {
  const { projects, add, update, remove, toggle, loading, error } =
    useProjects();
  const { lookup } = useHealth();
  const { status } = useStatus();

  const [addOpen, setAddOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<string | null>(null);
  const [logsOpen, setLogsOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);

  function handleDelete(name: string) {
    if (confirm(`Delete project "${name}"?`)) {
      remove(name);
    }
  }

  const editProject = editTarget ? projects[editTarget] : null;

  return (
    <div className="flex min-h-screen flex-col bg-linen">
      <Header
        status={status}
        onAddProject={() => setAddOpen(true)}
        logsOpen={logsOpen}
        onToggleLogs={() => setLogsOpen((o) => !o)}
        onOpenSettings={() => setSettingsOpen(true)}
      />
      <main className="mx-auto w-full max-w-5xl flex-1 overflow-y-auto p-4">
        {loading && (
          <p className="py-16 text-center text-text-muted">Loading…</p>
        )}
        {error && (
          <p className="py-16 text-center text-destructive">{error}</p>
        )}
        {!loading && !error && (
          <>
            <ProjectList
              projects={projects}
              healthLookup={lookup}
              onToggle={toggle}
              onEdit={setEditTarget}
              onDelete={handleDelete}
              onAdd={() => setAddOpen(true)}
            />
            <RouteMap projects={projects} healthLookup={lookup} />
          </>
        )}
      </main>
      {logsOpen && <LogViewer />}
      <AddProjectDialog open={addOpen} onOpenChange={setAddOpen} onAdd={add} />
      <SettingsDialog open={settingsOpen} onOpenChange={setSettingsOpen} />
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

export default App;
