import { useEffect, useRef } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { useProcessOutput } from "@/hooks/use-process-output";
import { cn } from "@/lib/utils";

interface ProcessOutputDialogProps {
  project: string;
  service: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ProcessOutputDialog({
  project,
  service,
  open,
  onOpenChange,
}: ProcessOutputDialogProps) {
  const { lines, connected, clear } = useProcessOutput(project, service, open);
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [lines.length]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl max-h-[80vh] flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <span className="font-mono">
              {project}/{service}
            </span>
            <span
              className={cn(
                "inline-block size-2",
                connected ? "bg-status-healthy" : "bg-status-error",
              )}
            />
          </DialogTitle>
        </DialogHeader>
        <div className="flex-1 min-h-0 overflow-auto border border-border bg-background p-3 font-mono text-xs leading-relaxed">
          {lines.length === 0 && (
            <p className="text-muted-foreground">No output yet.</p>
          )}
          {lines.map((line) => (
            <div
              key={line.id}
              className={cn(
                "whitespace-pre-wrap break-all",
                line.source === "stderr" ? "text-status-error" : "text-foreground",
              )}
            >
              {line.line}
            </div>
          ))}
          <div ref={bottomRef} />
        </div>
        <DialogFooter>
          <Button variant="outline" size="sm" onClick={clear}>
            Clear
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
