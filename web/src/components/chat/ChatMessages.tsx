import React, { useRef, useEffect, useCallback } from "react";
import type { RefObject } from "react";
import type { Message, SessionItem } from "../../types/chat";
import { cn, toAbs } from "../../utils/chatUtils";

interface ChatMessagesProps {
  messages: Message[];
  activeId: string;
  active: SessionItem | undefined;
  userId: string | undefined;
  avatarVersion: number;
  userAvatar?: string;
  listRef: RefObject<HTMLDivElement | null>;
  hasMore?: boolean;
  isDark?: boolean;
  onLoadMore?: () => void;
  onRecall?: (msg: Message) => void;
}

// ---- Telegram 风格已读勾 ----
const ReadTick: React.FC<{ isRead: boolean }> = ({ isRead }) => (
  <span className="inline-flex items-center ml-1" title={isRead ? "已读" : "已发送"}>
    {isRead ? (
      <svg width="16" height="10" viewBox="0 0 16 10" fill="none">
        <path d="M1 5L5 9L11 1" stroke="#4fc3f7" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"/>
        <path d="M5 5L9 9L15 1" stroke="#4fc3f7" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"/>
      </svg>
    ) : (
      <svg width="10" height="10" viewBox="0 0 10 10" fill="none">
        <path d="M1 5L4 8L9 2" stroke="#888" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"/>
      </svg>
    )}
  </span>
);

// ---- 文件类型图标 ----
const FileIcon: React.FC<{ fileType?: string }> = ({ fileType }) => {
  const t = (fileType || "").toLowerCase();
  const color =
    t.includes("pdf") ? "text-red-500" :
    t.includes("word") || t.includes("doc") ? "text-blue-600" :
    t.includes("sheet") || t.includes("excel") || t.includes("xls") ? "text-green-600" :
    t.includes("zip") || t.includes("rar") || t.includes("7z") ? "text-yellow-600" :
    "text-gray-500";
  return (
    <svg className={`w-8 h-8 flex-shrink-0 ${color}`} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
      <polyline points="14 2 14 8 20 8"/>
    </svg>
  );
};

// ---- 右键菜单 ----
interface CtxMenuProps { x: number; y: number; canRecall: boolean; onRecall: () => void; onClose: () => void; }
const CtxMenu: React.FC<CtxMenuProps> = ({ x, y, canRecall, onRecall, onClose }) => {
  useEffect(() => {
    const h = () => onClose();
    document.addEventListener("click", h);
    return () => document.removeEventListener("click", h);
  }, [onClose]);

  if (!canRecall) return null;
  return (
    <div
      className="fixed bg-white border border-gray-200 rounded-lg shadow-xl z-[9999] py-1 min-w-[110px]"
      style={{ left: x, top: y }}
      onClick={(e) => e.stopPropagation()}
    >
      <button
        className="block w-full text-left px-4 py-2 text-sm text-red-500 hover:bg-gray-50"
        onClick={() => { onRecall(); onClose(); }}
      >
        撤回消息
      </button>
    </div>
  );
};

const formatBytes = (bytes?: string): string => {
  if (!bytes) return "";
  const n = parseInt(bytes, 10);
  if (isNaN(n)) return bytes;
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
};

