import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { ConfigEditor } from "@/components/config-editor";
import { CertStatus } from "@/components/cert-status";

interface SettingsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

type Tab = "config" | "certificates";

export function SettingsDialog({ open, onOpenChange }: SettingsDialogProps) {
  const [tab, setTab] = useState<Tab>("config");

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle className="text-muted-teal">Settings</DialogTitle>
        </DialogHeader>
        <div className="flex gap-1 border-b border-border pb-0">
          <Button
            size="sm"
            variant="ghost"
            onClick={() => setTab("config")}
            className={cn(
              "rounded-b-none",
              tab === "config" && "bg-secondary"
            )}
          >
            Config
          </Button>
          <Button
            size="sm"
            variant="ghost"
            onClick={() => setTab("certificates")}
            className={cn(
              "rounded-b-none",
              tab === "certificates" && "bg-secondary"
            )}
          >
            Certificates
          </Button>
        </div>
        <div className="min-h-[300px]">
          {tab === "config" && <ConfigEditor />}
          {tab === "certificates" && <CertStatus />}
        </div>
      </DialogContent>
    </Dialog>
  );
}
