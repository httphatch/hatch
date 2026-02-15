export interface Service {
  proxy: string;
  route?: string;
  subdomain?: string;
  websocket?: boolean;
}

export interface Project {
  domain: string;
  path: string;
  enabled: boolean;
  services: Record<string, Service>;
}

export interface DaemonStatus {
  pid: number;
  uptime: string;
  version: string;
}

export interface ServiceHealth {
  project: string;
  service: string;
  status: "healthy" | "unhealthy" | "unknown";
  addr: string;
  since: string;
  last_check: string;
}

export interface LogEntry {
  id: number;
  timestamp: string;
  level: string;
  message: string;
  fields: Record<string, unknown>;
}

export interface CertInfo {
  exists: boolean;
  subject?: string;
  not_after?: string;
  trusted?: boolean;
}

export interface CertStatus {
  root_ca: CertInfo;
  intermediate_ca: CertInfo;
}
