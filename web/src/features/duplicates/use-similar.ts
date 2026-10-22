"use client";
import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { readSimilar, similarQuery } from "@/lib/team";
import { useProjectScope } from "../projects/project-context";

// Debounced live lookup while a person types a new task. A failed lookup is
// silent: the server still checks duplicates authoritatively on create.
export function useSimilar(fields: {
  title: string;
  description?: string;
  type?: string;
  role?: string;
  enabled: boolean;
}) {
  const scope = useProjectScope();
  const [query, setQuery] = useState("");
  const raw =
    fields.enabled && fields.title.trim().length >= 4
      ? similarQuery(fields)
      : "";
  useEffect(() => {
    const id = window.setTimeout(() => setQuery(raw), 450);
    return () => window.clearTimeout(id);
  }, [raw]);
  return useQuery({
    queryKey: ["similar", scope.key, query],
    queryFn: ({ signal }) => readSimilar(signal, scope.id, query),
    enabled: Boolean(query) && query === raw,
    retry: false,
    staleTime: 15000,
    gcTime: 60000,
    placeholderData: (previous) => previous,
  });
}
