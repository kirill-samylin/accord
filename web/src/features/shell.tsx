"use client";
import { ProjectLink as Link } from "@/components/project-link";
import { usePathname } from "next/navigation";
import { useEffect, useRef, useState, type ReactNode } from "react";
import { Dialog as Primitive } from "radix-ui";
import {
  ChevronRight,
  CircleHelp,
  ExternalLink,
  Menu,
  Moon,
  Search,
  SquarePen,
  Sun,
} from "lucide-react";
import { restoreDialogFocus } from "@/components/ui/dialog-focus";
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
import { ProjectSwitcher } from "./projects/project-switcher";
import { useProjectScope } from "./projects/project-context";
import { useAppearance } from "./hooks/use-appearance";
import { HelpDialog } from "./help-dialog";
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
  const [helpOpen, setHelpOpen] = useState(false);
  const menuTrigger = useRef<HTMLButtonElement | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const { dark, toggle: toggleTheme } = useAppearance();
  const scope = useProjectScope();
  const section = currentSection(path);
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
  function sidebar() {
    return (
      <aside className="sidebar" aria-label="Основная навигация">
        <ProjectSwitcher page={page} />
        <nav className="sidebar-nav">
          <div className="nav-group">
            <button
              className="nav-row"
              disabled={
                page.can_continue ||
                page.project?.can_write === false ||
                scope.locked
              }
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
              data-command-trigger
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
                    prefetch={false}
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
            onCloseAutoFocus={(event) => {
              event.preventDefault();
              restoreDialogFocus(menuTrigger.current);
            }}
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
            ref={menuTrigger}
            onClick={() => setDrawerOpen(true)}
          >
            <Menu size={19} />
          </button>
          <div className="breadcrumb">
            <Link prefetch={false} href="/app">
              {page.project?.name || page.session.project || "Accord"}
            </Link>
            <ChevronRight size={13} />
            <Link prefetch={false} href={section.href}>
              {section.title}
            </Link>
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
              data-command-trigger
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
                <DropdownMenuItem onSelect={() => setHelpOpen(true)}>
                  Как начать работу
                </DropdownMenuItem>
                <div className="menu-session">
                  <p className="muted">{label(page.session.role)}</p>
                  <SessionControls page={page} onSuccess={onSuccess} />
                </div>
                {!scope.id ? (
                  <DropdownMenuItem asChild>
                    <a href={scope.web("/web/")}>
                      Упрощённая версия <ExternalLink size={13} />
                    </a>
                  </DropdownMenuItem>
                ) : null}
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
        <HelpDialog
          open={helpOpen}
          onOpenChange={setHelpOpen}
          onCreate={() => {
            setHelpOpen(false);
            onCreate();
          }}
        />
        <main id="main-content" className="main-scroll">
          {scope.locked ? (
            <div className="page-notice notice" role="status">
              Сначала проверьте результат отправки. Переходы между проектами
              пока недоступны.
            </div>
          ) : null}
          {children}
          <noscript>
            <p className="notice">
              Для панели нужен JavaScript.{" "}
              {!scope.id ? (
                <a href="/web/">Откройте базовый интерфейс</a>
              ) : null}
            </p>
          </noscript>
        </main>
      </div>
      <nav className="mobile-nav" aria-label="Быстрая навигация">
        {navigation
          .filter((item) =>
            ["/app", "/app/team", "/app/tasks", "/app/questions"].includes(
              item.href,
            ),
          )
          .map((item) => (
            <Link
              prefetch={false}
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
