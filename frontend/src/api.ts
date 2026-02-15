import type { CertStatus, DaemonStatus, Project, ServiceHealth } from "./types";

const BASE = "http://127.0.0.1:42824";

async function request<T>(
  path: string,
  options?: RequestInit
): Promise<T> {
  const res = await fetch(`${BASE}${path}`, options);
  if (!res.ok) {
    const text = await res.text().catch(() => res.statusText);
    throw new Error(`${res.status}: ${text}`);
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}

export function getStatus(): Promise<DaemonStatus> {
  return request("/api/status");
}

export function getProjects(): Promise<Record<string, Project>> {
  return request("/api/projects");
}

export function addProject(
  name: string,
  project: Project
): Promise<Project> {
  return request("/api/projects", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name, project }),
  });
}

export function updateProject(
  name: string,
  project: Project
): Promise<Project> {
  return request(`/api/projects/${encodeURIComponent(name)}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(project),
  });
}

export function deleteProject(name: string): Promise<void> {
  return request(`/api/projects/${encodeURIComponent(name)}`, {
    method: "DELETE",
  });
}

export function toggleProject(
  name: string
): Promise<{ enabled: boolean }> {
  return request(`/api/projects/${encodeURIComponent(name)}/toggle`, {
    method: "PATCH",
  });
}

export function getHealth(): Promise<ServiceHealth[]> {
  return request("/api/health");
}

export function restartDaemon(): Promise<{ status: string }> {
  return request("/api/restart", { method: "POST" });
}

export async function getConfig(): Promise<string> {
  const res = await fetch(`${BASE}/api/config`);
  if (!res.ok) {
    const text = await res.text().catch(() => res.statusText);
    throw new Error(`${res.status}: ${text}`);
  }
  return res.text();
}

export async function updateConfig(yaml: string): Promise<void> {
  const res = await fetch(`${BASE}/api/config`, {
    method: "PUT",
    headers: { "Content-Type": "application/yaml" },
    body: yaml,
  });
  if (!res.ok) {
    const text = await res.text().catch(() => res.statusText);
    throw new Error(`${res.status}: ${text}`);
  }
}

export function getCerts(): Promise<CertStatus> {
  return request("/api/certs");
}
