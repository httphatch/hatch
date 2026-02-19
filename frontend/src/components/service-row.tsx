import { useState } from "react";
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
import { stopProcess, startProcess, restartProcess } from "@/api";
import type { ProcessStatus, Service, ServiceHealth } from "@/types";
import { ExternalLink, Play, RotateCw, Square, Terminal } from "lucide-react";
import { Browser } from "@wailsio/runtime";

interface ServiceRowProps {
  name: string;
  service: Service;
  health?: ServiceHealth;
  domain: string;
  process?: ProcessStatus;
  projectName: string;
  onRefresh: () => void;
}

export function ServiceRow({ name, service, health, domain, process, projectName, onRefresh }: ServiceRowProps) {
  const url = serviceUrl(domain, service);
  const [outputOpen, setOutputOpen] = useState(false);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  async function handleAction(action: "stop" | "start" | "restart") {
    setActionLoading(action);
    try {
      if (action === "stop") await stopProcess(projectName, name);
      else if (action === "start") await startProcess(projectName, name);
      else await restartProcess(projectName, name);
      onRefresh();
    } finally {
      setActionLoading(null);
    }
  }

  return (
    <div className="flex items-center gap-2 text-sm py-1">
      <HealthDot health={health} />
      <span className="font-medium text-text-primary">{name}</span>
      {service.proxy && (
        <span className="text-text-muted">{service.proxy}</span>
      )}
      {!service.proxy && service.command && (
        <span className="text-text-muted truncate max-w-48" title={service.command}>
          {service.command}
        </span>
      )}
      {service.route && (
        <Badge variant="outline" className="text-xs">
          {service.route}
        </Badge>
      )}
      {service.subdomain && (
        <Badge variant="secondary" className="text-xs">
          {service.subdomain}.*
        </Badge>
      )}
      {service.websocket && (
        <Badge variant="outline" className="text-xs">
          WS
        </Badge>
      )}
      {process && (
        <Badge
          variant="outline"
          className={`text-xs ${process.running ? "text-muted-teal border-muted-teal" : "text-light-coral border-light-coral"}`}
        >
          {process.running ? "Running" : "Stopped"}
        </Badge>
      )}
      {process && process.restarts > 0 && (
        <Badge variant="outline" className="text-xs">
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
              onClick={() => Browser.OpenURL(url)}
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
