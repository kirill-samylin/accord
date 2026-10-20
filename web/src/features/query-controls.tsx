"use client";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { ArrowLeft, ChevronRight, Filter, Search, X } from "lucide-react";
import { useState, type ReactNode } from "react";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { label } from "@/lib/format";
import { useProjectScope } from "./projects/project-context";
export function useFilters() {
  const scope = useProjectScope();
  const params = useSearchParams();
  const path = usePathname();
  const router = useRouter();
  function set(values: Record<string, string | undefined>, reset = true) {
    if (scope.locked) return;
    const next = new URLSearchParams(params.toString());
    if (reset) {
      next.delete("cursor");
      next.delete("epic_cursor");
    }
    for (const [key, value] of Object.entries(values)) {
      if (value) next.set(key, value);
      else next.delete(key);
    }
    router.push(`${path}${next.size ? `?${next}` : ""}`);
  }
  return {
    params,
    path,
    set,
    reset: () => {
      if (!scope.locked) router.push(path);
    },
  };
}
export function SearchField({
  value = "",
  onSearch,
}: {
  value?: string;
  onSearch: (value: string) => void;
}) {
  const [query, setQuery] = useState(value);
  return (
    <form
      className="search-input"
      onSubmit={(event) => {
        event.preventDefault();
        onSearch(query);
      }}
    >
      <Search size={14} />
      <input
        aria-label="Поиск задач"
        placeholder="Поиск задач…"
        maxLength={200}
        value={query}
        onChange={(event) => setQuery(event.target.value)}
      />
    </form>
  );
}
export function FilterMenu({
  children,
  count = 0,
}: {
  children: ReactNode;
  count?: number;
}) {
  return (
    <Popover>
      <PopoverTrigger asChild>
        <button className="button button-secondary button-sm">
          <Filter size={14} />
          Фильтры{count ? <span className="nav-count">{count}</span> : null}
        </button>
      </PopoverTrigger>
      <PopoverContent>
        <div className="mutation-form">{children}</div>
      </PopoverContent>
    </Popover>
  );
}
export function SelectFilter({
  title,
  value,
  values,
  onChange,
}: {
  title: string;
  value?: string;
  values: string[] | { value: string; title: string }[];
  onChange: (value: string) => void;
}) {
  return (
    <label className="form-field">
      {title}
      <select
        value={value ?? ""}
        onChange={(event) => onChange(event.target.value)}
      >
        <option value="">Все</option>
        {values.map((value) =>
          typeof value === "string" ? (
            <option key={value} value={value}>
              {label(value)}
            </option>
          ) : (
            <option key={value.value} value={value.value}>
              {value.title}
            </option>
          ),
        )}
      </select>
    </label>
  );
}
export function Pagination({
  cursor,
  epicCursor,
}: {
  cursor?: string;
  epicCursor?: string;
}) {
  const filters = useFilters();
  const current =
    filters.params.get("cursor") ?? filters.params.get("epic_cursor");
  if (!cursor && !epicCursor && !current) return null;
  return (
    <div className="pagination">
      <span className="muted">
        {current ? "Продолжение списка" : "Показана первая страница"}
      </span>
      <div className="toolbar-end">
        {current ? (
          <button
            className="button button-secondary button-sm"
            onClick={() =>
              filters.set({ cursor: undefined, epic_cursor: undefined })
            }
          >
            <ArrowLeft size={14} />В начало
          </button>
        ) : null}
        {cursor ? (
          <button
            className="button button-secondary button-sm"
            onClick={() => filters.set({ cursor }, false)}
          >
            Следующая страница
            <ChevronRight size={14} />
          </button>
        ) : null}
        {epicCursor ? (
          <button
            className="button button-secondary button-sm"
            onClick={() => filters.set({ epic_cursor: epicCursor }, false)}
          >
            Ещё эпики
            <ChevronRight size={14} />
          </button>
        ) : null}
      </div>
    </div>
  );
}
export function ClearFilters() {
  const filters = useFilters();
  return (
    <button className="button button-ghost button-sm" onClick={filters.reset}>
      <X size={13} />
      Сбросить
    </button>
  );
}
