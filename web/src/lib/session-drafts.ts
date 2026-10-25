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
          key.startsWith("accord:answer-draft:") ||
          key.startsWith("accord:pending-task:"))
      )
        keys.push(key);
    }
    for (const key of keys) storage.removeItem(key);
  } catch {
    // Optional browser storage can be unavailable or blocked.
  }
}
// A question answer draft survives reload in sessionStorage, scoped by project
// scope and question key; it is removed once the answer is saved and on
// sign-out (clearSessionDrafts).
export function answerDraftKey(scope: string, taskKey: string) {
  return `accord:answer-draft:${scope}:${taskKey}`;
}
export function loadAnswerDraft(scope: string, taskKey: string) {
  try {
    return window.sessionStorage.getItem(answerDraftKey(scope, taskKey)) ?? "";
  } catch {
    return "";
  }
}
export function saveAnswerDraft(scope: string, taskKey: string, text: string) {
  try {
    const key = answerDraftKey(scope, taskKey);
    if (text) window.sessionStorage.setItem(key, text);
    else window.sessionStorage.removeItem(key);
  } catch {
    // Optional browser storage can be unavailable or blocked.
  }
}
