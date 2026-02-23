import { Badge } from "@/components/ui/badge";
import { useCerts } from "@/hooks/use-certs";
import type { CertInfo } from "@/types";

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, {
    year: "numeric",
    month: "long",
    day: "numeric",
  });
}

function CertSection({
  label,
  info,
  showTrust,
}: {
  label: string;
  info: CertInfo;
  showTrust?: boolean;
}) {
  return (
    <div className="border-b border-border">
      <div className="flex items-center gap-2 border-b border-border bg-surface/50 px-4 py-2">
        <h3 className="text-sm font-semibold text-foreground">{label}</h3>
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

export function CertsView() {
  const { certs, loading, error } = useCerts();

  return (
    <>
      {loading && (
        <p className="py-16 text-center text-muted-foreground">
          Loading certificates...
        </p>
      )}
      {error && <p className="py-16 text-center text-destructive">{error}</p>}
      {!loading && !error && certs && (
        <>
          <CertSection label="Root CA" info={certs.root_ca} showTrust />
          <CertSection label="Intermediate CA" info={certs.intermediate_ca} />
          <p className="px-4 py-3 text-xs text-muted-foreground">
            Run{" "}
            <code className="bg-secondary px-1 py-0.5 font-mono">hatch trust</code>{" "}
            to update certificate trust (requires admin privileges).
          </p>
        </>
      )}
    </>
  );
}
