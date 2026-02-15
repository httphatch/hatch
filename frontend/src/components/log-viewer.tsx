import { useCallback, useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { LogEntryRow } from "@/components/log-entry";
import { useLogs } from "@/hooks/use-logs";
import { ArrowDown, Trash2 } from "lucide-react";
import { cn } from "@/lib/utils";

const LEVELS = ["debug", "info", "warn", "error"] as const;

const levelToggleStyles: Record<string, { active: string; inactive: string }> = {
  debug: {
    active: "bg-muted text-text-primary border-border",
    inactive: "bg-transparent text-text-muted border-border opacity-40",
  },
  info: {
    active: "bg-muted-teal/15 text-muted-teal border-muted-teal/30",
    inactive: "bg-transparent text-muted-teal/50 border-border opacity-40",
  },
  warn: {
    active: "bg-honey-bronze/15 text-honey-bronze border-honey-bronze/30",
    inactive: "bg-transparent text-honey-bronze/50 border-border opacity-40",
  },
  error: {
    active: "bg-light-coral/15 text-light-coral border-light-coral/30",
    inactive: "bg-transparent text-light-coral/50 border-border opacity-40",
  },
};

export function LogViewer() {
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
    <div className="border-t border-border bg-card/80 backdrop-blur-sm">
      {/* Toolbar */}
      <div className="flex items-center gap-2 border-b border-border px-4 py-2">
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
                  "rounded-full border px-2 py-0.5 text-[10px] font-medium uppercase transition-opacity cursor-pointer",
                  active ? styles.active : styles.inactive
                )}
              >
                {level}
              </button>
            );
          })}
        </div>
        <Input
          placeholder="Search logs…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="h-7 max-w-48 text-xs"
        />
        <Button variant="ghost" size="sm" onClick={clear} className="h-7 px-2">
          <Trash2 className="size-3.5" />
        </Button>
        <div className="ml-auto flex items-center gap-1.5 text-xs text-text-muted">
          <span
            className={cn(
              "inline-block size-2 rounded-full",
              connected ? "bg-muted-teal" : "bg-light-coral"
            )}
          />
          {connected ? "Connected" : "Disconnected"}
        </div>
      </div>

      {/* Log area */}
      <div className="relative">
        <div
          ref={scrollRef}
          onScroll={handleScroll}
          className="h-80 overflow-y-auto font-mono text-xs"
        >
          {filtered.length === 0 ? (
            <p className="py-8 text-center text-text-muted">No log entries</p>
          ) : (
            filtered.map((entry) => (
              <LogEntryRow key={entry.id} entry={entry} />
            ))
          )}
        </div>
        {!autoScroll && (
          <Button
            size="sm"
            variant="secondary"
            onClick={jumpToLatest}
            className="absolute right-4 bottom-3 h-7 gap-1 text-xs shadow-md"
          >
            <ArrowDown className="size-3" />
            Jump to latest
          </Button>
        )}
      </div>
    </div>
  );
}
