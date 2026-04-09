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

// ===== 备注管理（localStorage，key: contact_remarks）=====
const REMARK_KEY = "contact_remarks";

export const loadRemarks = (): Record<string, string> => {
  try {
    const raw = localStorage.getItem(REMARK_KEY);
    return raw ? JSON.parse(raw) : {};
  } catch {
    return {};
  }
};

export const saveRemark = (userId: string, remark: string) => {
  const map = loadRemarks();
  if (remark) {
    map[userId] = remark;
  } else {
    delete map[userId];
  }
  try {
    localStorage.setItem(REMARK_KEY, JSON.stringify(map));
  } catch {}
};

export const getRemark = (userId: string): string => {
  return loadRemarks()[userId] || "";
};

/** 删除好友时同时清除其备注和本地消息 */
export const clearContactData = (
  userId: string,
  messagesMap: Record<string, Message[]>
): Record<string, Message[]> => {
  // 清除备注
  const remarks = loadRemarks();
  delete remarks[userId];
  try { localStorage.setItem(REMARK_KEY, JSON.stringify(remarks)); } catch {}

  // 返回清除该联系人后的消息 map
  const newMap = { ...messagesMap };
  delete newMap[userId];
  return newMap;
};
