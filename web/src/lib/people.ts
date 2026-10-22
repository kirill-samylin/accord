// Stable visual identity for owner labels. Colours are a recognition aid only;
// every place that uses them also prints the owner name.
export const PERSON_TONES = 8;

export function personTone(owner?: string): number {
  if (!owner) return 0;
  let hash = 0;
  for (const char of owner.trim().toLowerCase())
    hash = (hash * 31 + char.codePointAt(0)!) >>> 0;
  return hash % PERSON_TONES;
}

export function personClass(owner?: string) {
  return `person-tone-${personTone(owner)}`;
}

export function since(value?: string, now = Date.now()): string {
  if (!value) return "";
  const at = Date.parse(value);
  if (!Number.isFinite(at)) return "";
  const seconds = Math.max(0, Math.round((now - at) / 1000));
  if (seconds < 60) return "<1 мин";
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return `${minutes} мин`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours} ч ${minutes % 60} мин`;
  return `${Math.floor(hours / 24)} д ${hours % 24} ч`;
}

export function ago(value?: string, now = Date.now()): string {
  const span = since(value, now);
  if (!span) return "";
  return span === "<1 мин" ? "только что" : `${span} назад`;
}

export function percent(score: number) {
  return `${Math.round(Math.min(1, Math.max(0, score)) * 100)}%`;
}

export const reasonLabels: Record<string, string> = {
  title: "заголовок",
  description: "описание",
  areas: "те же области",
  external_id: "тот же внешний ID",
};
