import type { Message } from "../types/chat";
import { API_BASE } from "../config";

export const cn = (...a: Array<string | false | undefined>) =>
  a.filter(Boolean).join(" ");

/** 将相对路径补全为绝对 URL，若已是 http/https 则原样返回 */
export const toAbs = (rel?: string): string => {
  if (!rel) return "";
  if (rel.startsWith("http")) return rel;
  return `${API_BASE}${rel}`;
};

export const saveMessagesToStorage = (map: Record<string, Message[]>) => {
  try {
    localStorage.setItem("chat_messages", JSON.stringify(map));
  } catch {}
};

export const loadMessagesFromStorage = (): Record<string, Message[]> => {
  try {
    const raw = localStorage.getItem("chat_messages");
    return raw ? JSON.parse(raw) : {};
  } catch {
    return {};
  }
};

export const saveActiveId = (id: string) => {
  if (id) localStorage.setItem("chat_activeId", id);
};

export const loadActiveId = (): string => {
  return localStorage.getItem("chat_activeId") || "";
};
