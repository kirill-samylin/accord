import { Select } from "@/components/ui/select";
import type {
  CoordinationDraft,
  CoordinationOperation,
} from "@/lib/coordination";
import { latestPositiveReview, transitionChoices } from "@/lib/coordination";
import type { PageData, Task } from "@/lib/types";
import { ProfileChoice } from "./profile-choice";
export function CoordinationFields({
  operation,
  draft,
  task,
  page,
  change,
}: {
  operation: CoordinationOperation;
  draft: CoordinationDraft;
  task: Task;
  page: PageData;
  change: (name: keyof CoordinationDraft, value: string) => void;
}) {
  if (operation === "assign")
    return (
      <>
        <ProfileChoice
          value={draft.profile}
          role={task.role}
          profiles={page.assignment_profiles}
          onChange={(value) => change("profile", value)}
        />
        <Reason
          value={draft.reason}
          change={(value) => change("reason", value)}
        />
      </>
    );
  if (operation === "checks") {
    const reviewer = latestPositiveReview(page.coordination);
    return (
      <>
        <label className="form-field">
          Проверка
          <Select
            value={draft.role}
            onChange={(event) => {
              change("role", event.target.value);
              change("profile", "");
            }}
          >
            <option value="reviewer">Ревью</option>
            <option value="tester" disabled={!reviewer}>
              QA после одобренного ревью
            </option>
          </Select>
        </label>
        <dl className="coord-target">
          <dt>Проверяемый результат</dt>
          <dd className="mono break">
            {task.key} · поколение {task.generation}
            <br />
            {task.evidence.commit}
          </dd>
        </dl>
        {draft.role === "tester" && reviewer ? (
          <p className="form-help">
            QA получит обязательную зависимость от ревью {reviewer.task_key}.
            Если результат или ревью изменятся, сервер заново проверит допуск.
          </p>
        ) : (
          <p className="form-help">
            Проверку выполнит отдельная сессия выбранной роли.
          </p>
        )}
        <ProfileChoice
          value={draft.profile}
          role={draft.role}
          profiles={page.assignment_profiles}
          onChange={(value) => change("profile", value)}
        />
      </>
    );
  }
  if (operation === "link")
    return (
      <>
        <label className="form-field">
          Действие
          <Select
            value={draft.unlink}
            onChange={(event) => change("unlink", event.target.value)}
          >
            <option value="false">Добавить связь</option>
            <option value="true">Удалить связь</option>
          </Select>
        </label>
        <label className="form-field">
          Исходная задача
          <input
            name="from"
            required
            pattern="T-[1-9][0-9]*"
            placeholder="T-…"
            value={draft.from}
            onChange={(event) => change("from", event.target.value)}
            list="coordination-linked-tasks"
          />
          <datalist id="coordination-linked-tasks">
            {page.coordination?.dependencies?.map(({ link }) => (
              <option key={`${link.from}:${link.type}`} value={link.from}>
                {link.type}
              </option>
            ))}
          </datalist>
          <span className="form-help">
            Связь направлена от указанной задачи к {task.key}.
          </span>
        </label>
        <label className="form-field">
          Тип связи
          <Select
            value={draft.type}
            onChange={(event) => change("type", event.target.value)}
          >
            <option value="blocks">Блокирует</option>
            <option value="contract_of">Предоставляет контракт</option>
            <option value="spawned_by">Порождена</option>
            <option value="related">Связана</option>
          </Select>
        </label>
        {draft.unlink !== "true" ? (
          <label className="form-field">
            Условие готовности
            <Select
              value={draft.condition}
              onChange={(event) => change("condition", event.target.value)}
            >
              <option value="">Успешный результат роли</option>
              <option value="completed">Завершена успешно</option>
              <option value="approved">Ревью одобрено</option>
              <option value="passed">QA пройдено</option>
            </Select>
          </label>
        ) : (
          <p className="form-help">
            Удаление связи не отменяет ранее назначенные обязательства review и
            QA.
          </p>
        )}
      </>
    );
  const reopeningTask = task.type === "task" && task.state === "done";
  return (
    <>
      <label className="form-field">
        Новое состояние
        <Select
          value={draft.state}
          onChange={(event) => change("state", event.target.value)}
        >
          {transitionChoices(task).map((choice) => (
            <option key={choice.value} value={choice.value}>
              {choice.title}
            </option>
          ))}
        </Select>
      </label>
      {draft.state !== "done" ? (
        <Reason
          value={draft.reason}
          change={(value) => change("reason", value)}
        />
      ) : null}
      {reopeningTask ? (
        <label className="form-field">
          Задача с доказательством дефекта
          <input
            name="evidence_ref"
            required
            pattern="T-[1-9][0-9]*"
            placeholder="T-…"
            value={draft.evidence_ref}
            onChange={(event) => change("evidence_ref", event.target.value)}
          />
          <span className="form-help">
            Нужна существующая завершённая задача с доказательством дефекта.
          </span>
        </label>
      ) : null}
      {task.type === "epic" && draft.state === "done" ? (
        <p className="form-help">
          Закрытие возможно только после всех обязательных работ, актуальных
          проверок и завершения аренд.
        </p>
      ) : null}
    </>
  );
}
function Reason({
  value,
  change,
}: {
  value: string;
  change: (value: string) => void;
}) {
  return (
    <label className="form-field">
      Причина
      <textarea
        name="reason"
        rows={3}
        maxLength={4096}
        required
        value={value}
        onChange={(event) => change(event.target.value)}
      />
    </label>
  );
}
