"use client";
import { Select } from "@/components/ui/select";
import type { CatalogPage } from "@/lib/projects";
import type { CatalogMutationResult } from "@/lib/types";
import { RegistryForm } from "./registry-form";
import { Disclosure } from "@/components/disclosure";
const roleNames: Record<string, string> = {
  owner: "Владелец",
  editor: "Редактор",
  viewer: "Наблюдатель",
};
export const projectRole = (role: string) => roleNames[role] ?? role;
function Roles({ value = "viewer" }: { value?: string }) {
  return (
    <Select name="role" defaultValue={value}>
      <option value="viewer">Наблюдатель · только чтение</option>
      <option value="editor">Редактор · работа с задачами</option>
      <option value="owner">Владелец · управление доступом</option>
    </Select>
  );
}
export function ProjectMembers({
  page,
  onSuccess,
}: {
  page: CatalogPage;
  onSuccess: (result: CatalogMutationResult) => void;
}) {
  const project = page.project;
  if (!project?.can_manage) return null;
  const endpoint = `/web/catalog/projects/${project.id}/members`;
  const owners =
    page.members?.filter((member) => member.role === "owner").length ?? 0;
  return (
    <section className="settings-section">
      <h2>Участники и доступ</h2>
      <p className="form-help">
        Добавьте участника по его ID учётной записи. Одинаковые имена не
        объединяют доступы. Изменения прав действуют на стороне сервера.
      </p>
      {page.members?.map((member) => (
        <div className="member-card" key={member.identity_id}>
          <div>
            <strong>{member.owner}</strong>
            <p className="muted">{member.machine}</p>
            <code className="mono break">{member.identity_id}</code>
          </div>
          <RegistryForm
            page={page}
            actionKey={`member:${member.identity_id}:set`}
            endpoint={endpoint}
            title="Изменить доступ"
            fields={{
              action: "set",
              identity_id: member.identity_id,
              version: String(member.version),
            }}
            onSuccess={onSuccess}
            disabled={member.role === "owner" && owners === 1}
          >
            <label className="form-field">
              Роль
              <Roles value={member.role} />
            </label>
          </RegistryForm>
          <RegistryForm
            page={page}
            actionKey={`member:${member.identity_id}:revoke`}
            endpoint={endpoint}
            title="Отозвать доступ"
            fields={{
              action: "revoke",
              identity_id: member.identity_id,
              version: String(member.version),
            }}
            confirm="Отозвать доступ участника к этому проекту? История его работы сохранится."
            onSuccess={onSuccess}
            disabled={member.role === "owner" && owners === 1}
          />
          {member.role === "owner" && owners === 1 ? (
            <p className="form-help">
              Последнего владельца нельзя удалить или понизить.
            </p>
          ) : null}
        </div>
      ))}
      <Disclosure className="registry-add" summary="Добавить участника">
        <RegistryForm
          page={page}
          actionKey="member:grant"
          endpoint={endpoint}
          title="Предоставить доступ"
          fields={{ action: "grant" }}
          onSuccess={onSuccess}
        >
          <label className="form-field">
            ID учётной записи
            <input
              name="identity_id"
              required
              maxLength={160}
              autoComplete="off"
              spellCheck={false}
            />
          </label>
          <label className="form-field">
            Роль
            <Roles />
          </label>
        </RegistryForm>
      </Disclosure>
    </section>
  );
}
