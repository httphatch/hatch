import { Badge } from "@/components/ui/badge";
import { SectionHeader } from "@/components/section-header";
import type { CertInfo } from "@/types";

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, {
    year: "numeric",
    month: "long",
    day: "numeric",
  });
}

interface CertSectionProps {
  label: string;
  info: CertInfo;
  showTrust?: boolean;
}

export function CertSection({ label, info, showTrust }: CertSectionProps) {
  return (
    <div className="border-b border-border">
      <SectionHeader title={label} as="h3">
        <div className="flex items-center gap-2">
          {info.exists ? (
            <Badge variant="default" className="bg-status-healthy text-background">
              Installed
            </Badge>
          ) : (
            <Badge variant="destructive">Missing</Badge>
          )}
          {showTrust && info.trusted !== undefined && (
            info.trusted ? (
              <Badge variant="default" className="bg-status-healthy text-background">
                Trusted
              </Badge>
            ) : (
              <Badge variant="destructive">Not Trusted</Badge>
            )
          )}
        </div>
      </SectionHeader>
      {info.exists && (
        <div className="space-y-1 px-4 py-3 text-sm">
          {info.subject && (
            <p>
              <span className="text-muted-foreground">Subject:</span>{" "}
              <span className="font-mono">{info.subject}</span>
            </p>
          )}
          {info.not_after && (
            <p>
              <span className="text-muted-foreground">Expires:</span>{" "}
              {formatDate(info.not_after)}
            </p>
          )}
        </div>
      )}
    </div>
  );
}
