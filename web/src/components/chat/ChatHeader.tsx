import React from "react";
import type { SessionItem } from "../../types/chat";
import { toAbs } from "../../utils/chatUtils";

interface ChatHeaderProps {
  active: SessionItem | undefined;
  avatarVersion: number;
  groupMemberCount: number;
  groupNotice: string;
  isDark: boolean;
  onShowGroupMembers: () => void;
  onShowGroupInfo: () => void;
  // ✅ 点击好友头像/名字区域 → 打开好友资料弹窗
  onShowFriendProfile?: () => void;
}

const ChatHeader: React.FC<ChatHeaderProps> = ({
  active,
  avatarVersion,
  groupMemberCount,
  groupNotice,
  isDark,
  onShowGroupMembers,
  onShowGroupInfo,
  onShowFriendProfile,
}) => {
  const headerBg = isDark ? "bg-[#2e2e2e] border-black/20" : "bg-white border-gray-200";
  const textMain = isDark ? "text-gray-100" : "text-gray-800";
  const textSub  = isDark ? "text-gray-400" : "text-gray-500";
  const btnClass = isDark
    ? "px-3 py-1 text-sm rounded bg-white/10 hover:bg-white/20 text-gray-200"
    : "px-3 py-1 text-sm rounded bg-gray-100 hover:bg-gray-200 text-gray-700";

  return (
    <div className={`h-16 border-b px-5 flex items-center justify-between ${headerBg}`}>
      {active?.type === "group" ? (
        <div className="flex items-center space-x-3">
          {active.avatar ? (
            <img
              src={
                active.avatar.startsWith("http")
                  ? `${active.avatar}?v=${avatarVersion}`
                  : `${toAbs(active.avatar)}?v=${avatarVersion}`
              }
              className="w-10 h-10 rounded-md object-cover cursor-pointer"
              onClick={onShowGroupInfo}
              title="查看群资料"
            />
          ) : (
            <div
              className="w-10 h-10 bg-gray-400 rounded-md flex items-center justify-center cursor-pointer text-white font-bold"
              onClick={onShowGroupInfo}
              title="查看群资料"
            >
              群
            </div>
          )}
          <div className="flex flex-col leading-tight">
            <div className={`text-base font-semibold ${textMain}`}>{active.name}</div>
            <div className={`text-xs ${textSub}`}>成员 {groupMemberCount} 人</div>
            {groupNotice && (
              <div className={`text-xs ${textSub} truncate max-w-[300px]`}>
                公告：{groupNotice}
              </div>
            )}
          </div>
        </div>
      ) : (
        /* ✅ 一对一聊天：点击头像/名字区域打开好友资料 */
        <button
          onClick={active?.name ? onShowFriendProfile : undefined}
          className={`flex items-center space-x-3 bg-transparent border-none p-0
            ${active?.name ? "cursor-pointer" : "cursor-default"}
            hover:opacity-80 transition`}
          title={active?.name ? "查看好友资料" : undefined}
        >
          {active?.avatar && (
            <img
              src={
                active.avatar.startsWith("http")
                  ? `${active.avatar}?v=${avatarVersion}`
                  : `${toAbs(active.avatar)}?v=${avatarVersion}`
              }
              className="w-9 h-9 rounded-md object-cover"
              onError={(e) => { (e.target as HTMLImageElement).style.display = "none"; }}
            />
          )}
          <span className={`text-base font-semibold ${textMain}`}>
            {active?.name || "请选择会话"}
          </span>
        </button>
      )}

      {active?.type === "group" && (
        <button onClick={onShowGroupMembers} className={btnClass}>
          群成员
        </button>
      )}
    </div>
  );
};

export default ChatHeader;