const ChatMessages: React.FC<ChatMessagesProps> = ({
  messages,
  activeId,
  active,
  userId,
  avatarVersion,
  userAvatar,
  listRef,
  hasMore = false,
  isDark = true,
  onLoadMore,
  onRecall,
}) => {
  const [contextMenu, setContextMenu] = React.useState<{x: number; y: number; msg: Message} | null>(null);
  // 用于加载更多时保持滚动位置
  const prevScrollHeight = useRef(0);
  const isLoadingMore = useRef(false);

  const handleContextMenu = (e: React.MouseEvent, msg: Message) => {
    if (msg.sendId !== userId || msg.isRecalled) return;
    e.preventDefault();
    setContextMenu({ x: e.clientX, y: e.clientY, msg });
  };

  const canRecall = (msg: Message) => {
    if (!msg.createdAt || msg.isRecalled) return false;
    const ts = typeof msg.createdAt === "number" ? msg.createdAt * 1000 : Date.parse(msg.createdAt as string);
    return Date.now() - ts < 10 * 60 * 1000;
  };

  // 加载更多时保持滚动位置
  const handleLoadMore = useCallback(() => {
    if (!listRef.current) return;
    prevScrollHeight.current = listRef.current.scrollHeight;
    isLoadingMore.current = true;
    onLoadMore?.();
  }, [onLoadMore, listRef]);

  // 加载更多后恢复滚动位置
  useEffect(() => {
    if (isLoadingMore.current && listRef.current) {
      const newScrollHeight = listRef.current.scrollHeight;
      listRef.current.scrollTop = newScrollHeight - prevScrollHeight.current;
      isLoadingMore.current = false;
    }
  }, [messages, listRef]);

  const formatTime = (createdAt?: number | string) => {
    if (!createdAt) return "";
    const ts = typeof createdAt === "number" ? createdAt * 1000 : Date.parse(createdAt as string);
    const d = new Date(ts);
    const now = new Date();
    const isToday =
      d.getFullYear() === now.getFullYear() &&
      d.getMonth() === now.getMonth() &&
      d.getDate() === now.getDate();
    const hm = d.toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit", hour12: false });
    if (isToday) return hm;
    const ymd = `${d.getFullYear()}/${String(d.getMonth() + 1).padStart(2, "0")}/${String(d.getDate()).padStart(2, "0")}`;
    return `${ymd} ${hm}`;
  };

  const getAvatarSrc = (url?: string) => {
    if (!url) return "";
    const abs = toAbs(url);
    return `${abs}?v=${avatarVersion}`;
  };

  return (
    <div ref={listRef} className={`flex-1 overflow-y-auto px-6 py-4 ${isDark ? "bg-[#1e1e1e]" : "bg-[#eaeaea]"}`}>
      {/* 加载更多 */}
      {hasMore && (
        <div className="flex justify-center mb-4">
          <button
            onClick={handleLoadMore}
            className="px-4 py-1.5 text-xs text-gray-500 bg-white rounded-full shadow-sm hover:bg-gray-50 border border-gray-200 transition"
          >
            查看更多历史消息
          </button>
        </div>
      )}

      {(!activeId || messages.length === 0) && !hasMore && (
        <div className="text-gray-400 text-sm text-center mt-16">暂无消息</div>
      )}

      {messages.map((m, idx) => {
        const isSelf = m.sendId === userId;

        // 撤回提示
        if (m.isRecalled) {
          return (
            <div key={m.uuid || idx} className="flex justify-center my-2">
              <span className="text-xs text-gray-400 bg-gray-200/80 px-3 py-1 rounded-full">
                {isSelf ? "你" : (m.sendName || "对方")} 撤回了一条消息
              </span>
            </div>
          );
        }

        const isRead = !!m.readAt;
        const selfAvatarUrl = userAvatar || m.sendAvatar;

        // 头像 URL（对方）
        const peerAvatarUrl = m.sendAvatar ? toAbs(m.sendAvatar) : "";

        // 判断是否是图片消息
        const isImage = m.type === 1 && (
          m.fileType?.startsWith("image/") ||
          /\.(png|jpg|jpeg|gif|webp|svg)$/i.test(m.url || m.content || "")
        );

        return (
          <div key={m.uuid || idx} className="mb-3">
            {/* 群聊：发送方昵称 */}
            {active?.type === "group" && !isSelf && (
              <div className="text-xs text-gray-500 ml-12 mb-0.5">
                {m.sendName || m.sendId}
              </div>
            )}

            <div
              className={cn("flex items-end gap-2", isSelf ? "justify-end" : "justify-start")}
              onContextMenu={(e) => handleContextMenu(e, m)}
            >
              {/* 对方头像（静态，不可点击；好友资料请点击顶部标题栏） */}
              {!isSelf && (
                <div className="flex-shrink-0 self-start mt-1">
                  {peerAvatarUrl ? (
                    <img
                      src={`${peerAvatarUrl}?v=${avatarVersion}`}
                      alt={m.sendName}
                      className="w-8 h-8 rounded-md object-cover"
                      onError={(e) => { (e.currentTarget as HTMLImageElement).src = ""; (e.currentTarget as HTMLImageElement).style.display = "none"; }}
                    />
                  ) : (
                    <div className="w-8 h-8 rounded-md bg-gray-400 flex items-center justify-center text-xs text-white font-bold">
                      {(m.sendName || m.sendId || "?")[0].toUpperCase()}
                    </div>
                  )}
                </div>
              )}

              {/* 消息气泡 */}
              <div
                className={cn(
                  "max-w-[60%] rounded-2xl text-sm shadow-sm",
                  isSelf
                    ? isDark
                      ? "bg-[#1e5e30] text-gray-100 rounded-br-sm"
                      : "bg-[#95ec69] text-black rounded-br-sm"
                    : isDark
                      ? "bg-[#3a3b3d] text-gray-100 rounded-bl-sm"
                      : "bg-white text-gray-900 rounded-bl-sm"
                )}
              >
                {/* 文件消息（非图片） */}
                {m.type === 1 && !isImage ? (() => {
                  const ts = m.createdAt
                    ? (typeof m.createdAt === "number" ? m.createdAt * 1000 : Date.parse(m.createdAt as string))
                    : 0;
                  const expired = ts > 0 && Date.now() - ts > 60 * 60 * 1000;
                  return expired ? (
                    <div className="flex items-center gap-3 px-3 py-2.5">
                      <FileIcon fileType={m.fileType} />
                      <div className="min-w-0">
                        <div className="text-sm font-medium truncate max-w-[180px] text-gray-400">
                          {m.fileName || "文件"}
                        </div>
                        <div className="text-[11px] text-red-400 mt-0.5">文件已过期</div>
                      </div>
                    </div>
                  ) : (
                  <a
                    href={toAbs(m.url || m.content)}
                    download={m.fileName || true}
                    target="_blank"
                    rel="noreferrer"
                    className="flex items-center gap-3 px-3 py-2.5 hover:opacity-80 transition"
                    title="点击下载"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <FileIcon fileType={m.fileType} />
                    <div className="min-w-0">
                      <div className="text-sm font-medium truncate max-w-[180px]">
                        {m.fileName || "文件"}
                      </div>
                      <div className="text-[11px] text-gray-500 flex items-center gap-1">
                        <span>{(m.fileType || "").split("/")[1]?.toUpperCase() || "FILE"}</span>
                        {m.fileSize && <span>· {formatBytes(m.fileSize)}</span>}
                      </div>
                    </div>
                    <svg className="w-4 h-4 flex-shrink-0 text-gray-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                      <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                      <polyline points="7 10 12 15 17 10"/>
                      <line x1="12" y1="15" x2="12" y2="3"/>
                    </svg>
                  </a>
                  );
                })()
                : m.type === 1 && isImage ? (
                  <div className="p-1">
                    <img
                      src={toAbs(m.url || m.content)}
                      className="max-w-[200px] max-h-[200px] rounded-xl object-contain"
                      alt="图片"
                    />
                  </div>
                ) : (
                  <div className="px-3 py-2" style={{ whiteSpace: "pre-wrap", wordBreak: "break-word" }}>
                    {m.content}
                  </div>
                )}

                {/* 时间 + 已读状态 */}
                <div className={cn("flex items-center px-3 pb-1.5", isSelf ? "justify-end" : "justify-start")}>
                  <span className="text-[10px] opacity-50">{formatTime(m.createdAt)}</span>
                  {isSelf && active?.type !== "group" && <ReadTick isRead={isRead} />}
                </div>
              </div>

              {/* 自己头像 */}
              {isSelf && (
                <div className="flex-shrink-0 self-start mt-1">
                  {selfAvatarUrl ? (
                    <img
                      src={`${toAbs(selfAvatarUrl)}?v=${avatarVersion}`}
                      className="w-8 h-8 rounded-md object-cover"
                      alt="me"
                      onError={(e) => { (e.currentTarget as HTMLImageElement).style.display = "none"; }}
                    />
                  ) : (
                    <div className="w-8 h-8 rounded-md bg-blue-400 flex items-center justify-center text-xs text-white font-bold">我</div>
                  )}
                </div>
              )}
            </div>
          </div>
        );
      })}

      {/* 右键菜单 */}
      {contextMenu && (
        <CtxMenu
          x={contextMenu.x}
          y={contextMenu.y}
          canRecall={canRecall(contextMenu.msg)}
          onRecall={() => onRecall?.(contextMenu.msg)}
          onClose={() => setContextMenu(null)}
        />
      )}
    </div>
  );
};

export default ChatMessages;
