"use client";
import { ProjectLink as Link } from "@/components/project-link";
import { useState } from "react";
import { FileText, Pencil } from "lucide-react";
import { Disclosure } from "@/components/disclosure";
import {
  EmptyState,
  EventRows,
  Fact,
  PageHeading,
  Status,
} from "@/components/common";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import type { MutationResult, PageData } from "@/lib/types";
import { date, label } from "@/lib/format";
import { QuestionActions } from "../mutations/QuestionActions";
import { TaskProperties } from "./task-properties";
import { EpicTasks } from "./epic-tasks";
import { EditTaskDialog } from "../mutations/EditTaskDialog";
import { CoordinationPanel } from "../coordination/coordination-panel";
import { DuplicateBanners, SimilarTasks } from "../duplicates/task-duplicates";
export function TaskDetail({
  page,
  onSuccess,
  compact = false,
}: {
  page: PageData;
  onSuccess: (result: MutationResult) => void;
  compact?: boolean;
}) {
  const [editOpen, setEditOpen] = useState(false);
  const task = page.task;
  if (!task)
    return (
      <div className="page">
        <EmptyState title="Задача не найдена" />
      </div>
    );
  const properties = <TaskProperties task={task} />;
  return (
    <div className={`page task-detail-page${compact ? " compact-detail" : ""}`}>
      <div className="task-detail-heading">
        <div className="task-detail-kicker">
          <span className="mono muted">{task.key}</span>
          <Status value={task.state} blocked={task.blocked} />
        </div>
        <PageHeading
          title={task.title}
          actions={
            !compact && page.can_update ? (
              <button
                className="button button-secondary button-sm"
                onClick={() => setEditOpen(true)}
              >
                <Pencil size={13} />
                Изменить
              </button>
            ) : undefined
          }
        />
      </div>
      <div className="detail-layout">
        <div className="detail-main">
          <Tabs defaultValue="description">
            <TabsList>
              <TabsTrigger value="description">Описание</TabsTrigger>
              <TabsTrigger value="coordination">Координация</TabsTrigger>
              <TabsTrigger value="activity">Активность</TabsTrigger>
              <TabsTrigger value="attempts">
                Попытки
                {task.attempts?.length ? ` · ${task.attempts.length}` : ""}
              </TabsTrigger>
              <TabsTrigger value="evidence">Результат</TabsTrigger>
            </TabsList>
            <TabsContent value="description" forceMount>
              <div className="screen-stack">
                {compact ? (
                  <Link
                    prefetch={false}
                    className="button button-secondary button-sm"
                    href={`/app/tasks/${task.key}`}
                  >
                    Открыть задачу для работы
                  </Link>
                ) : null}
                <DuplicateBanners task={task} />
                <div className="task-description">
                  {task.description || (
                    <span className="muted">Описание ещё не добавлено.</span>
                  )}
                </div>
                {task.type === "epic" && !compact ? (
                  <EpicTasks epicKey={task.key} />
                ) : null}
                {task.links?.length ? (
                  <section>
                    <h3>Связи</h3>
                    <div className="linked-tasks">
                      {task.links.map((link, i) => (
                        <Link
                          prefetch={false}
                          key={`${link.from}-${link.to}-${i}`}
                          href={`/app/tasks/${link.to === task.key ? link.from : link.to}`}
                        >
                          <span className="mono">
                            {link.to === task.key ? link.from : link.to}
                          </span>
                          <span className="muted">
                            {(
                              {
                                duplicate_of:
                                  link.from === task.key
                                    ? "основная задача"
                                    : "дубль этой задачи",
                                related: "связана",
                                blocks: "блокирует",
                                contract_of: "контракт",
                                spawned_by: "порождена",
                              } as Record<string, string>
                            )[link.type] ?? link.type}
                            {link.condition ? ` · ${link.condition}` : ""}
                          </span>
                        </Link>
                      ))}
                    </div>
                  </section>
                ) : null}
                <SimilarTasks
                  key={`${task.key}:${task.version}`}
                  page={page}
                  onSuccess={onSuccess}
                  readOnly={compact}
                />
                {!compact && task.role === "human" && task.type === "task" ? (
                  <section className="answer-panel">
                    <QuestionActions
                      key={task.key}
                      page={page}
                      task={task}
                      onSuccess={onSuccess}
                    />
                  </section>
                ) : null}
                {task.comments?.length ? (
                  <section className="comments">
                    <h3>Комментарии</h3>
                    {task.comments.map((comment, index) => (
                      <div className="comment" key={index}>
                        <div className="muted mono">
                          {comment.session_id} · {date(comment.at)}
                        </div>
                        <p>{comment.text}</p>
                      </div>
                    ))}
                  </section>
                ) : null}
              </div>
            </TabsContent>
            <TabsContent value="coordination">
              <CoordinationPanel
                page={page}
                onSuccess={onSuccess}
                readOnly={compact}
              />
            </TabsContent>
            <TabsContent value="activity">
              <EventRows events={page.activity ?? []} />
              <Link
                prefetch={false}
                className="inline-link"
                href={`/app/activity?task_key=${task.key}`}
              >
                Вся активность задачи
              </Link>
            </TabsContent>
            <TabsContent value="attempts">
              {task.attempts?.length ? (
                <div className="attempt-list">
                  {task.attempts.map((attempt) => (
                    <section className="attempt" key={attempt.generation}>
                      <div className="section-heading">
                        <h3>Попытка {attempt.generation}</h3>
                        <span className="muted">
                          {attempt.ended_at ? "Завершена" : "В работе"}
                        </span>
                      </div>
                      <dl>
                        <Fact name="Начало" value={date(attempt.started_at)} />
                        <Fact name="Окончание" value={date(attempt.ended_at)} />
                        <Fact
                          name="Сессия"
                          value={
                            <Link
                              prefetch={false}
                              className="mono"
                              href={`/app/agents/${attempt.session_id}`}
                            >
                              {attempt.session_id}
                            </Link>
                          }
                        />
                        <Fact name="Машина" value={attempt.machine} />
                        <Fact name="Причина" value={attempt.reason} />
                        <Fact name="Ветка" value={attempt.branch} />
                        <Fact
                          name="Коммит"
                          value={
                            <span className="mono">
                              {attempt.commit ?? "—"}
                            </span>
                          }
                        />
                        <Fact
                          name="Токены"
                          value={attempt.tokens_spent ?? "Не записаны"}
                        />
                      </dl>
                    </section>
                  ))}
                </div>
              ) : (
                <EmptyState
                  title="Попыток пока нет"
                  description="История появится после первого захвата задачи."
                />
              )}
            </TabsContent>
            <TabsContent value="evidence">
              {Object.values(task.evidence).some((value) =>
                typeof value === "string" ? value.length > 0 : value?.length,
              ) ? (
                <dl className="evidence-list">
                  {Object.entries(task.evidence).map(([key, value]) => (
                    <Fact
                      key={key}
                      name={
                        (
                          {
                            commit: "Коммит",
                            verified_commit: "Проверенный коммит",
                            answer: "Ответ",
                            summary: "Итог",
                            run_summary: "Проверки",
                            artifact: "Артефакты",
                            findings: "Замечания",
                          } as Record<string, string>
                        )[key] ?? key
                      }
                      value={
                        <span className="pre-wrap break">
                          {Array.isArray(value) ? value.join("\n") : value}
                        </span>
                      }
                    />
                  ))}
                  {task.review ? (
                    <>
                      <Fact name="Вердикт" value={task.review.outcome} />
                      <Fact
                        name="Проверена задача"
                        value={
                          <Link
                            prefetch={false}
                            href={`/app/tasks/${task.review.target_key}`}
                          >
                            {task.review.target_key}
                          </Link>
                        }
                      />
                      <Fact
                        name="Проверен коммит"
                        value={
                          <span className="mono">
                            {task.review.target_commit}
                          </span>
                        }
                      />
                    </>
                  ) : null}
                </dl>
              ) : (
                <EmptyState
                  title="Результат ещё не сохранён"
                  icon={<FileText size={24} />}
                />
              )}
            </TabsContent>
          </Tabs>
        </div>
        <aside className="properties-panel">
          <h3>Свойства</h3>
          {properties}
        </aside>
        <Disclosure className="mobile-properties" summary="Свойства задачи">
          {properties}
        </Disclosure>
      </div>
      <EditTaskDialog
        key={`${task.key}:${task.version}`}
        page={page}
        onSuccess={onSuccess}
        open={editOpen}
        onOpenChange={setEditOpen}
      />
    </div>
  );
}
