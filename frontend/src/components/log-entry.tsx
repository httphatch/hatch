import { Badge } from "@/components/ui/badge";
import type { LogEntry } from "@/types";
import { cn } from "@/lib/utils";

const levelStyles: Record<string, string> = {
  debug: "border-border text-text-muted",
  info: "bg-muted-teal/15 text-muted-teal border-muted-teal/30",
  warn: "bg-honey-bronze/15 text-honey-bronze border-honey-bronze/30",
  error: "bg-light-coral/15 text-light-coral border-light-coral/30",
};

function formatTime(ts: string): string {
  try {
    const d = new Date(ts);
    const h = d.getHours().toString().padStart(2, "0");
    const m = d.getMinutes().toString().padStart(2, "0");
    const s = d.getSeconds().toString().padStart(2, "0");
    const ms = d.getMilliseconds().toString().padStart(3, "0");
    return `${h}:${m}:${s}.${ms}`;
  } catch {
    return ts;
  }
}

interface LogEntryRowProps {
  entry: LogEntry;
}

export function LogEntryRow({ entry }: LogEntryRowProps) {
  const fieldPairs = Object.entries(entry.fields);

  return (
    <div className="flex items-baseline gap-2 px-2 py-0.5 hover:bg-muted/50">
      <span className="shrink-0 text-text-muted">{formatTime(entry.timestamp)}</span>
      <Badge
        variant="outline"
        className={cn("shrink-0 text-[10px] uppercase", levelStyles[entry.level])}
      >
        {entry.level}
      </Badge>
      <span className="text-text-primary">{entry.message}</span>
      {fieldPairs.length > 0 && (
        <span className="text-text-muted">
          {fieldPairs.map(([k, v]) => `${k}=${String(v)}`).join(" ")}
        </span>
      )}
    </div>
  );
}
