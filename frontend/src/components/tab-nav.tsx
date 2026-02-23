import { cn } from "@/lib/utils";

export type Tab = "projects" | "routes" | "logs" | "certs" | "settings";

const TABS: { id: Tab; label: string }[] = [
  { id: "projects", label: "Projects" },
  { id: "routes", label: "Routes" },
  { id: "logs", label: "Logs" },
  { id: "certs", label: "Certs" },
  { id: "settings", label: "Settings" },
];

interface TabNavProps {
  active: Tab;
  onChange: (tab: Tab) => void;
}

export function TabNav({ active, onChange }: TabNavProps) {
  return (
    <nav
      className="flex items-center border-b border-border bg-surface"
      style={{ "--wails-draggable": "drag" } as React.CSSProperties}
    >
      <span className="shrink-0 pl-24 pr-6 font-mono text-sm font-bold uppercase tracking-widest text-primary">
        HATCH
      </span>
      <div
        className="flex items-center"
        style={{ "--wails-draggable": "no-drag" } as React.CSSProperties}
      >
        {TABS.map((tab) => (
          <button
            key={tab.id}
            onClick={() => onChange(tab.id)}
            className={cn(
              "px-4 py-3 text-sm font-medium transition-colors",
              active === tab.id
                ? "border-b-2 border-primary text-primary"
                : "text-muted-foreground hover:text-foreground"
            )}
          >
            {tab.label}
          </button>
        ))}
      </div>
    </nav>
  );
}
