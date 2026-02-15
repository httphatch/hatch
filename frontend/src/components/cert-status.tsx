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
            <Badge variant="secondary" className="bg-emerald-100 text-emerald-700">
              Installed
            </Badge>
          ) : (
            <Badge variant="destructive">Missing</Badge>
          )}
          {showTrust && info.trusted !== undefined && (
            info.trusted ? (
              <Badge variant="secondary" className="bg-emerald-100 text-emerald-700">
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
              <span className="text-text-muted">Subject:</span> {info.subject}
            </p>
          )}
          {info.not_after && (
            <p>
              <span className="text-text-muted">Expires:</span>{" "}
              {formatDate(info.not_after)}
            </p>
          )}
        </CardContent>
      )}
    </Card>
  );
}

export function CertStatus() {
  const { certs, loading, error } = useCerts();

  if (loading) {
    return (
      <p className="py-8 text-center text-text-muted">
        Loading certificates…
      </p>
    );
  }

  if (error) {
    return <p className="py-8 text-center text-destructive">{error}</p>;
  }

  if (!certs) return null;

  return (
    <div className="space-y-3">
      <CertCard label="Root CA" info={certs.root_ca} showTrust />
      <CertCard label="Intermediate CA" info={certs.intermediate_ca} />
      <p className="text-xs text-text-muted">
        Run <code className="rounded bg-secondary px-1 py-0.5">hatch trust</code> to
        update certificate trust (requires admin privileges).
      </p>
    </div>
  );
}
