"use client";
import {
  EmptyState,
  Metric,
  PageHeading,
  SectionHeading,
} from "@/components/common";
import type { PageData } from "@/lib/types";
import { duration, label, number } from "@/lib/format";
import { useFilters } from "../query-controls";
export function StatsScreen({ page }: { page: PageData }) {
  const filters = useFilters();
  const stats = page.stats;
  if (!stats)
    return (
      <div className="page">
        <EmptyState title="Статистика недоступна" />
      </div>
    );
  const totals = stats.totals;
  const maximum = Math.max(
    1,
    ...stats.groups.map((group) => group.metrics.first_completions),
  );
  return (
    <div className="page">
      <PageHeading
        title="Статистика"
        description="Фактические результаты, попытки и учёт токенов"
      />
      <div className="toolbar">
        <div className="toolbar-start">
          <label className="toolbar-select">
            Период
            <select
              aria-label="Период статистики"
              value={page.draft.days ?? "7"}
              onChange={(event) => filters.set({ days: event.target.value })}
            >
              {["1", "7", "30", "90"].map((days) => (
                <option key={days} value={days}>
                  {days === "1" ? "Последние сутки" : `${days} дней`}
                </option>
              ))}
            </select>
          </label>
          <label className="toolbar-select">
            Группировка
            <select
              aria-label="Группировка статистики"
              value={stats.group_by}
              onChange={(event) =>
                filters.set({ group_by: event.target.value })
              }
            >
              <option value="role">Роль</option>
              <option value="owner">Исполнитель</option>
              <option value="epic">Эпик</option>
            </select>
          </label>
        </div>
      </div>
      {stats.incomplete_data ? (
        <div className="notice" role="status">
          Часть исторических данных неполная: {stats.incomplete_events} событий
          без полного учёта.
        </div>
      ) : null}
      <div className="metrics-grid">
        <Metric
          title="Завершено впервые"
          value={number(totals.first_completions)}
          caption={`${totals.epic_completions} эпиков завершено`}
        />
        <Metric
          title="Повторных завершений"
          value={number(totals.rework_completions)}
          caption={`${totals.returns} возвратов`}
        />
        <Metric
          title="Учтено токенов"
          value={number(totals.tokens_spent)}
          caption={`${number(totals.missing_tokens_share * 100)}% попыток без учёта`}
        />
        <Metric
          title="Попытки"
          value={number(totals.attempts)}
          caption={`${totals.expired_leases} истёкших аренд`}
        />
      </div>
      <div className="dashboard-charts">
        <section className="chart-panel">
          <SectionHeading title="Завершения по группам" />
          {stats.groups.length ? (
            <div
              className="group-chart"
              role="img"
              aria-label={stats.groups
                .map(
                  (group) =>
                    `${label(group.key)}: ${group.metrics.first_completions}`,
                )
                .join(", ")}
            >
              {stats.groups.map((group) => (
                <div className="group-bar-row" key={group.key}>
                  <span className="truncate" title={group.key}>
                    {label(group.key)}
                  </span>
                  <div className="bar-track">
                    <span
                      className="bar-fill fill-done"
                      style={{
                        width: `${(group.metrics.first_completions / maximum) * 100}%`,
                      }}
                    />
                  </div>
                  <strong>{group.metrics.first_completions}</strong>
                </div>
              ))}
            </div>
          ) : (
            <EmptyState title="Нет завершений за период" />
          )}
        </section>
        <section className="chart-panel">
          <SectionHeading title="Время и качество" />
          <dl className="queue-summary">
            <div>
              <dt>Календарная длительность</dt>
              <dd>{duration(totals.wall_duration_seconds)}</dd>
            </div>
            <div>
              <dt>Время блокировок</dt>
              <dd>{duration(totals.blocked_duration_seconds)}</dd>
            </div>
            <div>
              <dt>Предупреждения правил</dt>
              <dd>{totals.rule_warnings}</dd>
            </div>
          </dl>
          <p className="form-help">
            Календарное время попыток. Данные о стоимости и вычислительном
            времени не собираются.
          </p>
        </section>
      </div>
      <section>
        <SectionHeading title="По группам" />
        <div className="table-scroll">
          <table className="data-table">
            <thead>
              <tr>
                <th>Группа</th>
                <th>Завершено</th>
                <th>Повторно</th>
                <th>Возвраты</th>
                <th>Токены</th>
                <th>Без учёта</th>
                <th>Длительность</th>
              </tr>
            </thead>
            <tbody>
              {stats.groups.map((group) => (
                <tr key={group.key}>
                  <td>{label(group.key)}</td>
                  <td>{group.metrics.first_completions}</td>
                  <td>{group.metrics.rework_completions}</td>
                  <td>{group.metrics.returns}</td>
                  <td>{number(group.metrics.tokens_spent)}</td>
                  <td>{number(group.metrics.missing_tokens_share * 100)}%</td>
                  <td>{duration(group.metrics.wall_duration_seconds)}</td>
                </tr>
              ))}
            </tbody>
          </table>
          {!stats.groups.length ? (
            <EmptyState title="Нет данных за выбранный период" />
          ) : null}
        </div>
      </section>
    </div>
  );
}
