// ============================================================
// 文件：web/src/components/chat/SearchHistoryModal.tsx
// 作用：搜索当前会话历史消息的弹窗（关键词搜索，高亮匹配结果）。
// ============================================================
import React, { useState, useCallback, useRef } from "react";
import type { Message } from "../../types/chat";
import { toAbs } from "../../utils/chatUtils";

interface SearchHistoryModalProps {
  open: boolean;
  onClose: () => void;
  messages: Message[];
  userId?: string;
}

function useDebounce<T>(value: T, delay: number): T {
  const [debounced, setDebounced] = useState(value);
  const timer = useRef<ReturnType<typeof setTimeout>>();
  React.useEffect(() => {
    clearTimeout(timer.current);
    timer.current = setTimeout(() => setDebounced(value), delay);
    return () => clearTimeout(timer.current);
  }, [value, delay]);
  return debounced;
}

const formatTime = (createdAt?: number | string) => {
  if (!createdAt) return "";
  const ts = typeof createdAt === "number" ? createdAt * 1000 : Date.parse(createdAt as string);
  const d = new Date(ts);
  return d.toLocaleDateString("zh-CN") + " " + d.toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" });
};

const SearchHistoryModal: React.FC<SearchHistoryModalProps> = ({ open, onClose, messages, userId }) => {
  const [query, setQuery] = useState("");
  const debouncedQuery = useDebounce(query, 250);

  const filtered = React.useMemo(() => {
    if (!debouncedQuery.trim()) return [];
    const q = debouncedQuery.toLowerCase();
    return messages
      .filter((m) => !m.isRecalled && (
        m.content?.toLowerCase().includes(q) ||
        m.fileName?.toLowerCase().includes(q) ||
        m.sendName?.toLowerCase().includes(q)
      ))
      .slice()
      .reverse()
      .slice(0, 50);
  }, [debouncedQuery, messages]);

  if (!open) return null;

  return (
    <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50" onClick={onClose}>
      <div
        className="bg-white w-[480px] max-h-[70vh] rounded-xl shadow-2xl flex flex-col overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="px-5 pt-5 pb-3 border-b border-gray-100">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-base font-semibold text-gray-800">搜索历史消息</h2>
            <button onClick={onClose} className="text-gray-400 hover:text-gray-600 text-xl leading-none">×</button>
          </div>
          <div className="relative">
            <svg className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/>
            </svg>
            <input
              autoFocus
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="搜索消息内容、文件名..."
              className="w-full pl-9 pr-4 py-2 text-sm border border-gray-200 rounded-lg outline-none focus:border-green-400 transition"
            />
          </div>
        </div>

        {/* Results */}
        <div className="flex-1 overflow-y-auto">
          {!debouncedQuery.trim() ? (
            <div className="text-center text-gray-400 text-sm py-10">输入关键词搜索</div>
          ) : filtered.length === 0 ? (
            <div className="text-center text-gray-400 text-sm py-10">未找到相关消息</div>
          ) : (
            <div className="divide-y divide-gray-50">
              {filtered.map((m, i) => {
                const isSelf = m.sendId === userId;
                return (
                  <div key={m.uuid || i} className="px-5 py-3 hover:bg-gray-50 transition">
                    <div className="flex items-start gap-3">
                      {/* 头像 */}
                      <div className="flex-shrink-0 w-8 h-8 rounded-md overflow-hidden bg-gray-200 flex items-center justify-center">
                        {(isSelf ? undefined : m.sendAvatar) ? (
                          <img src={toAbs(m.sendAvatar)} className="w-full h-full object-cover" alt="" />
                        ) : (
                          <span className="text-xs font-bold text-gray-500">
                            {isSelf ? "我" : (m.sendName || "?")[0]}
                          </span>
                        )}
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-0.5">
                          <span className="text-xs font-semibold text-gray-700">
                            {isSelf ? "我" : (m.sendName || m.sendId)}
                          </span>
                          <span className="text-[10px] text-gray-400">{formatTime(m.createdAt)}</span>
                        </div>
                        {m.type === 1 ? (
                          <div className="text-xs text-blue-600 flex items-center gap-1">
                            <span>📎</span>
                            <span className="truncate">{m.fileName || "文件"}</span>
                          </div>
                        ) : (
                          <p className="text-sm text-gray-700 line-clamp-2 break-words">{m.content}</p>
                        )}
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default SearchHistoryModal;
