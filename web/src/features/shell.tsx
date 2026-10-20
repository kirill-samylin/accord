"use client";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState, type ReactNode } from "react";
import { Dialog as Primitive } from "radix-ui";
import {
  Check,
  ChevronDown,
  ChevronRight,
  CircleHelp,
  ExternalLink,
  Menu,
  Moon,
  Search,
  SquarePen,
  Sun,
} from "lucide-react";
import { Avatar } from "@/components/common";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { MutationResult, PageData } from "@/lib/types";
import { label } from "@/lib/format";
import { CommandPalette } from "./command-palette";
import { currentSection, navigation } from "./navigation";
import { SessionControls } from "./mutations/SessionControls";
export function Shell({
  page,
  children,
  onCreate,
  onSuccess,
}: {
  page: PageData;
  children: ReactNode;
  onCreate: () => void;
  onSuccess: (result: MutationResult) => void;
}) {
  const path = usePathname();
  const [searchOpen, setSearchOpen] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [dark, setDark] = useState(false);
  const section = currentSection(path);
  useEffect(() => {
    let selected: string | null = null;
    try {
      selected = localStorage.getItem("accord.appearance.v1");
    } catch {
      /* Appearance is optional when browser storage is blocked. */
    }
    const value =
      selected === "dark" ||
      (!selected && matchMedia("(prefers-color-scheme: dark)").matches);
    setDark(value);
    document.documentElement.dataset.theme = value ? "dark" : "light";
  }, []);
  useEffect(() => {
    function key(event: KeyboardEvent) {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        setSearchOpen((value) => !value);
      }
    }
    window.addEventListener("keydown", key);
    return () => window.removeEventListener("keydown", key);
  }, []);
  useEffect(() => {
    setDrawerOpen(false);
  }, [path]);
  function toggleTheme() {
    const value = !dark;
    setDark(value);
    document.documentElement.dataset.theme = value ? "dark" : "light";
    try {
      localStorage.setItem("accord.appearance.v1", value ? "dark" : "light");
    } catch {
      /* Optional preference. */
    }
  }
  function sidebar() {
    return (
      <aside className="sidebar" aria-label="Основная навигация">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button className="project-switch" title={page.session.project}>
              <span className="project-mark">A</span>
              <strong className="truncate">
                {page.session.project || "Accord"}
              </strong>
              <ChevronDown size={14} />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start">
            <DropdownMenuItem asChild>
              <Link href="/app/projects">
                <span className="project-mark">A</span>
                <span className="truncate">{page.session.project}</span>
                <Check size={14} />
              </Link>
            </DropdownMenuItem>
            <DropdownMenuItem asChild>
              <Link href="/app/settings">Настройки проекта</Link>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
        <nav className="sidebar-nav">
          <div className="nav-group">
            <button
              className="nav-row"
              disabled={page.can_continue}
              onClick={() => {
                setDrawerOpen(false);
                onCreate();
              }}
            >
              <SquarePen />
              <span className="nav-name">Новая задача</span>
            </button>
            <button
              className="nav-row"
              onClick={() => {
                setDrawerOpen(false);
                setSearchOpen(true);
              }}
            >
              <Search />
              <span className="nav-name">Поиск</span>
              <kbd className="kbd">⌘ K</kbd>
            </button>
          </div>
          {["Работа", "Организация", "Ресурсы"].map((group) => (
            <div className="nav-group" key={group}>
              <div className="nav-group-label">{group}</div>
              {navigation
                .filter((item) => item.group === group)
                .map((item) => (
                  <Link
                    key={item.href}
                    className="nav-row"
                    aria-current={
                      section.href === item.href ? "page" : undefined
                    }
                    href={item.href}
                  >
                    <item.icon />
                    <span className="nav-name">{item.title}</span>
                    {item.href === "/app/questions" &&
                    page.session.queue?.ready ? (
                      <span className="nav-count">
                        {page.session.queue.ready}
                      </span>
                    ) : null}
                  </Link>
                ))}
            </div>
          ))}
        </nav>
        <div className="sidebar-bottom">
          <div className="identity">
            <Avatar name={page.principal.owner} />
            <div className="identity-copy">
              <strong className="truncate">{page.principal.owner}</strong>
              <small className="muted truncate">
                {label(page.session.role)} · {page.principal.machine}
              </small>
            </div>
            <button
              className="icon-button"
              aria-label={dark ? "Светлая тема" : "Тёмная тема"}
              onClick={toggleTheme}
            >
              {dark ? <Sun size={15} /> : <Moon size={15} />}
            </button>
          </div>
        </div>
      </aside>
    );
  }
  return (
    <div className="control-shell">
      <div className="desktop-sidebar">{sidebar()}</div>
      <Primitive.Root open={drawerOpen} onOpenChange={setDrawerOpen}>
        <Primitive.Portal>
          <Primitive.Overlay className="dialog-overlay" />
          <Primitive.Content
            className="drawer-content"
            aria-describedby={undefined}
          >
            <Primitive.Title className="sr-only">
              Навигация Accord
            </Primitive.Title>
            {sidebar()}
          </Primitive.Content>
        </Primitive.Portal>
      </Primitive.Root>
      <div className="main-shell">
        <header className="breadcrumb-bar">
          <button
            className="icon-button mobile-only"
            aria-label="Открыть навигацию"
            onClick={() => setDrawerOpen(true)}
          >
            <Menu size={19} />
          </button>
          <div className="breadcrumb">
            <Link href="/app">{page.session.project || "Accord"}</Link>
            <ChevronRight size={13} />
            <Link href={section.href}>{section.title}</Link>
            {page.task || page.agent ? (
              <>
                <ChevronRight size={13} />
                <strong className="truncate">
                  {page.task?.key ?? page.agent?.owner}
                </strong>
              </>
            ) : null}
          </div>
          <div className="breadcrumb-actions">
            <button
              className="icon-button mobile-only"
              aria-label="Поиск"
              onClick={() => setSearchOpen(true)}
            >
              <Search size={18} />
            </button>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <button className="icon-button" aria-label="Справка и сессия">
                  <CircleHelp size={17} />
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <div className="menu-session">
                  <p className="muted">{label(page.session.role)}</p>
                  <SessionControls page={page} onSuccess={onSuccess} />
                </div>
                <DropdownMenuItem asChild>
                  <a href="/web/">
                    Базовый интерфейс <ExternalLink size={13} />
                  </a>
                </DropdownMenuItem>
                <DropdownMenuItem asChild>
                  <a
                    href="/PAPERCLIP-LICENSE.txt"
                    target="_blank"
                    rel="noreferrer"
                  >
                    Компоненты Paperclip · MIT
                  </a>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </header>
        <main id="main-content" className="main-scroll">
          {children}
          <noscript>
            <p className="notice">
              Для панели нужен JavaScript.{" "}
              <a href="/web/">Откройте базовый интерфейс</a>.
            </p>
          </noscript>
        </main>
      </div>
      <nav className="mobile-nav" aria-label="Быстрая навигация">
        {navigation
          .filter((item) =>
            ["/app", "/app/tasks", "/app/questions", "/app/agents"].includes(
              item.href,
            ),
          )
          .map((item) => (
            <Link
              key={item.href}
              href={item.href}
              aria-current={section.href === item.href ? "page" : undefined}
            >
              <item.icon size={19} />
              {item.title}
            </Link>
          ))}
      </nav>
      <CommandPalette open={searchOpen} onOpenChange={setSearchOpen} />
    </div>
  );
}
