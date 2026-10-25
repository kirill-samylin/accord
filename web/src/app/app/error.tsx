"use client";
export default function ErrorPage({ reset }: { reset: () => void }) {
  return (
    <main className="welcome">
      <h1>Не удалось открыть панель</h1>
      <button className="button" onClick={reset}>
        Повторить
      </button>
    </main>
  );
}
