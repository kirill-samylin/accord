"use client";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { Command } from "cmdk";
import { Search } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/components/ui/dialog";
import { navigation } from "./navigation";
import { readPage } from "@/lib/api";
export function CommandPalette({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (value: boolean) => void;
}) {
  const [query, setQuery] = useState("");
  const [search, setSearch] = useState("");
  const router = useRouter();
  useEffect(() => {
    const timer = setTimeout(() => setSearch(query.trim()), 250);
    return () => clearTimeout(timer);
  }, [query]);
  const result = useQuery({
    queryKey: ["command", search],
    queryFn: ({ signal }) =>
      readPage(`/app/tasks?q=${encodeURIComponent(search)}`, signal),
    enabled: open && search.length >= 2,
    retry: false,
    staleTime: 0,
    gcTime: 0,
  });
  function navigate(href: string) {
    onOpenChange(false);
    setQuery("");
    router.push(href);
  }
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="command-content" showCloseButton={false}>
        <DialogTitle className="sr-only">Поиск и навигация</DialogTitle>
        <DialogDescription className="sr-only">
          Перейдите к разделу или найдите задачу по названию.
        </DialogDescription>
        <Command shouldFilter={false}>
          <div className="command-input-wrap">
            <Search size={18} />
            <Command.Input
              placeholder="Найти задачу или перейти к разделу…"
              value={query}
              onValueChange={setQuery}
            />
            <kbd className="kbd">Esc</kbd>
          </div>
          <Command.List className="command-list">
            <Command.Empty>
              {result.isFetching ? "Поиск…" : "Ничего не найдено"}
            </Command.Empty>
            {result.isError ? (
              <div className="notice error-notice" role="alert">
                Поиск временно недоступен.{" "}
                <button onClick={() => result.refetch()}>Повторить</button>
              </div>
            ) : null}
            {query ? (
              <Command.Group heading="Задачи">
                {result.data?.tasks?.map((task) => (
                  <Command.Item
                    key={task.key}
                    className="command-item"
                    value={task.key}
                    onSelect={() => navigate(`/app/tasks/${task.key}`)}
                  >
                    <span className="mono muted">{task.key}</span>
                    <span className="truncate">{task.title}</span>
                  </Command.Item>
                ))}
              </Command.Group>
            ) : null}
            <Command.Group heading="Разделы">
              {navigation
                .filter((item) =>
                  item.title.toLowerCase().includes(query.toLowerCase()),
                )
                .map((item) => (
                  <Command.Item
                    key={item.href}
                    className="command-item"
                    value={item.href}
                    onSelect={() => navigate(item.href)}
                  >
                    <item.icon size={16} />
                    {item.title}
                  </Command.Item>
                ))}
            </Command.Group>
          </Command.List>
          <div className="command-footer">
            ↑ ↓ выбор · Enter открыть · Поиск задач от 2 символов
          </div>
        </Command>
      </DialogContent>
    </Dialog>
  );
}
