import { useCallback, useEffect, useState } from "react";
import * as api from "@/api";
import type { Project } from "@/types";

export function useProjects() {
  const [projects, setProjects] = useState<Record<string, Project>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    try {
      const data = await api.getProjects();
      setProjects(data ?? {});
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load projects");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const add = useCallback(
    async (name: string, project: Project) => {
      await api.addProject(name, project);
      await refresh();
    },
    [refresh]
  );

  const update = useCallback(
    async (name: string, project: Project) => {
      await api.updateProject(name, project);
      await refresh();
    },
    [refresh]
  );

  const remove = useCallback(
    async (name: string) => {
      await api.deleteProject(name);
      await refresh();
    },
    [refresh]
  );

  const toggle = useCallback(
    async (name: string) => {
      await api.toggleProject(name);
      await refresh();
    },
    [refresh]
  );

  return { projects, loading, error, add, update, remove, toggle, refresh };
}
