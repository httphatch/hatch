import { useCallback, useEffect, useRef, useState } from "react";
import type { OutputLine } from "@/types";

const MAX_LINES = 1000;
const RECONNECT_DELAY = 3000;

function parseOutputLine(
  raw: string,
  idRef: React.RefObject<number>,
): (OutputLine & { id: number }) | null {
  const stripped = raw.startsWith("data: ") ? raw.slice(6) : raw;
  if (!stripped) return null;
  try {
    const data = JSON.parse(stripped) as OutputLine;
    return { ...data, id: ++idRef.current };
  } catch {
    return null;
  }
}

export function useProcessOutput(
  project: string,
  service: string,
  enabled: boolean,
) {
  const [lines, setLines] = useState<(OutputLine & { id: number })[]>([]);
  const [connected, setConnected] = useState(false);
  const idRef = useRef(0);

  useEffect(() => {
    if (!enabled) return;

    let cancelled = false;
    let reconnectTimer: ReturnType<typeof setTimeout>;

    async function connect() {
      if (cancelled) return;

      try {
        const url = `/api/processes/output?project=${encodeURIComponent(project)}&service=${encodeURIComponent(service)}`;
        const res = await fetch(url);
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
          const parts = buffer.split("\n");
          buffer = parts.pop() ?? "";

          for (const part of parts) {
            const trimmed = part.trim();
            if (!trimmed) continue;
            const entry = parseOutputLine(trimmed, idRef);
            if (entry) {
              setLines((prev) => {
                const next = [...prev, entry];
                return next.length > MAX_LINES ? next.slice(-MAX_LINES) : next;
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
  }, [enabled, project, service]);

  const clear = useCallback(() => setLines([]), []);

  return { lines, connected, clear };
}
