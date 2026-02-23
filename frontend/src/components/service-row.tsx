import { useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { HealthDot } from "@/components/health-dot";
import { ProcessOutputDialog } from "@/components/process-output-dialog";
import { serviceUrl } from "@/lib/service-url";
import { stopProcess, startProcess, restartProcess, startTunnel, stopTunnel } from "@/api";
import type { ProcessStatus, Service, ServiceHealth, TunnelStatus } from "@/types";
import { Cloud, CloudOff, ExternalLink, Play, RotateCw, Square, Terminal } from "lucide-react";
import { safeOpenURL } from "@/lib/utils";
import { cn } from "@/lib/utils";

interface ServiceRowProps {
  name: string;
  service: Service;
  health?: ServiceHealth;
  domain: string;
  process?: ProcessStatus;
  tunnel?: TunnelStatus;
  projectName: string;
  onRefresh: () => void;
  onTunnelRefresh: () => void;
}

export function ServiceRow({ name, service, health, domain, process, tunnel, projectName, onRefresh, onTunnelRefresh }: ServiceRowProps) {
  const url = serviceUrl(domain, service);
  const [outputOpen, setOutputOpen] = useState(false);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [tunnelLoading, setTunnelLoading] = useState(false);

  async function handleTunnelAction(action: "start" | "stop") {
    setTunnelLoading(true);
    try {
      if (action === "start") await startTunnel(projectName, name);
      else await stopTunnel(projectName, name);
      onTunnelRefresh();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Tunnel action failed");
    } finally {
      setTunnelLoading(false);
    }
  }

  async function handleAction(action: "stop" | "start" | "restart") {
    setActionLoading(action);
    try {
      if (action === "stop") await stopProcess(projectName, name);
      else if (action === "start") await startProcess(projectName, name);
      else await restartProcess(projectName, name);
      onRefresh();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : `Failed to ${action} process`);
    } finally {
      setActionLoading(null);
    }
  }

  const isProxy = Boolean(service.proxy);
  const target = service.proxy || service.command || "";
  const upstreamLabel = isProxy ? "proxy" : "cmd";
  const route = [
    service.route,
    service.websocket ? "ws" : null,
  ].filter(Boolean).join(" ");

  const statusText = process
    ? process.running
      ? "running"
      : "stopped"
    : "";
  const statusClass = process
    ? process.running
      ? "text-status-healthy"
      : "text-status-error"
    : "";

  return (
    <>
      <div className="grid grid-cols-[0.625rem_minmax(0,1fr)_4.5rem_minmax(0,1.5fr)_6.5rem_5.5rem_8rem] items-center gap-x-3 py-2 text-xs border-b border-border/50 last:border-b-0">
        {/* Health */}
        <HealthDot health={health} />

        {/* Name + subdomain */}
        <div className="min-w-0">
          <span className="font-medium text-foreground truncate block">{name}</span>
          {service.subdomain && (
            <span className="text-[10px] text-muted-foreground/70 truncate block">{service.subdomain}.{domain}</span>
          )}
        </div>

        {/* Upstream label */}
        <span className={cn(
          "text-[10px] font-medium uppercase tracking-wider text-center px-1.5 py-0.5 rounded inline-flex items-center justify-center gap-1",
          isProxy ? "text-blue-400 bg-blue-400/10" : "text-amber-400 bg-amber-400/10"
        )}>
          {upstreamLabel} <span className="opacity-60">&rarr;</span>
        </span>

        {/* Target */}
        <span className="font-mono text-muted-foreground truncate" title={target}>
          {target}
          {route && (
            <span className="ml-1.5 text-muted-foreground/60">{route}</span>
          )}
        </span>

        {/* Status */}
        <span className={cn("font-mono truncate", statusClass)}>
          {statusText}
          {process && process.restarts > 0 && (
            <span className="text-muted-foreground"> ({process.restarts})</span>
          )}
        </span>

        {/* Tunnel */}
        <span className="font-mono text-muted-foreground truncate text-center">
          {tunnel && tunnel.running && tunnel.url && (
            <button
              type="button"
              onClick={() => tunnel.url && safeOpenURL(tunnel.url)}
              className="text-primary hover:underline cursor-pointer truncate block"
              title={tunnel.url}
            >
              {tunnel.url.replace(/^https?:\/\//, "")}
            </button>
          )}
          {tunnel && tunnel.running && !tunnel.url && (
            <button
              type="button"
              onClick={() => handleTunnelAction("stop")}
              className="text-primary hover:underline cursor-pointer"
              title="Tunnel active (click to stop)"
            >
              tunnel active
            </button>
          )}
          {(tunnelLoading || (tunnel && tunnel.starting)) && !tunnel?.running && (
            <span className="animate-pulse">starting</span>
          )}
        </span>

        {/* Actions */}
        <div className="flex items-center gap-0.5">
          {process && process.running && (
            <>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button variant="ghost" size="icon-xs" disabled={actionLoading !== null} onClick={() => handleAction("stop")}>
                    <Square className={cn("size-3.5", actionLoading === "stop" && "animate-pulse")} />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Stop</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button variant="ghost" size="icon-xs" disabled={actionLoading !== null} onClick={() => handleAction("restart")}>
                    <RotateCw className={cn("size-3.5", actionLoading === "restart" && "animate-spin")} />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Restart</TooltipContent>
              </Tooltip>
            </>
          )}
          {process && process.stopped && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon-xs" disabled={actionLoading !== null} onClick={() => handleAction("start")}>
                  <Play className={cn("size-3.5", actionLoading === "start" && "animate-pulse")} />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Start</TooltipContent>
            </Tooltip>
          )}
          {tunnel && tunnel.running && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon-xs" disabled={tunnelLoading} onClick={() => handleTunnelAction("stop")}>
                  <CloudOff className={cn("size-3.5", tunnelLoading && "animate-pulse")} />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Stop tunnel</TooltipContent>
            </Tooltip>
          )}
          {service.proxy && (!tunnel || (!tunnel.running && !tunnel.starting)) && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon-xs" disabled={tunnelLoading} onClick={() => handleTunnelAction("start")}>
                  <Cloud className={cn("size-3.5", tunnelLoading && "animate-pulse")} />
                </Button>
              </TooltipTrigger>
              <TooltipContent>{tunnelLoading ? "Starting tunnel..." : "Start tunnel"}</TooltipContent>
            </Tooltip>
          )}
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon-xs" onClick={() => setOutputOpen(true)}>
                <Terminal className="size-3.5" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Output</TooltipContent>
          </Tooltip>
          {url && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon-xs" onClick={() => safeOpenURL(url)}>
                  <ExternalLink className="size-3.5" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Open</TooltipContent>
            </Tooltip>
          )}
        </div>
      </div>
      <ProcessOutputDialog
        project={projectName}
        service={name}
        open={outputOpen}
        onOpenChange={setOutputOpen}
      />
    </>
  );
}
