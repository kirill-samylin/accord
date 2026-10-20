"use client";
import { useState, type FormEvent } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  profileDraft,
  profileDraftErrors,
  profileFields,
  profileVersionChanged,
  type ProfileDraft,
} from "@/lib/agent-profiles";
import {
  actionBody,
  actionID,
  type MutationProps,
} from "../mutations/contracts";
import { ActionNotice } from "../mutations/ActionNotice";
import { useAction } from "../mutations/useAction";
import { ProfileFields } from "./profile-fields";
export function ProfileDialog({
  page,
  onSuccess,
  open,
  onOpenChange,
}: MutationProps & {
  open: boolean;
  onOpenChange: (value: boolean) => void;
}) {
  const [draft, setDraft] = useState(() => profileDraft(page.profile));
  const [errors, setErrors] = useState<Record<string, string>>({});
  const editing = Boolean(page.profile);
  const operation = editing ? "profile:update" : "profile:create";
  const action = useAction(page, (result) => {
    onSuccess(result);
    onOpenChange(false);
  });
  const changed = profileVersionChanged(draft, action.page.profile);
  const canManage =
    action.page.can_manage_profiles && action.page.project?.can_write !== false;
  function change(name: keyof ProfileDraft, value: string) {
    setDraft((previous) => ({ ...previous, [name]: value }));
    setErrors((previous) => ({ ...previous, [name]: "" }));
  }
  function submit(event: FormEvent) {
    event.preventDefault();
    const validation = profileDraftErrors(draft);
    setErrors(validation);
    if (Object.keys(validation).length || changed || !canManage) return;
    void action.send(
      editing ? `/web/profiles/${page.profile!.id}/update` : "/web/profiles",
      actionBody(action.page, operation, profileFields(draft, editing)),
    );
  }
  const pending = action.phase === "sending" || action.phase === "uncertain";
  return (
    <Dialog
      open={open}
      onOpenChange={(value) => {
        if (!pending) onOpenChange(value);
      }}
    >
      <DialogContent
        className="profile-dialog"
        aria-describedby="profile-form-help"
      >
        <DialogHeader>
          <DialogTitle>
            {editing ? "Изменить профиль" : "Новый профиль"}
          </DialogTitle>
        </DialogHeader>
        <p className="form-help" id="profile-form-help">
          Профиль хранит настройки для явного выбора клиентом. Открытые сессии
          сохраняют свою версию инструкций.
        </p>
        <form className="mutation-form" onSubmit={submit}>
          <fieldset
            disabled={action.locked || !canManage || action.page.can_continue}
          >
            <ProfileFields
              draft={draft}
              change={change}
              errors={{ ...action.fieldErrors, ...errors }}
            />
            {changed ? (
              <div className="notice" role="status">
                <p>
                  На сервере уже версия {action.page.profile?.version}. Черновик
                  сохранён на основе версии {draft.version}.
                </p>
                <button
                  type="button"
                  className="button button-secondary"
                  onClick={() =>
                    change("version", String(action.page.profile!.version))
                  }
                >
                  Применить черновик к версии {action.page.profile?.version}
                </button>
              </div>
            ) : null}
            <div className="form-actions">
              {editing ? (
                <span className="form-help">
                  Основа: версия {draft.version}
                </span>
              ) : null}
              <button
                className="button"
                type="submit"
                disabled={changed || !actionID(action.page, operation)}
              >
                {action.busy
                  ? "Сохраняем…"
                  : editing
                    ? "Сохранить изменения"
                    : "Создать профиль"}
              </button>
            </div>
          </fieldset>
          <ActionNotice
            message={action.message}
            uncertain={action.phase === "uncertain"}
            busy={action.busy}
            onRetry={() => void action.retry()}
          />
          {editing && !pending ? (
            <button
              className="button button-ghost"
              type="button"
              onClick={() =>
                void action.refresh(`/app/profiles/${page.profile!.id}`)
              }
            >
              Проверить актуальную версию
            </button>
          ) : null}
        </form>
      </DialogContent>
    </Dialog>
  );
}
