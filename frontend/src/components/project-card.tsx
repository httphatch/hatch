import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { ServiceRow } from "@/components/service-row";
import type { ProcessStatus, Project, ServiceHealth, TunnelStatus } from "@/types";
import { Copy, ExternalLink, Pencil, Trash2 } from "lucide-react";
import { cn, safeOpenURL } from "@/lib/utils";

interface ProjectCardProps {
  name: string;
  project: Project;
  healthLookup: (project: string, service: string) => ServiceHealth | undefined;
  processLookup: (project: string, service: string) => ProcessStatus | undefined;
  onToggle: (name: string) => void;
  onEdit: (name: string) => void;
  onDelete: (name: string) => void;
  tunnelLookup: (project: string, service: string) => TunnelStatus | undefined;
  onProcessRefresh: () => void;
  onTunnelRefresh: () => void;
}

export function ProjectCard({
  name,
  project,
  healthLookup,
  processLookup,
  onToggle,
  onEdit,
  onDelete,
  tunnelLookup,
  onProcessRefresh,
  onTunnelRefresh,
}: ProjectCardProps) {
  const url = `https://${project.domain}`;

  return (
    <div className={cn("py-3", !project.enabled && "opacity-50")}>
      {/* Project header */}
      <div className="flex items-center gap-3 mb-1">
        <Switch
          size="sm"
          checked={project.enabled}
          onCheckedChange={() => onToggle(name)}
          aria-label={`Toggle ${name}`}
        />
        <span className="text-sm font-semibold text-primary">{name}</span>
        <span className="font-mono text-xs text-muted-foreground truncate">{project.domain}</span>
        <div className="ml-auto flex items-center gap-0.5">
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon-xs" onClick={() => navigator.clipboard.writeText(url)}>
                <Copy className="size-3" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Copy URL</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon-xs" onClick={() => safeOpenURL(url)}>
                <ExternalLink className="size-3" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Open in browser</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon-xs" onClick={() => onEdit(name)}>
                <Pencil className="size-3" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Edit</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon-xs" onClick={() => onDelete(name)}>
                <Trash2 className="size-3 text-destructive" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Delete</TooltipContent>
          </Tooltip>
        </div>
      </div>

      {/* Services */}
      <div className="ml-9">
        {Object.entries(project.services).map(([svcName, svc]) => (
          <ServiceRow
            key={svcName}
            name={svcName}
            service={svc}
            health={healthLookup(name, svcName)}
            domain={project.domain}
            process={processLookup(name, svcName)}
            tunnel={tunnelLookup(name, svcName)}
            projectName={name}
            onRefresh={onProcessRefresh}
            onTunnelRefresh={onTunnelRefresh}
          />
        ))}
      </div>
    </div>
  );
}
