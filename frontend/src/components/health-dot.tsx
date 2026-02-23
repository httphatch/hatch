import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import type { ServiceHealth } from "@/types";
import { cn } from "@/lib/utils";

const statusColor: Record<string, string> = {
  healthy: "bg-status-healthy",
  unhealthy: "bg-status-error",
  unknown: "bg-status-unknown",
};

interface HealthDotProps {
  health?: ServiceHealth;
}

export function HealthDot({ health }: HealthDotProps) {
  const status = health?.status ?? "unknown";
  const since = health?.since
    ? new Date(health.since).toLocaleTimeString()
    : "\u2014";

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span
          className={cn(
            "inline-block size-2.5 shrink-0",
            statusColor[status]
          )}
        />
      </TooltipTrigger>
      <TooltipContent>
        <p className="capitalize">{status}</p>
        <p className="text-xs text-muted-foreground">Since {since}</p>
      </TooltipContent>
    </Tooltip>
  );
}
