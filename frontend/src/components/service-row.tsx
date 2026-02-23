import { useState } from "react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
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

  return (
    <div className="flex items-center gap-2 text-sm py-1">
      <HealthDot health={health} />
      <span className="font-medium text-foreground">{name}</span>
      {service.proxy && (
        <span className="font-mono text-muted-foreground">{service.proxy}</span>
      )}
      {!service.proxy && service.command && (
        <span className="font-mono text-muted-foreground truncate max-w-48" title={service.command}>
          {service.command}
        </span>
      )}
      {service.route && (
        <Badge variant="outline" className="font-mono text-xs">
          {service.route}
        </Badge>
      )}
      {service.subdomain && (
        <Badge variant="secondary" className="font-mono text-xs">
          {service.subdomain}.*
        </Badge>
      )}
      {service.websocket && (
        <Badge variant="outline" className="font-mono text-xs">
          WS
        </Badge>
      )}
      {process && (
        <Badge
          variant="outline"
          className={`font-mono text-xs ${process.running ? "text-status-healthy border-status-healthy" : "text-status-error border-status-error"}`}
        >
          {process.running ? "Running" : "Stopped"}
        </Badge>
      )}
      {process && process.restarts > 0 && (
        <Badge variant="outline" className="font-mono text-xs">
          {process.restarts} restart{process.restarts !== 1 ? "s" : ""}
        </Badge>
      )}
      {process && process.running && (
        <>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon-xs"
                disabled={actionLoading !== null}
                onClick={() => handleAction("stop")}
              >
                <Square className={actionLoading === "stop" ? "animate-pulse" : ""} />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Stop</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon-xs"
                disabled={actionLoading !== null}
                onClick={() => handleAction("restart")}
              >
                <RotateCw className={actionLoading === "restart" ? "animate-spin" : ""} />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Restart</TooltipContent>
          </Tooltip>
        </>
      )}
      {process && process.stopped && (
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon-xs"
              disabled={actionLoading !== null}
              onClick={() => handleAction("start")}
            >
              <Play className={actionLoading === "start" ? "animate-pulse" : ""} />
            </Button>
          </TooltipTrigger>
          <TooltipContent>Start</TooltipContent>
        </Tooltip>
      )}
      {tunnel && tunnel.running && (
        <>
          {tunnel.url && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Badge
                  variant="outline"
                  className="font-mono text-xs text-primary border-primary cursor-pointer hover:bg-primary/10"
                  onClick={() => safeOpenURL(tunnel.url)}
                >
                  <Cloud className="mr-1 size-3" />
                  {tunnel.url.replace(/^https?:\/\//, "")}
                </Badge>
              </TooltipTrigger>
              <TooltipContent>Open tunnel URL</TooltipContent>
            </Tooltip>
          )}
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon-xs"
                disabled={tunnelLoading}
                onClick={() => handleTunnelAction("stop")}
              >
                <CloudOff className={tunnelLoading ? "animate-pulse" : ""} />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Stop tunnel</TooltipContent>
          </Tooltip>
        </>
      )}
      {service.proxy && (!tunnel || (!tunnel.running && !tunnel.starting)) && !tunnelLoading && (
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon-xs"
              onClick={() => handleTunnelAction("start")}
            >
              <Cloud />
            </Button>
          </TooltipTrigger>
          <TooltipContent>Start tunnel</TooltipContent>
        </Tooltip>
      )}
      {(tunnelLoading || (tunnel && tunnel.starting)) && (
        <Badge variant="outline" className="font-mono text-xs text-muted-foreground border-muted-foreground animate-pulse">
          <Cloud className="mr-1 size-3" />
          Starting tunnel...
        </Badge>
      )}
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="ghost"
            size="icon-xs"
            onClick={() => setOutputOpen(true)}
          >
            <Terminal />
          </Button>
        </TooltipTrigger>
        <TooltipContent>View output</TooltipContent>
      </Tooltip>
      {url && (
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon-xs"
              onClick={() => safeOpenURL(url)}
            >
              <ExternalLink />
            </Button>
          </TooltipTrigger>
          <TooltipContent>Open {url}</TooltipContent>
        </Tooltip>
      )}
      <ProcessOutputDialog
        project={projectName}
        service={name}
        open={outputOpen}
        onOpenChange={setOutputOpen}
      />
    </div>
  );
}
