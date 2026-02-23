import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useCerts } from "@/hooks/use-certs";
import type { CertInfo } from "@/types";

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, {
    year: "numeric",
    month: "long",
    day: "numeric",
  });
}

function CertCard({
  label,
  info,
  showTrust,
}: {
  label: string;
  info: CertInfo;
  showTrust?: boolean;
}) {
  return (
    <Card className="gap-4 py-4">
      <CardHeader className="pb-0">
        <CardTitle className="flex items-center gap-2 text-sm">
          {label}
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
        </CardTitle>
      </CardHeader>
      {info.exists && (
        <CardContent className="space-y-1 text-sm">
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
        </CardContent>
      )}
    </Card>
  );
}

export function CertsView() {
  const { certs, loading, error } = useCerts();

  return (
    <div className="mx-auto w-full max-w-5xl">
      <div className="flex items-center border-b border-border bg-surface/50 px-4 py-2">
        <h2 className="text-lg font-semibold text-foreground">Certificates</h2>
      </div>
      {loading && (
        <p className="py-16 text-center text-muted-foreground">
          Loading certificates...
        </p>
      )}
      {error && <p className="py-16 text-center text-destructive">{error}</p>}
      {!loading && !error && certs && (
        <div className="space-y-3 p-4">
          <CertCard label="Root CA" info={certs.root_ca} showTrust />
          <CertCard label="Intermediate CA" info={certs.intermediate_ca} />
          <p className="text-xs text-muted-foreground">
            Run{" "}
            <code className="bg-secondary px-1 py-0.5 font-mono">hatch trust</code>{" "}
            to update certificate trust (requires admin privileges).
          </p>
        </div>
      )}
    </div>
  );
}
