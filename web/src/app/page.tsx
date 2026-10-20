import { ProjectLink as Link } from "@/components/project-link";
export default function Home() {
  return (
    <main className="welcome">
      <div className="project-mark">A</div>
      <h1>Accord</h1>
      <p className="muted">Работа проекта. Задачи. Агенты.</p>
      <Link prefetch={false} className="button" href="/app">
        Открыть панель
      </Link>
      <a href="/web/">Базовый интерфейс</a>
    </main>
  );
}
