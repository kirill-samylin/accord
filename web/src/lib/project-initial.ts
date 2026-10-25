// First letter or digit of a project name for its avatar; symbols like "<"
// or emoji make a confusing badge.
export function projectInitial(name: string | undefined, fallback = "A") {
  const match = /[\p{L}\p{N}]/u.exec(name ?? "");
  return match ? match[0].toUpperCase() : fallback;
}
