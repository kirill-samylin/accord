"use client";
import { useState } from "react";
import { useSearchParams } from "next/navigation";
import { Archive, ArchiveRestore, Pencil } from "lucide-react";
import { ProjectLink as Link } from "@/components/project-link";
import { Avatar, EmptyState, Fact, PageHeading } from "@/components/common";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import type { MutationResult, PageData } from "@/lib/types";
import { date, label } from "@/lib/format";
import { providerLabels } from "@/lib/agent-profiles";
import { actionBody, actionID } from "../mutations/contracts";
import { ActionNotice } from "../mutations/ActionNotice";
import { useAction } from "../mutations/useAction";
import { useProjectScope } from "../projects/project-context";
import { AgentNavigation } from "./agent-navigation";
import { ProfileDialog } from "./profile-dialog";
export function ProfileDetail({
  page,
  onSuccess,
}: {
  page: PageData;
  onSuccess: (result: MutationResult) => void;
}) {
  const [edit, setEdit] = useState(false);
  const action = useAction(page, onSuccess);
  const scope = useProjectScope();
  const historical = Boolean(useSearchParams().get("version"));
  const profile = action.page.profile;
  if (!profile)
    return (
      <div className="page">
        <EmptyState title="Профиль не найден" />
      </div>
    );
  const canManage =
    !historical &&
    action.page.can_manage_profiles &&
    action.page.project?.can_write !== false;
  const archived = profile.state === "archived";
  function archive() {
    if (!profile || !canManage) return;
    void action.send(
      `/web/profiles/${profile.id}/archive`,
      actionBody(action.page, "profile:archive", {
        version: String(profile.version),
        state: archived ? "active" : "archived",
      }),
    );
  }
  return (
    <div className="page profile-page">
      <AgentNavigation current="profiles" />
      <Link prefetch={false} className="inline-link" href="/app/profiles">
        ← Все профили
      </Link>
      <div className="agent-identity">
        <Avatar name={profile.name} role={profile.role} agent />
        <PageHeading
          title={profile.name}
          description={`${label(profile.role)} · Версия ${profile.version}${archived ? " · В архиве" : ""}`}
          actions={
            canManage ? (
              <>
                <button
                  className="button button-secondary button-sm"
                  disabled={scope.locked || edit}
                  onClick={() => setEdit(true)}
                >
                  <Pencil size={14} />
                  Изменить
                </button>
                <button
                  className="button button-secondary button-sm"
                  disabled={
                    scope.locked ||
                    edit ||
                    action.locked ||
                    !actionID(action.page, "profile:archive")
                  }
                  onClick={archive}
                >
                  {archived ? (
                    <ArchiveRestore size={14} />
                  ) : (
                    <Archive size={14} />
                  )}
                  {action.busy
                    ? "Сохраняем…"
                    : archived
                      ? "Восстановить"
                      : "В архив"}
                </button>
              </>
            ) : undefined
          }
        />
      </div>
      <ActionNotice
        message={action.message}
        uncertain={action.phase === "uncertain"}
        busy={action.busy}
        onRetry={() => void action.retry()}
      />
      {action.message && !action.locked ? (
        <button
          type="button"
          className="button button-ghost"
          onClick={() => void action.refresh(`/app/profiles/${profile.id}`)}
        >
          Проверить актуальную версию
        </button>
      ) : null}
      {historical ? (
        <div className="notice">
          Открыта сохранённая ревизия {profile.version}.{" "}
          <Link
            prefetch={false}
            className="inline-link"
            href={`/app/profiles/${profile.id}`}
          >
            Перейти к текущей версии
          </Link>
        </div>
      ) : null}
      {archived ? (
        <div className="notice">
          Архивный профиль нельзя выбрать для новой сессии. Прежние сессии
          сохраняют свои настройки и аренды.
        </div>
      ) : null}
      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">Обзор</TabsTrigger>
          <TabsTrigger value="instructions">Инструкции</TabsTrigger>
          <TabsTrigger value="history">История версий</TabsTrigger>
          <TabsTrigger value="use">Использование</TabsTrigger>
        </TabsList>
        <TabsContent value="overview">
          <div className="detail-layout">
            <div className="screen-stack">
              <section className="settings-section">
                <h2>Назначение</h2>
                <p className="profile-text">
                  {profile.description || "Описание не добавлено"}
                </p>
              </section>
              <section className="settings-section">
                <h2>Предпочтения модели</h2>
                <dl>
                  <Fact
                    name="Провайдер"
                    value={providerLabels[profile.provider] ?? profile.provider}
                  />
                  <Fact name="Модель" value={profile.model || "Не указана"} />
                </dl>
                <p className="form-help">
                  Это сохранённые настройки. Сервис не подключается к провайдеру
                  и не запускает агента.
                </p>
              </section>
            </div>
            <aside className="properties-panel">
              <h3>Профиль проекта</h3>
              <dl>
                <Fact
                  name="ID"
                  value={<span className="mono break">{profile.id}</span>}
                />
                <Fact name="Роль" value={label(profile.role)} />
                <Fact name="Версия" value={profile.version} />
                <Fact
                  name="Состояние"
                  value={archived ? "В архиве" : "Действующий"}
                />
                <Fact name="Создан" value={date(profile.created_at)} />
                <Fact name="Изменён" value={date(profile.updated_at)} />
              </dl>
            </aside>
          </div>
        </TabsContent>
        <TabsContent value="instructions">
          <section className="settings-section">
            <h2>Инструкции версии {profile.version}</h2>
            <p className="profile-text profile-instructions">
              {profile.instructions || "Инструкции не добавлены"}
            </p>
            <p className="form-help">
              Участники проекта видят этот текст. Новая сессия получает его
              только при явном выборе профиля и версии.
            </p>
          </section>
        </TabsContent>
        <TabsContent value="history">
          <div className="list-surface profile-history">
            {(action.page.profile_revisions ?? []).map((revision) => (
              <div className="profile-history-row" key={revision.version}>
                <Link
                  prefetch={false}
                  className="inline-link"
                  href={`/app/profiles/${profile.id}?version=${revision.version}`}
                >
                  Версия {revision.version}
                </Link>
                <span>
                  {revision.state === "archived" ? "В архиве" : "Действующий"}
                </span>
                <span className="muted">{date(revision.updated_at)}</span>
                <span
                  className="mono break muted"
                  title="Сессия автора изменения"
                >
                  {revision.updated_by}
                </span>
              </div>
            ))}
          </div>
          <p className="caption-note muted">
            Ревизии сохраняются неизменно. Сессия использует выбранную при
            открытии версию.
          </p>
        </TabsContent>
        <TabsContent value="use">
          <section className="settings-section">
            <h2>Явный выбор профиля</h2>
            <p>
              Клиент открывает сессию в этом проекте через describe с ролью, ID
              профиля и его актуальной версией. Настройки возвращаются в ответе
              и сохраняются в сессии.
            </p>
            <dl>
              <Fact name="profile_id" value={<code>{profile.id}</code>} />
              <Fact name="profile_version" value={profile.version} />
              <Fact name="role" value={<code>{profile.role}</code>} />
            </dl>
            <p className="form-help">
              Редактирование профиля не меняет уже открытые сессии. Для новой
              версии клиент явно открывает новую сессию. Настройка профиля не
              является запуском процесса или подключением Claude Code/Codex.
            </p>
            <Link prefetch={false} className="inline-link" href="/app/agents">
              Открыть сессии проекта
            </Link>
          </section>
        </TabsContent>
      </Tabs>
      {edit ? (
        <ProfileDialog
          key={profile.id}
          page={action.page}
          onSuccess={onSuccess}
          open={edit}
          onOpenChange={setEdit}
        />
      ) : null}
    </div>
  );
}
