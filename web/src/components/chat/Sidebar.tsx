import React, { useState, useRef, useEffect } from "react";
import type { SessionItem } from "../../types/chat";
import { cn, toAbs } from "../../utils/chatUtils";

interface SidebarProps {
  user: { nickname?: string; avatar?: string; uuid?: string } | null;
  avatarVersion: number;
  sessions: SessionItem[];
  activeId: string;
  unreadCounts: Record<string, number>;
  onlineUsers: Set<string>;
  isDark: boolean;
  newApplyCount?: number;
  onToggleTheme: () => void;
  onSelectSession: (id: string) => void;
  onShowProfile: () => void;
  onLogout: () => void;
  onShowNewFriend: () => void;
  onShowAddFriend: () => void;
  onShowCreateGroup: () => void;
  onShowJoinGroup: () => void;
}

const Sidebar: React.FC<SidebarProps> = ({
  user,
  avatarVersion,
  sessions,
  activeId,
  unreadCounts,
  onlineUsers,
  isDark,
  newApplyCount = 0,
  onToggleTheme,
  onSelectSession,
  onShowProfile,
  onLogout,
  onShowNewFriend,
  onShowAddFriend,
  onShowCreateGroup,
  onShowJoinGroup,
}) => {
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!menuOpen) return;
    const handler = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [menuOpen]);

  const menuItems = [
    { label: "个人资料", action: onShowProfile },
    { label: "新朋友申请", action: onShowNewFriend, badge: newApplyCount },
    { label: "添加好友", action: onShowAddFriend },
    { label: "创建群聊", action: onShowCreateGroup },
    { label: "加入群聊", action: onShowJoinGroup },
    { divider: true },
    { label: "退出登录", action: onLogout, danger: true },
  ];

  // ===== 主题色 token =====
  const bg = isDark ? "bg-[#2e2e2e]" : "bg-gray-50";
  const border = isDark ? "border-black/20" : "border-gray-200";
  const textMain = isDark ? "text-gray-200" : "text-gray-800";
  const textSub = isDark ? "text-gray-400" : "text-gray-500";
  const hoverBg = isDark ? "hover:bg-[#3a3b3d]" : "hover:bg-gray-100";
  const activeBg = isDark ? "bg-[#3a3b3d]" : "bg-gray-200";
  const searchBg = isDark ? "bg-white/10" : "bg-gray-200";
  const dropBg = isDark ? "bg-[#3a3b3d] border-black/30" : "bg-white border-gray-200";
  const dropItem = isDark ? "hover:bg-white/10 text-gray-200" : "hover:bg-gray-100 text-gray-700";
  const dotBorder = isDark ? "border-[#2e2e2e]" : "border-gray-50";
  const avatarFallbackBg = isDark ? "bg-white/20" : "bg-gray-300";
  // ✅ 切换按钮：使用实色背景 + 边框，确保始终清晰可见
  const toggleBg = isDark
    ? "bg-amber-500/25 border border-amber-400/60 text-amber-300 hover:bg-amber-500/40"
    : "bg-indigo-100 border border-indigo-300 text-indigo-600 hover:bg-indigo-200";

  return (
    <aside className={`w-full md:w-[260px] ${bg} ${textMain} flex flex-col border-r ${border} h-full`}>
      {/* 顶部用户栏 */}
      <div className={`h-14 md:h-16 px-4 flex items-center border-b ${border} relative flex-shrink-0`} ref={menuRef}>
        <button
          className="flex items-center space-x-3 flex-1 min-w-0 bg-transparent border-none p-0"
          onClick={() => setMenuOpen((v) => !v)}
          title="点击查看菜单"
        >
          <div className="relative flex-shrink-0">
            {user?.avatar ? (
              <img
                src={`${toAbs(user.avatar)}?v=${avatarVersion}`}
                alt="me"
                className="w-9 h-9 rounded-md object-cover"
                onError={(e) => { (e.target as HTMLImageElement).style.display = "none"; }}
              />
            ) : (
              <div className={`w-9 h-9 rounded-md ${avatarFallbackBg} flex items-center justify-center text-sm font-bold`}>
                {user?.nickname?.[0] || "我"}
              </div>
            )}
            {/* 自己在线状态（绿色） */}
            <span className={`absolute -bottom-0.5 -right-0.5 w-2.5 h-2.5 bg-green-400 rounded-full border-2 ${dotBorder}`} />
          </div>
          <div className="leading-tight min-w-0 text-left">
            <div className={`font-semibold text-sm truncate ${textMain}`}>{user?.nickname || "未登录"}</div>
            <div className={`text-[10px] ${textSub} hidden md:block`}>点击头像显示菜单</div>
          </div>
        </button>

        {/* 移动端右侧加号按钮 */}
        <button
          className={`md:hidden w-8 h-8 flex items-center justify-center ${textSub} hover:opacity-80 bg-transparent border-none p-0`}
          onClick={(e) => { e.stopPropagation(); setMenuOpen((v) => !v); }}
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="w-5 h-5">
            <line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" />
          </svg>
        </button>

        {/* 下拉菜单 */}
        {menuOpen && (
          <div className={`absolute top-14 left-3 z-50 w-44 ${dropBg} border rounded-lg shadow-xl overflow-hidden py-1`}>
            {menuItems.map((item, i) =>
              "divider" in item ? (
                <div key={i} className={`h-px ${isDark ? "bg-black/20" : "bg-gray-100"} my-1`} />
              ) : (
                <button
                  key={i}
                  onClick={() => { item.action(); setMenuOpen(false); }}
                  className={cn(
                    `w-full text-left px-4 py-2.5 text-sm transition bg-transparent border-none ${dropItem} flex items-center justify-between`,
                    item.danger ? "!text-red-400" : ""
                  )}
                >
                  <span>{item.label}</span>
                  {"badge" in item && item.badge > 0 && (
                    <span className="bg-red-500 text-white text-[10px] rounded-full min-w-[16px] h-4 flex items-center justify-center px-1 font-bold">
                      {item.badge > 99 ? "99+" : item.badge}
                    </span>
                  )}
                </button>
              )
            )}
          </div>
        )}
      </div>

      {/* 搜索框 */}
      <div className="px-3 py-2 flex-shrink-0">
        <div className={`${searchBg} rounded-lg px-3 py-1.5 flex items-center gap-2`}>
          <svg viewBox="0 0 24 24" className={`w-3.5 h-3.5 ${textSub}`} fill="none" stroke="currentColor" strokeWidth="2">
            <circle cx="11" cy="11" r="8" /><line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
          <span className={`${textSub} text-xs`}>搜索</span>
        </div>
      </div>

      {/* 会话列表 */}
      <div className="flex-1 overflow-y-auto">
        {sessions.length === 0 && (
          <div className={`${textSub} text-sm p-4 text-center`}>暂无联系人</div>
        )}
        {sessions.map((s) => {
          const unread = unreadCounts[s.id] || 0;
          const isOnline = s.type === "user" && s.id !== user?.uuid && onlineUsers.has(s.id);
          return (
            <button
              key={s.id}
              onClick={() => onSelectSession(s.id)}
              className={cn(
                `w-full flex items-center px-3 py-3 md:py-2.5 transition active:opacity-70
                 bg-transparent border-none ${hoverBg}`,
                activeId === s.id && activeBg
              )}
            >
              <div className="relative flex-shrink-0">
                {s.avatar ? (
                  <img
                    src={`${toAbs(s.avatar)}?v=${avatarVersion}`}
                    alt={s.name}
                    className="w-11 h-11 md:w-10 md:h-10 rounded-md object-cover"
                    onError={(e) => { (e.target as HTMLImageElement).style.display = "none"; }}
                  />
                ) : (
                  <div className={`w-11 h-11 md:w-10 md:h-10 rounded-md ${avatarFallbackBg} flex items-center justify-center font-bold text-sm`}>
                    {s.name[0] || (s.type === "group" ? "群" : "友")}
                  </div>
                )}
                {unread > 0 && (
                  <span className="absolute -top-1 -right-1 bg-red-500 text-white text-[10px] rounded-full min-w-[16px] h-4 flex items-center justify-center px-1 font-bold">
                    {unread > 99 ? "99+" : unread}
                  </span>
                )}
                {s.type === "user" && s.id !== user?.uuid && (
                  <span
                    className={cn(
                      `absolute bottom-0 right-0 w-2.5 h-2.5 rounded-full border-2 ${dotBorder}`,
                      isOnline ? "bg-green-400" : "bg-gray-400"
                    )}
                  />
                )}
              </div>
              <div className="ml-3 min-w-0 flex-1 text-left">
                <div className={`text-sm font-semibold truncate ${textMain}`}>{s.name}</div>
                <div className={`text-[11px] ${textSub} truncate`}>
                  {s.type === "group" ? "群聊" : (isOnline ? "在线" : "离线")}
                </div>
              </div>
              {/* 移动端箭头 */}
              <svg viewBox="0 0 24 24" className={`w-4 h-4 ${textSub} flex-shrink-0 md:hidden`} fill="none" stroke="currentColor" strokeWidth="2">
                <polyline points="9 18 15 12 9 6" />
              </svg>
            </button>
          );
        })}
      </div>

      {/* 底部：日月模式切换按钮 */}
      <div className={`flex-shrink-0 px-4 py-3 border-t ${border} flex items-center justify-between`}>
        <span className={`text-xs ${textSub}`}>{isDark ? "深色模式" : "浅色模式"}</span>
        <button
          onClick={onToggleTheme}
          title={isDark ? "切换到浅色模式" : "切换到深色模式"}
          className={`w-9 h-9 rounded-full flex items-center justify-center transition border-none ${toggleBg}`}
        >
          {isDark ? (
            /* 太阳图标 */
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="w-5 h-5">
              <circle cx="12" cy="12" r="4" />
              <line x1="12" y1="2" x2="12" y2="4" />
              <line x1="12" y1="20" x2="12" y2="22" />
              <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
              <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
              <line x1="2" y1="12" x2="4" y2="12" />
              <line x1="20" y1="12" x2="22" y2="12" />
              <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
              <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
            </svg>
          ) : (
            /* 月亮图标 */
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="w-5 h-5">
              <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
            </svg>
          )}
        </button>
      </div>
    </aside>
  );
};

export default Sidebar;
