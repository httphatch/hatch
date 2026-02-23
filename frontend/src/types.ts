export interface Service {
  proxy?: string;
  route?: string;
  subdomain?: string;
  websocket?: boolean;
  command?: string;
  env_file?: string;
  tunnel?: string;
}

export interface ProcessStatus {
  project: string;
  service: string;
  command: string;
  running: boolean;
  stopped: boolean;
  restarts: number;
  started_at: string;
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

export interface OutputLine {
  project: string;
  service: string;
  source: "stdout" | "stderr";
  line: string;
  time: string;
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

export interface Settings {
  tld: string;
  http_port: number;
  https_port: number;
  auto_start: boolean;
  tray_icon: boolean;
  log_level: string;
  cloudflare_token?: string;
  cloudflare_account_id?: string;
}

export interface TunnelStatus {
  project: string;
  service: string;
  running: boolean;
  starting: boolean;
  url?: string;
  type: "quick" | "named";
  started_at: string;
  error?: string;
}
