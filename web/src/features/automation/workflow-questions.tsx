"use client";
import { useState, type FormEvent } from "react";
import type { WorkflowQuestion } from "@/lib/automation";
import { ProjectLink as Link } from "@/components/project-link";
import type { MutationProps } from "../mutations/contracts";
import { actionBody, actionID } from "../mutations/contracts";
import { useAction } from "../mutations/useAction";
import { ActionNotice } from "../mutations/ActionNotice";
function QuestionAnswer({
  page,
  onSuccess,
  question,
}: MutationProps & { question: WorkflowQuestion }) {
  const [answer, setAnswer] = useState("");
  const [resume, setResume] = useState(false);
  const [error, setError] = useState("");
  const action = useAction(page, onSuccess);
  const detail = action.page.workflow,
    actor = action.page.automation?.actor_principal_id;
  const canAnswer =
    question.state === "open" &&
    actor &&
    question.designated_principals.includes(actor) &&
    action.page.project?.can_write !== false;
  function submit(event: FormEvent) {
    event.preventDefault();
    if (!answer.trim() || new TextEncoder().encode(answer).length > 16384) {
      setError("Ответ должен содержать от 1 символа до 16 КиБ.");
      return;
    }
    if (!detail) return;
    void action.send(
      "/web/automation/answer",
      actionBody(action.page, "automation:answer", {
        workflow_id: detail.workflow.id,
        version: String(detail.workflow.version),
        question_id: question.id,
        question_version: String(question.version),
        answer,
        resume: String(resume),
      }),
    );
  }
  return (
    <article className="automation-card screen-stack">
      <p className="automation-text">{question.question}</p>
      <Link
        href={`/app/tasks/${question.human_task_key}`}
        prefetch={false}
        className="muted"
      >
        Вопрос {question.human_task_key} ·{" "}
        {question.state === "open" ? "ожидает ответа" : "ответ сохранён"}
      </Link>
      {canAnswer ? (
        <form className="mutation-form" onSubmit={submit}>
          <fieldset disabled={action.locked}>
            {question.options?.length ? (
              <div className="form-actions">
                {question.options.map((option) => (
                  <button
                    key={option}
                    type="button"
                    className="button button-secondary button-sm"
                    onClick={() => setAnswer(option)}
                  >
                    {option}
                  </button>
                ))}
              </div>
            ) : null}
            <label className="form-field">
              Ваш ответ
              <textarea
                rows={3}
                value={answer}
                onChange={(e) => setAnswer(e.target.value)}
              />
            </label>
            <label className="automation-check">
              <input
                type="checkbox"
                checked={resume}
                onChange={(e) => setResume(e.target.checked)}
              />
              <span>Продолжить работу после сохранения ответа</span>
            </label>
            {error ? <p className="field-error">{error}</p> : null}
            <button
              type="submit"
              className="button"
              disabled={!actionID(action.page, "automation:answer")}
            >
              Сохранить ответ
            </button>
          </fieldset>
          <ActionNotice
            message={action.message}
            uncertain={action.phase === "uncertain"}
            busy={action.busy}
            onRetry={() => void action.retry()}
          />
        </form>
      ) : question.state === "open" ? (
        <p className="form-help">
          Ответ ожидается от назначенного участника. Агент не может ответить за
          человека.
        </p>
      ) : null}
    </article>
  );
}
export function WorkflowQuestions(props: MutationProps) {
  const questions = props.page.workflow?.questions ?? [];
  if (!questions.length) return null;
  return (
    <section className="screen-stack">
      <h2>Вопросы команде</h2>
      {questions.map((question) => (
        <QuestionAnswer {...props} key={question.id} question={question} />
      ))}
    </section>
  );
}
