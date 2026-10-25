// Drafts belong to the current browser login. Project scope alone cannot
// distinguish two people who use the same tab in succession.
export function clearSessionDrafts() {
  try {
    const storage = window.sessionStorage;
    const keys: string[] = [];
    for (let index = 0; index < storage.length; index++) {
      const key = storage.key(index);
      if (
        key &&
        (key === "accord:create-project-name" ||
          key.startsWith("accord:create-draft:") ||
          key.startsWith("accord:pending-task:"))
      )
        keys.push(key);
    }
    for (const key of keys) storage.removeItem(key);
  } catch {
    // Optional browser storage can be unavailable or blocked.
  }
}
