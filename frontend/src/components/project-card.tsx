import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { ServiceRow } from "@/components/service-row";
import { useProjectServices } from "@/hooks/use-project-services";
import type { Project } from "@/types";
import { Copy, ExternalLink, Pencil, Trash2 } from "lucide-react";
import { cn, safeOpenURL } from "@/lib/utils";

interface ProjectCardProps {
  name: string;
  project: Project;
  onToggle: (name: string) => void;
  onEdit: (name: string) => void;
  onDelete: (name: string) => void;
}

export function ProjectCard({
  name,
  project,
  onToggle,
  onEdit,
  onDelete,
}: ProjectCardProps) {
  const { healthLookup, processLookup, tunnelLookup } = useProjectServices();
  const url = `https://${project.domain}`;

  return (
    <div className={cn(!project.enabled && "opacity-50")}>
      {/* Project header */}
      <div className="flex items-center gap-3 border-y border-border bg-surface/50 px-3 py-2">
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
                <Copy className="size-3.5" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Copy URL</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon-xs" onClick={() => safeOpenURL(url)}>
                <ExternalLink className="size-3.5" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Open in browser</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon-xs" onClick={() => onEdit(name)}>
                <Pencil className="size-3.5" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Edit</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon-xs" onClick={() => onDelete(name)}>
                <Trash2 className="size-3.5 text-destructive" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Delete</TooltipContent>
          </Tooltip>
        </div>
      </div>

      {/* Services */}
      <div className="ml-6 mt-2 px-2 py-1">
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
          />
        ))}
      </div>
    </div>
  );
}
