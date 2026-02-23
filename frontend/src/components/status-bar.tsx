import type { DaemonStatus } from "@/types";

interface StatusBarProps {
  status: DaemonStatus | null;
}

export function StatusBar({ status }: StatusBarProps) {
  return (
    <footer className="flex items-center gap-4 border-t border-border bg-surface px-4 py-1.5 font-mono text-xs text-muted-foreground">
      <span className="flex items-center gap-1.5">
        <span className={`inline-block size-2 ${status ? "bg-status-healthy" : "bg-status-error"}`} />
        {status ? "Running" : "Stopped"}
      </span>
      {status && (
        <>
          <span>{status.version}</span>
          <span>up {status.uptime}</span>
        </>
      )}
    </footer>
  );
}
