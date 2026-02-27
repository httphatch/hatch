import { CertSection } from "@/components/cert-section";
import { LoadingState } from "@/components/loading-state";
import { ErrorState } from "@/components/error-state";
import { useCerts } from "@/hooks/use-certs";

export function CertsView() {
  const { certs, loading, error } = useCerts();

  return (
    <>
      {loading && <LoadingState message="Loading certificates..." />}
      {error && <ErrorState message={error} />}
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
