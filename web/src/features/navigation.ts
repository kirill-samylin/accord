import {
  Activity,
  BarChart3,
  Bot,
  Clock3,
  Files,
  FolderKanban,
  House,
  Inbox,
  ListTodo,
  Settings2,
} from "lucide-react";
import { screenPath } from "@/lib/project-routes";
export const navigation = [
  { href: "/app", title: "Обзор", icon: House, group: "Работа" },
  { href: "/app/tasks", title: "Задачи", icon: ListTodo, group: "Работа" },
  { href: "/app/questions", title: "Вопросы", icon: Inbox, group: "Работа" },
  {
    href: "/app/projects",
    title: "Проекты",
    icon: FolderKanban,
    group: "Работа",
  },
  { href: "/app/agents", title: "Агенты", icon: Bot, group: "Организация" },
  {
    href: "/app/activity",
    title: "Активность",
    icon: Activity,
    group: "Организация",
  },
  {
    href: "/app/stats",
    title: "Статистика",
    icon: BarChart3,
    group: "Организация",
  },
  { href: "/app/designs", title: "Дизайн", icon: Files, group: "Ресурсы" },
  { href: "/app/history", title: "Решения", icon: Clock3, group: "Ресурсы" },
  {
    href: "/app/settings",
    title: "Настройки",
    icon: Settings2,
    group: "Ресурсы",
  },
];
export function currentSection(path: string) {
  path = screenPath(path);
  return (
    navigation.find(
      (item) => item.href !== "/app" && path.startsWith(item.href),
    ) ?? navigation[0]
  );
}
