"use client";
import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { notFound, usePathname } from "next/navigation";
import {
  projectAppPath,
  projectRoute,
  projectWebPath,
  routeScope,
} from "@/lib/project-routes";
type Scope = {
  id?: string;
  key: string;
  locked: boolean;
  lock: (owner: object, value: boolean) => void;
  app: (path: string) => string;
  web: (path: string) => string;
};
const fallback: Scope = {
  key: "default",
  locked: false,
  lock: () => {},
  app: (path) => path,
  web: (path) => path,
};
const ProjectContext = createContext<Scope>(fallback);
export const useProjectScope = () => useContext(ProjectContext);
export function ProjectProvider({ children }: { children: ReactNode }) {
  const path = usePathname();
  let id: string | undefined, key: string;
  try {
    id = projectRoute(path)?.id;
    key = routeScope(path);
  } catch {
    // An unknown /app/p/{id}/... address is a missing page, not a crash.
    notFound();
  }
  const [locks, setLocks] = useState<Set<object>>(() => new Set());
  const lock = useCallback((owner: object, value: boolean) => {
    setLocks((previous) => {
      if (previous.has(owner) === value) return previous;
      const next = new Set(previous);
      if (value) next.add(owner);
      else next.delete(owner);
      return next;
    });
  }, []);
  const value = useMemo<Scope>(
    () => ({
      id,
      key,
      locked: locks.size > 0,
      lock,
      app: (value) => projectAppPath(id, value),
      web: (value) => projectWebPath(id, value),
    }),
    [id, key, locks, lock],
  );
  return (
    <ProjectContext.Provider value={value}>{children}</ProjectContext.Provider>
  );
}
