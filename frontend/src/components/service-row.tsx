import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { HealthDot } from "@/components/health-dot";
import { serviceUrl } from "@/lib/service-url";
import type { ProcessStatus, Service, ServiceHealth } from "@/types";
import { ExternalLink } from "lucide-react";
import { Browser } from "@wailsio/runtime";

interface ServiceRowProps {
  name: string;
  service: Service;
  health?: ServiceHealth;
  domain: string;
  process?: ProcessStatus;
}

export function ServiceRow({ name, service, health, domain, process }: ServiceRowProps) {
  const url = serviceUrl(domain, service);

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
    </div>
  );
}
