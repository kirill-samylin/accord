import { ProjectLink as Link } from "@/components/project-link";
import { Fact } from "@/components/common";
import type { ProfileSnapshot } from "@/lib/agent-profiles";
import { providerLabels } from "@/lib/agent-profiles";
export function SessionProfile({ profile }: { profile?: ProfileSnapshot }) {
  return (
    <section className="settings-section">
      <h2>Профиль этой сессии</h2>
      {profile ? (
        <>
          <dl>
            <Fact
              name="Профиль"
              value={
                <Link
                  prefetch={false}
                  className="inline-link"
                  href={`/app/profiles/${profile.id}`}
                >
                  {profile.name}
                </Link>
              }
            />
            <Fact
              name="ID"
              value={<span className="mono break">{profile.id}</span>}
            />
            <Fact name="Версия при открытии" value={profile.version} />
            <Fact
              name="Провайдер"
              value={providerLabels[profile.provider] ?? profile.provider}
            />
            <Fact name="Модель" value={profile.model || "Не указана"} />
          </dl>
          <p className="form-help">
            Снимок выбран при открытии сессии. Позднейшие изменения профиля на
            него не влияют. Подключение к провайдеру и процесс этим не
            подтверждаются.
          </p>
          <details>
            <summary>Описание и инструкции сессии</summary>
            <p className="profile-text">
              {profile.description || "Описание не добавлено"}
            </p>
            <p className="profile-text profile-instructions">
              {profile.instructions || "Инструкции не добавлены"}
            </p>
          </details>
        </>
      ) : (
        <p className="muted">
          Без профиля. Сессия открыта без явной привязки; совпадение имени или
          роли не создаёт связь.
        </p>
      )}
    </section>
  );
}
