import type { Service } from "@/types";

export function serviceUrl(domain: string, service: Service): string | null {
  if (!service.proxy) return null;

  const host = service.subdomain
    ? `${service.subdomain}.${domain}`
    : domain;

  try {
    const u = new URL(`https://${host}`);
    if (service.route) u.pathname = service.route;
    return u.toString();
  } catch {
    return null;
  }
}
