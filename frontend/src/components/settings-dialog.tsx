import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { SettingsForm } from "@/components/settings-form";
import { ConfigEditor } from "@/components/config-editor";
import { CertStatus } from "@/components/cert-status";

interface SettingsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

type Tab = "settings" | "advanced" | "certificates";

export function SettingsDialog({ open, onOpenChange }: SettingsDialogProps) {
  const [tab, setTab] = useState<Tab>("settings");

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-3xl max-h-[90vh] flex flex-col">
        <DialogHeader>
          <DialogTitle className="text-muted-teal">Settings</DialogTitle>
        </DialogHeader>
        <div className="flex gap-1 border-b border-border pb-0">
          <Button
            size="sm"
            variant="ghost"
            onClick={() => setTab("settings")}
            className={cn(
              "rounded-b-none",
              tab === "settings" && "bg-secondary"
            )}
          >
            Settings
          </Button>
          <Button
            size="sm"
            variant="ghost"
            onClick={() => setTab("advanced")}
            className={cn(
              "rounded-b-none",
              tab === "advanced" && "bg-secondary"
            )}
          >
            Advanced
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
        <div className="min-h-[300px] overflow-y-auto flex-1">
          {tab === "settings" && <SettingsForm />}
          {tab === "advanced" && <ConfigEditor />}
          {tab === "certificates" && <CertStatus />}
        </div>
      </DialogContent>
    </Dialog>
  );
}
