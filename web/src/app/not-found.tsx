import Link from "next/link";
// Shown for unknown /app addresses (including /app/p/{id}/unknown), in the
// interface language instead of the framework's English default.
export default function NotFound() {
  return (
    <main className="auth-page">
      <section className="auth-card">
        <div className="auth-brand">
          <span className="project-mark" aria-hidden>
            A
          </span>
          Accord
        </div>
        <h1>Страница не найдена</h1>
        <p>Такого раздела нет. Проверьте адрес или вернитесь к проекту.</p>
        <Link className="button" href="/app/projects">
          К проектам
        </Link>
      </section>
    </main>
  );
}
