"use client";
import { Select } from "@/components/ui/select";
import { FieldError } from "../mutations/ActionNotice";
import {
  profileRoles,
  providerLabels,
  type ProfileDraft,
} from "@/lib/agent-profiles";
import { roleLabels } from "@/lib/format";
export function ProfileFields({
  draft,
  change,
  errors,
}: {
  draft: ProfileDraft;
  change: (name: keyof ProfileDraft, value: string) => void;
  errors: Record<string, string>;
}) {
  function attributes(name: string) {
    return {
      "aria-invalid": Boolean(errors[name]),
      "aria-describedby": errors[name] ? `${name}-error` : undefined,
    };
  }
  return (
    <>
      <label className="form-field">
        Название
        <input
          name="name"
          required
          maxLength={240}
          value={draft.name}
          onChange={(event) => change("name", event.target.value)}
          {...attributes("name")}
        />
        <FieldError name="name" errors={errors} />
      </label>
      <div className="form-grid">
        <label className="form-field">
          Роль
          <Select
            name="role"
            value={draft.role}
            onChange={(event) => change("role", event.target.value)}
            {...attributes("role")}
          >
            {profileRoles.map((role) => (
              <option key={role} value={role}>
                {roleLabels[role]}
              </option>
            ))}
          </Select>
          <FieldError name="role" errors={errors} />
        </label>
        <label className="form-field">
          Предпочтительный провайдер
          <Select
            name="provider"
            value={draft.provider}
            onChange={(event) => change("provider", event.target.value)}
            {...attributes("provider")}
          >
            {Object.entries(providerLabels).map(([value, title]) => (
              <option key={value} value={value}>
                {title}
              </option>
            ))}
          </Select>
          <FieldError name="provider" errors={errors} />
        </label>
      </div>
      <label className="form-field">
        Предпочтительная модель
        <input
          name="model"
          maxLength={400}
          value={draft.model}
          placeholder="Название модели, необязательно"
          onChange={(event) => change("model", event.target.value)}
          {...attributes("model")}
        />
        <span className="form-help">
          Сохранённое предпочтение. Подключение к провайдеру и запуск здесь не
          выполняются.
        </span>
        <FieldError name="model" errors={errors} />
      </label>
      <label className="form-field">
        Описание
        <textarea
          name="description"
          rows={3}
          value={draft.description}
          onChange={(event) => change("description", event.target.value)}
          {...attributes("description")}
        />
        <span className="form-help">До 4 КиБ текста.</span>
        <FieldError name="description" errors={errors} />
      </label>
      <label className="form-field">
        Инструкции
        <textarea
          name="instructions"
          rows={8}
          value={draft.instructions}
          onChange={(event) => change("instructions", event.target.value)}
          {...attributes("instructions")}
        />
        <span className="form-help">
          До 16 КиБ. Эти данные видят участники проекта. Не добавляйте токены,
          пароли и другие секреты.
        </span>
        <FieldError name="instructions" errors={errors} />
      </label>
    </>
  );
}
