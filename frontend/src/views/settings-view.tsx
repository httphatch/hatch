import { useState } from "react";
import { cn } from "@/lib/utils";
import { RoutesView } from "@/views/routes-view";
import { CertsView } from "@/views/certs-view";
import { SettingsForm } from "@/components/settings-form";
import { ConfigEditor } from "@/components/config-editor";

type Section = "routes" | "certificates" | "general" | "advanced";

const SECTIONS: { id: Section; label: string }[] = [
  { id: "routes", label: "Routes" },
  { id: "certificates", label: "Certificates" },
  { id: "general", label: "General" },
  { id: "advanced", label: "Advanced" },
];

const SECTION_TITLES: Record<Section, string> = {
  routes: "Routes",
  certificates: "Certificates",
  general: "Settings",
  advanced: "Advanced Configuration",
};

function SectionHeader({ title }: { title: string }) {
  return (
    <div className="flex items-center border-b border-border bg-surface/50 px-4 py-2">
      <h2 className="text-lg font-semibold text-foreground">{title}</h2>
    </div>
  );
}

export function SettingsView() {
  const [section, setSection] = useState<Section>("general");

  return (
    <div className="flex h-full overflow-hidden">
      <nav className="w-48 shrink-0 border-r border-border bg-surface/30 p-2">
        {SECTIONS.map((s) => (
          <button
            key={s.id}
            type="button"
            onClick={() => setSection(s.id)}
            className={cn(
              "w-full rounded px-3 py-1.5 text-left text-sm transition-colors",
              section === s.id
                ? "bg-surface/50 text-foreground"
                : "text-muted-foreground hover:text-foreground"
            )}
          >
            {s.label}
          </button>
        ))}
      </nav>
      <div className="flex-1 overflow-y-auto">
        <div className="mx-auto w-full max-w-5xl">
          <SectionHeader title={SECTION_TITLES[section]} />
          {section === "routes" && <RoutesView />}
          {section === "certificates" && <CertsView />}
          {section === "general" && <SettingsForm />}
          {section === "advanced" && (
            <div className="px-4 py-4">
              <ConfigEditor />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
