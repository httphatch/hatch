import { useCallback, useEffect, useRef, useState } from "react";
import type { LogEntry } from "@/types";

const MAX_ENTRIES = 1000;
const SSE_URL = "http://127.0.0.1:42824/api/logs";
const RECONNECT_DELAY = 3000;

function parseEntry(line: string, idRef: React.RefObject<number>): LogEntry | null {
  // SSE lines look like "data: {...json...}"
  const stripped = line.startsWith("data: ") ? line.slice(6) : line;
  if (!stripped) return null;
  try {
    const data = JSON.parse(stripped);
    const { time, timestamp, level, message, ...fields } = data;
    return {
      id: ++idRef.current,
      timestamp: time ?? timestamp ?? "",
      level: level ?? "info",
      message: message ?? "",
      fields,
    };
  } catch {
    return null;
  }
}

export function useLogs(enabled: boolean) {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [connected, setConnected] = useState(false);
  const idRef = useRef(0);

  useEffect(() => {
    if (!enabled) return;

    let cancelled = false;
    let reconnectTimer: ReturnType<typeof setTimeout>;

    async function connect() {
      if (cancelled) return;

      try {
        const res = await fetch(SSE_URL);
        if (cancelled) return;

        if (!res.ok || !res.body) {
          throw new Error("bad response");
        }

        setConnected(true);

        const reader = res.body.getReader();
        const decoder = new TextDecoder();
        let buffer = "";

        while (true) {
          const { done, value } = await reader.read();
          if (done || cancelled) break;

          buffer += decoder.decode(value, { stream: true });
          const lines = buffer.split("\n");
          // Keep the last (possibly incomplete) chunk
          buffer = lines.pop() ?? "";

          for (const line of lines) {
            const trimmed = line.trim();
            if (!trimmed) continue;
            const entry = parseEntry(trimmed, idRef);
            if (entry) {
              setLogs((prev) => {
                const next = [...prev, entry];
                return next.length > MAX_ENTRIES ? next.slice(-MAX_ENTRIES) : next;
              });
            }
          }
        }
      } catch {
        // connection failed or dropped
      }

      if (!cancelled) {
        setConnected(false);
        reconnectTimer = setTimeout(connect, RECONNECT_DELAY);
      }
    }

    connect();

    return () => {
      cancelled = true;
      clearTimeout(reconnectTimer);
      setConnected(false);
    };
  }, [enabled]);

  const clear = useCallback(() => setLogs([]), []);

  return { logs, connected, clear };
}
