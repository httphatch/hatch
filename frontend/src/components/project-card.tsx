import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardAction,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { ServiceRow } from "@/components/service-row";
import type { Project, ServiceHealth } from "@/types";
import { Copy, ExternalLink, Pencil, Trash2 } from "lucide-react";
import { cn } from "@/lib/utils";

interface ProjectCardProps {
  name: string;
  project: Project;
  healthLookup: (project: string, service: string) => ServiceHealth | undefined;
  onToggle: (name: string) => void;
  onEdit: (name: string) => void;
  onDelete: (name: string) => void;
}

export function ProjectCard({
  name,
  project,
  healthLookup,
  onToggle,
  onEdit,
  onDelete,
}: ProjectCardProps) {
  const url = `https://${project.domain}`;

  function copyDomain() {
    navigator.clipboard.writeText(url);
  }

  function openInBrowser() {
    window.open(url, "_blank");
  }

  return (
    <Card className={cn(!project.enabled && "opacity-60")}>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <span className="text-lg font-semibold text-muted-teal">{name}</span>
          <Switch
            checked={project.enabled}
            onCheckedChange={() => onToggle(name)}
            aria-label={`Toggle ${name}`}
          />
        </CardTitle>
        <CardAction>
          <div className="flex gap-1">
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon-xs"
                  onClick={() => onEdit(name)}
                >
                  <Pencil />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Edit</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon-xs"
                  onClick={() => onDelete(name)}
                >
                  <Trash2 className="text-destructive" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Delete</TooltipContent>
            </Tooltip>
          </div>
        </CardAction>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="flex items-center gap-2 text-sm">
          <span className="text-text-muted truncate">{project.domain}</span>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon-xs" onClick={copyDomain}>
                <Copy />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Copy URL</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon-xs" onClick={openInBrowser}>
                <ExternalLink />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Open in browser</TooltipContent>
          </Tooltip>
        </div>
        <p className="text-xs text-text-muted truncate">{project.path}</p>
        <div>
          {Object.entries(project.services).map(([svcName, svc]) => (
            <ServiceRow
              key={svcName}
              name={svcName}
              service={svc}
              health={healthLookup(name, svcName)}
            />
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
