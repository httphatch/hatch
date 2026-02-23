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

  const target = service.proxy || service.command || "";
  const route = [
    service.subdomain ? `${service.subdomain}.*` : null,
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
      <div className="grid grid-cols-[auto_minmax(0,1fr)_minmax(0,1.5fr)_5rem_4rem_auto] items-center gap-x-3 py-1 text-xs">
        {/* Health */}
        <HealthDot health={health} />

        {/* Name */}
        <span className="font-medium text-foreground truncate">{name}</span>

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
          {tunnel && tunnel.running && (
            <button
              type="button"
              onClick={() => tunnel.url && safeOpenURL(tunnel.url)}
              className="text-primary hover:underline cursor-pointer"
              title={tunnel.url}
            >
              tunnel
            </button>
          )}
          {(tunnelLoading || (tunnel && tunnel.starting)) && (
            <span className="animate-pulse">...</span>
          )}
        </span>

        {/* Actions */}
        <div className="flex items-center gap-0.5">
          {process && process.running && (
            <>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button variant="ghost" size="icon-xs" disabled={actionLoading !== null} onClick={() => handleAction("stop")}>
                    <Square className={cn("size-3", actionLoading === "stop" && "animate-pulse")} />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Stop</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button variant="ghost" size="icon-xs" disabled={actionLoading !== null} onClick={() => handleAction("restart")}>
                    <RotateCw className={cn("size-3", actionLoading === "restart" && "animate-spin")} />
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
                  <Play className={cn("size-3", actionLoading === "start" && "animate-pulse")} />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Start</TooltipContent>
            </Tooltip>
          )}
          {tunnel && tunnel.running && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon-xs" disabled={tunnelLoading} onClick={() => handleTunnelAction("stop")}>
                  <CloudOff className={cn("size-3", tunnelLoading && "animate-pulse")} />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Stop tunnel</TooltipContent>
            </Tooltip>
          )}
          {service.proxy && (!tunnel || (!tunnel.running && !tunnel.starting)) && !tunnelLoading && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon-xs" onClick={() => handleTunnelAction("start")}>
                  <Cloud className="size-3" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Start tunnel</TooltipContent>
            </Tooltip>
          )}
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon-xs" onClick={() => setOutputOpen(true)}>
                <Terminal className="size-3" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Output</TooltipContent>
          </Tooltip>
          {url && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon-xs" onClick={() => safeOpenURL(url)}>
                  <ExternalLink className="size-3" />
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
