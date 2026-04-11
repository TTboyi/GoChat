// src/utils/session.ts

// 这组工具函数解决的是“一个浏览器开多个标签页时，token 如何隔离”的问题。
// 当前实现思路是：
// 1. 每个标签页第一次加载时生成一个 tabId；
// 2. token 既存一份通用 key，也存一份 token_<tabId>；
// 3. 读取时优先取当前标签页专属 token。
export function getTabId(): string {
  let tabId = sessionStorage.getItem("tabId");
  if (!tabId) {
    tabId = Math.random().toString(36).substring(2, 10);
    sessionStorage.setItem("tabId", tabId);
  }
  return tabId;
}

// setToken 同时写入“当前标签页专属 key”和一个通用 fallback key。
export function setToken(token: string) {
  const tabId = getTabId();
  sessionStorage.setItem(`token_${tabId}`, token);
  sessionStorage.setItem("token", token);
}

// getToken 优先返回当前 tab 的 token，兼容旧逻辑时再回退到通用 key。
export function getToken(): string | null {
  const tabId = sessionStorage.getItem("tabId");
  const scoped = tabId ? sessionStorage.getItem(`token_${tabId}`) : null;
  const fallback = sessionStorage.getItem("token");
  return scoped || fallback;
}

// clearToken 会把当前 tab 相关的 token 一并清掉。
export function clearToken() {
  const tabId = sessionStorage.getItem("tabId");
  if (tabId) sessionStorage.removeItem(`token_${tabId}`);
  sessionStorage.removeItem("token");
}

// refresh token 目前按整个标签页会话共享存储。
export function setRefreshToken(token: string) {
  sessionStorage.setItem("refresh_token", token);
}

export function getRefreshToken(): string | null {
  return sessionStorage.getItem("refresh_token");
}

export function clearRefreshToken() {
  sessionStorage.removeItem("refresh_token");
}
