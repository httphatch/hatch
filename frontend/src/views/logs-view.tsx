import { useCallback, useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useLogs } from "@/hooks/use-logs";
import { ArrowDown, Trash2 } from "lucide-react";
import { cn } from "@/lib/utils";

const LEVELS = ["debug", "info", "warn", "error"] as const;

const levelColors: Record<string, string> = {
  debug: "text-muted-foreground",
  info: "text-primary",
  warn: "text-status-warning",
  error: "text-status-error",
};

const levelToggleStyles: Record<string, { active: string; inactive: string }> = {
  debug: {
    active: "bg-muted text-foreground border-border",
    inactive: "bg-transparent text-muted-foreground border-border opacity-40",
  },
  info: {
    active: "bg-primary/15 text-primary border-primary/30",
    inactive: "bg-transparent text-primary/50 border-border opacity-40",
  },
  warn: {
    active: "bg-status-warning/15 text-status-warning border-status-warning/30",
    inactive: "bg-transparent text-status-warning/50 border-border opacity-40",
  },
  error: {
    active: "bg-status-error/15 text-status-error border-status-error/30",
    inactive: "bg-transparent text-status-error/50 border-border opacity-40",
  },
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

function getSource(fields: Record<string, unknown>): string {
  if (fields.component) return String(fields.component);
  if (fields.source) return String(fields.source);
  if (fields.module) return String(fields.module);
  return "";
}

export function LogsView() {
  const { logs, connected, clear } = useLogs(true);
  const [enabledLevels, setEnabledLevels] = useState<Set<string>>(
    () => new Set(LEVELS)
  );
  const [search, setSearch] = useState("");
  const scrollRef = useRef<HTMLDivElement>(null);
  const [autoScroll, setAutoScroll] = useState(true);

  const toggleLevel = useCallback((level: string) => {
    setEnabledLevels((prev) => {
      const next = new Set(prev);
      if (next.has(level)) next.delete(level);
      else next.add(level);
      return next;
    });
  }, []);

  const filtered = logs.filter((entry) => {
    if (!enabledLevels.has(entry.level)) return false;
    if (search) {
      const q = search.toLowerCase();
      if (entry.message.toLowerCase().includes(q)) return true;
      for (const v of Object.values(entry.fields)) {
        if (String(v).toLowerCase().includes(q)) return true;
      }
      return false;
    }
    return true;
  });

  const handleScroll = useCallback(() => {
    const el = scrollRef.current;
    if (!el) return;
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 40;
    setAutoScroll(atBottom);
  }, []);

  useEffect(() => {
    if (!autoScroll) return;
    const el = scrollRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [logs.length, autoScroll]);

  const jumpToLatest = useCallback(() => {
    const el = scrollRef.current;
    if (el) el.scrollTop = el.scrollHeight;
    setAutoScroll(true);
  }, []);

  return (
    <div className="flex h-full flex-col">
      {/* Toolbar */}
      <div className="flex items-center gap-2 border-b border-border bg-surface px-4 py-2">
        <div className="flex items-center gap-1">
          {LEVELS.map((level) => {
            const active = enabledLevels.has(level);
            const styles = levelToggleStyles[level];
            return (
              <button
                key={level}
                onClick={() => toggleLevel(level)}
                aria-pressed={active}
                className={cn(
                  "border px-2 py-0.5 text-[10px] font-medium uppercase transition-opacity cursor-pointer",
                  active ? styles.active : styles.inactive
                )}
              >
                {level}
              </button>
            );
          })}
        </div>
        <Input
          placeholder="Search logs..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="h-7 max-w-48 text-xs"
        />
        <Button variant="ghost" size="sm" onClick={clear} className="h-7 px-2">
          <Trash2 className="size-3.5" />
        </Button>
        <div className="ml-auto flex items-center gap-1.5 text-xs text-muted-foreground">
          <span
            className={cn(
              "inline-block size-2",
              connected ? "bg-status-healthy" : "bg-status-error"
            )}
          />
          {connected ? "Connected" : "Disconnected"}
        </div>
      </div>

      {/* Log table */}
      <div className="relative flex-1 min-h-0">
        <div
          ref={scrollRef}
          onScroll={handleScroll}
          className="h-full overflow-y-auto"
        >
          <table className="w-full font-mono text-xs">
            <thead className="sticky top-0 bg-surface text-left text-muted-foreground">
              <tr className="border-b border-border">
                <th className="w-24 shrink-0 px-3 py-1.5 font-medium">Time</th>
                <th className="w-14 px-2 py-1.5 font-medium">Level</th>
                <th className="w-28 px-2 py-1.5 font-medium">Source</th>
                <th className="px-3 py-1.5 font-medium">Message</th>
              </tr>
            </thead>
            <tbody>
              {filtered.length === 0 ? (
                <tr>
                  <td colSpan={4} className="py-8 text-center text-muted-foreground">
                    No log entries
                  </td>
                </tr>
              ) : (
                filtered.map((entry) => (
                  <tr
                    key={entry.id}
                    className="border-b border-border/50 hover:bg-surface-alt"
                  >
                    <td className="whitespace-nowrap px-3 py-0.5 text-muted-foreground">
                      {formatTime(entry.timestamp)}
                    </td>
                    <td className={cn("px-2 py-0.5 uppercase", levelColors[entry.level])}>
                      {entry.level}
                    </td>
                    <td className="truncate px-2 py-0.5 text-muted-foreground">
                      {getSource(entry.fields)}
                    </td>
                    <td className="px-3 py-0.5 text-foreground">
                      {entry.message}
                      {Object.keys(entry.fields).length > 0 && (
                        <span className="ml-2 text-muted-foreground">
                          {Object.entries(entry.fields)
                            .filter(([k]) => k !== "component" && k !== "source" && k !== "module")
                            .map(([k, v]) => `${k}=${String(v)}`)
                            .join(" ")}
                        </span>
                      )}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
        {!autoScroll && (
          <Button
            size="sm"
            variant="secondary"
            onClick={jumpToLatest}
            className="absolute right-4 bottom-3 h-7 gap-1 text-xs shadow-md"
          >
            <ArrowDown className="size-3.5" />
            Jump to latest
          </Button>
        )}
      </div>
    </div>
  );
}
