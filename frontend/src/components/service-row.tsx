import { Badge } from "@/components/ui/badge";
import { HealthDot } from "@/components/health-dot";
import type { Service, ServiceHealth } from "@/types";

interface ServiceRowProps {
  name: string;
  service: Service;
  health?: ServiceHealth;
}

export function ServiceRow({ name, service, health }: ServiceRowProps) {
  return (
    <div className="flex items-center gap-2 text-sm py-1">
      <HealthDot health={health} />
      <span className="font-medium text-text-primary">{name}</span>
      <span className="text-text-muted">{service.proxy}</span>
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
    </div>
  );
}
