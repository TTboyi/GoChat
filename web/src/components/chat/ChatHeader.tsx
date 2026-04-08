import React from "react";
import type { SessionItem } from "../../types/chat";
import { toAbs } from "../../utils/chatUtils";

interface ChatHeaderProps {
  active: SessionItem | undefined;
  avatarVersion: number;
  groupMemberCount: number;
  groupNotice: string;
  onShowGroupMembers: () => void;
  onShowGroupInfo: () => void;
}

const ChatHeader: React.FC<ChatHeaderProps> = ({
  active,
  avatarVersion,
  groupMemberCount,
  groupNotice,
  onShowGroupMembers,
  onShowGroupInfo,
}) => {
  return (
    <div className="h-16 bg-[#f0f0f0] border-b border-gray-200 px-5 flex items-center justify-between">
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
            <div className="text-base font-semibold text-black">{active.name}</div>
            <div className="text-xs text-gray-600">成员 {groupMemberCount} 人</div>
            {groupNotice && (
              <div className="text-xs text-gray-500 truncate max-w-[300px]">
                公告：{groupNotice}
              </div>
            )}
          </div>
        </div>
      ) : (
        <div className="text-base font-semibold text-black">
          {active?.name ? (
            <div className="flex items-center space-x-3">
              {active.avatar && (
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
              <span>{active.name}</span>
            </div>
          ) : "请选择会话"}
        </div>
      )}

      {active?.type === "group" && (
        <button
          onClick={onShowGroupMembers}
          className="px-3 py-1 text-sm rounded bg-gray-200 hover:bg-gray-300"
        >
          群成员
        </button>
      )}
    </div>
  );
};

export default ChatHeader;
