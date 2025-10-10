// src/utils/session.ts

export function getTabId(): string {
  let tabId = sessionStorage.getItem("tabId");
  if (!tabId) {
    tabId = Math.random().toString(36).substring(2, 10);
    sessionStorage.setItem("tabId", tabId);
  }
  return tabId;
}

export function setToken(token: string) {
  const tabId = getTabId();
  console.log("💾 保存 token:", token, "tabId:", tabId);
  sessionStorage.setItem(`token_${tabId}`, token);
  sessionStorage.setItem("token", token);
}

export function getToken(): string | null {
  const tabId = sessionStorage.getItem("tabId");
  const scoped = tabId ? sessionStorage.getItem(`token_${tabId}`) : null;
  const fallback = sessionStorage.getItem("token");
  console.log("🔍 getToken() 返回:", scoped || fallback);
  return scoped || fallback;
}

export function clearToken() {
  const tabId = sessionStorage.getItem("tabId");
  if (tabId) sessionStorage.removeItem(`token_${tabId}`);
  sessionStorage.removeItem("token");
}
