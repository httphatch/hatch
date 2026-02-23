import { useState } from "react";
import { Toaster } from "sonner";
import { TabNav, type Tab } from "@/components/tab-nav";
import { StatusBar } from "@/components/status-bar";
import { ProjectsView } from "@/views/projects-view";
import { LogsView } from "@/views/logs-view";
import { SettingsView } from "@/views/settings-view";
import { useStatus } from "@/hooks/use-status";

function App() {
  const [tab, setTab] = useState<Tab>("projects");
  const { status } = useStatus();

  return (
    <div className="flex h-screen flex-col bg-background">
      <TabNav active={tab} onChange={setTab} />
      <main className="flex-1 overflow-y-auto">
        {tab === "projects" && <ProjectsView />}
        {tab === "logs" && <LogsView />}
        {tab === "settings" && <SettingsView />}
      </main>
      <StatusBar status={status} />
      <Toaster
        theme="dark"
        toastOptions={{
          style: {
            borderRadius: "0px",
            background: "#1a1a1a",
            border: "1px solid #262626",
            color: "#e5e5e5",
          },
        }}
      />
    </div>
  );
}

export default App;
