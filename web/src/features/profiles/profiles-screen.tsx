"use client";
import { useState } from "react";
import { Plus, RefreshCw, Users } from "lucide-react";
import { ProjectLink as Link } from "@/components/project-link";
import { Avatar, EmptyState, PageHeading } from "@/components/common";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import type { MutationResult, PageData } from "@/lib/types";
import { date, label } from "@/lib/format";
import { providerLabels } from "@/lib/agent-profiles";
import { useFilters } from "../query-controls";
import { useProjectScope } from "../projects/project-context";
import { ProfileDialog } from "./profile-dialog";
import { AgentNavigation } from "./agent-navigation";
export function ProfilesScreen({
  page,
  refresh,
  onSuccess,
}: {
  page: PageData;
  refresh: () => void;
  onSuccess: (result: MutationResult) => void;
}) {
  const [create, setCreate] = useState(false);
  const filters = useFilters();
  const scope = useProjectScope();
  const profiles = page.profiles ?? [];
  const canManage =
    page.can_manage_profiles && page.project?.can_write !== false;
  return (
    <div className="page profile-page">
      <PageHeading
        title="Агенты"
        description="Постоянные профили с ролью, настройками и инструкциями"
        actions={
          <>
            <button
              className="button button-secondary button-sm"
              onClick={refresh}
            >
              <RefreshCw size={14} />
              Обновить
            </button>
            {canManage ? (
              <button
                className="button button-sm"
                disabled={scope.locked}
                onClick={() => setCreate(true)}
              >
                <Plus size={14} />
                Новый профиль
              </button>
            ) : null}
          </>
        }
      />
      <AgentNavigation current="profiles" />
      <div className="toolbar">
        <Tabs
          value={filters.params.get("state") || "active"}
          onValueChange={(state) =>
            filters.set({ state: state === "active" ? undefined : state })
          }
        >
          <TabsList>
            <TabsTrigger value="active">Действующие</TabsTrigger>
            <TabsTrigger value="archived">Архив</TabsTrigger>
            <TabsTrigger value="all">Все</TabsTrigger>
          </TabsList>
        </Tabs>
        <span className="toolbar-count">{profiles.length} в выборке</span>
      </div>
      <div className="profile-grid">
        {profiles.map((profile) => (
          <Link
            className="profile-card"
            prefetch={false}
            key={profile.id}
            href={`/app/profiles/${profile.id}`}
          >
            <div className="profile-card-heading">
              <Avatar name={profile.name} role={profile.role} agent />
              <div className="entity-main">
                <strong>{profile.name}</strong>
                <span className="muted">{label(profile.role)}</span>
              </div>
              <span className="profile-version">v{profile.version}</span>
            </div>
            <p className="profile-description muted">
              {profile.description || "Описание не добавлено"}
            </p>
            <div className="profile-card-meta">
              <span>
                {profile.state === "archived"
                  ? "В архиве"
                  : "Действующий профиль"}
              </span>
              <span>
                {providerLabels[profile.provider] ?? profile.provider}
              </span>
            </div>
            {profile.model ? (
              <span className="profile-model muted">
                Модель: {profile.model}
              </span>
            ) : null}
            <span className="profile-updated muted">
              Изменён {date(profile.updated_at)}
            </span>
          </Link>
        ))}
      </div>
      {!profiles.length ? (
        <EmptyState
          title="Профилей пока нет"
          description={
            filters.params.get("state") === "archived"
              ? "Архивные профили появятся здесь. Их можно восстановить с сохранением истории."
              : "Создайте профиль, чтобы сохранить роль, настройки и инструкции для новых сессий."
          }
          icon={<Users size={26} />}
          action={
            canManage ? (
              <button
                className="button button-secondary"
                onClick={() => setCreate(true)}
                disabled={scope.locked}
              >
                Создать профиль
              </button>
            ) : undefined
          }
        />
      ) : null}
      <p className="caption-note muted">
        Профиль хранится в этом проекте. Присутствие и действующие аренды
        показаны отдельно в сессиях. Подключение к провайдеру и запуск ещё не
        выполняются.
      </p>
      {create ? (
        <ProfileDialog
          page={page}
          onSuccess={onSuccess}
          open={create}
          onOpenChange={setCreate}
        />
      ) : null}
    </div>
  );
}
