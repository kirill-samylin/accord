import { Select } from "@/components/ui/select";
import type { ProfileSummary } from "@/lib/agent-profiles";
import { currentProfileChoice, profileChoice } from "@/lib/coordination";
export function ProfileChoice({
  value,
  role,
  profiles = [],
  onChange,
  id = "assignment-profile",
}: {
  value: string;
  role: string;
  profiles?: ProfileSummary[];
  onChange: (value: string) => void;
  id?: string;
}) {
  const current = currentProfileChoice(value, role, profiles);
  return (
    <label className="form-field" htmlFor={id}>
      Профиль исполнителя
      <Select
        id={id}
        name="profile"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        aria-describedby={`${id}-help`}
        aria-invalid={!current}
      >
        <option value="">Общая очередь роли</option>
        {!current ? (
          <option value={value} disabled>
            Выбранная версия недоступна
          </option>
        ) : null}
        {profiles
          .filter(
            (profile) => profile.state === "active" && profile.role === role,
          )
          .map((profile) => (
            <option key={profile.id} value={profileChoice(profile)}>
              {profile.name} · версия {profile.version} · {profile.id.slice(-8)}
            </option>
          ))}
      </Select>
      <span className="form-help" id={`${id}-help`}>
        {current
          ? "Назначение сохраняет выбранную версию инструкций. Смена профиля не запускает агента."
          : "Профиль изменён, архивирован или не подходит роли. Выберите актуальную версию явно."}
      </span>
    </label>
  );
}
