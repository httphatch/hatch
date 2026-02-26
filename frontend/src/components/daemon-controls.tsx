import { useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { startDaemon, stopDaemon, restartDaemon } from "@/api";
import type { DaemonStatus } from "@/types";
import { Play, RotateCw, Square } from "lucide-react";
import { cn } from "@/lib/utils";

interface DaemonControlsProps {
  status: DaemonStatus | null;
  onRefresh: () => void;
}

export function DaemonControls({ status, onRefresh }: DaemonControlsProps) {
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const running = status !== null;

  async function handleAction(action: "start" | "stop" | "restart") {
    setActionLoading(action);
    try {
      if (action === "start") await startDaemon();
      else if (action === "stop") await stopDaemon();
      else await restartDaemon();

      const delay = action === "stop" ? 1000 : 2000;
      setTimeout(() => {
        onRefresh();
        setActionLoading(null);
      }, delay);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : `Failed to ${action} daemon`);
      setActionLoading(null);
    }
  }

  if (running) {
    return (
      <div className="flex items-center gap-2">
        <Button variant="ghost" size="xs" disabled={actionLoading !== null} onClick={() => handleAction("restart")}>
          <RotateCw className={cn("size-3", actionLoading === "restart" && "animate-spin")} />
          Restart
        </Button>
        <Button variant="ghost" size="xs" disabled={actionLoading !== null} onClick={() => handleAction("stop")}>
          <Square className={cn("size-3", actionLoading === "stop" && "animate-pulse")} />
          Stop
        </Button>
      </div>
    );
  }

  return (
    <div className="flex items-center gap-2">
      <Button variant="ghost" size="xs" disabled={actionLoading !== null} onClick={() => handleAction("start")}>
        <Play className={cn("size-3", actionLoading === "start" && "animate-pulse")} />
        Start
      </Button>
    </div>
  );
}
