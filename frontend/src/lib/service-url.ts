import type { Service } from "@/types";

export function serviceUrl(domain: string, service: Service): string | null {
  if (!service.proxy) return null;

  const host = service.subdomain
    ? `${service.subdomain}.${domain}`
    : domain;

  const path = service.route ?? "";
  return `https://${host}${path}`;
}
