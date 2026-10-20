import Link from "next/link";
export default function Home() {
  return (
    <main className="welcome">
      <div className="project-mark">A</div>
      <h1>Accord</h1>
      <p className="muted">Работа проекта. Задачи. Агенты.</p>
      <Link className="button" href="/app">
        Открыть панель
      </Link>
      <a href="/web/">Базовый интерфейс</a>
    </main>
  );
}
