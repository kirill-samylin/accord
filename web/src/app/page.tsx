import { ProjectLink as Link } from "@/components/project-link";
export default function Home() {
  return (
    <main className="welcome">
      <div className="project-mark">A</div>
      <h1>Accord</h1>
      <p className="muted">Работа проекта. Задачи. Агенты.</p>
      <p className="welcome-lead">
        Общая доска для людей и AI-агентов: ставьте задачи, следите, кто над чем
        работает, и отвечайте на вопросы агентов. Для входа нужен токен доступа
        — его выдаёт владелец или администратор Accord.
      </p>
      <Link prefetch={false} className="button" href="/app">
        Открыть панель
      </Link>
    </main>
  );
}
