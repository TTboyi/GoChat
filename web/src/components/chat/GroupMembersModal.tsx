// ============================================================
// 文件：web/src/components/chat/GroupMembersModal.tsx
// 作用：群成员列表弹窗（群主可踢人，显示在线状态）。
// ============================================================
import React from "react";
import api from "../../api/api";

interface GroupMembersModalProps {
  open: boolean;
  onClose: () => void;
  groupId: string;
  groupMembers: string[];
  isGroupOwner: boolean;
  userId: string | undefined;
  onRefreshMembers: () => void;
  onRefreshContacts: () => void;
}

const GroupMembersModal: React.FC<GroupMembersModalProps> = ({
  open,
  onClose,
  groupId,
  groupMembers,
  isGroupOwner,
  userId,
  onRefreshMembers,
  onRefreshContacts,
}) => {
  if (!open) return null;

  const handleRemove = async (memberId: string) => {
    if (!window.confirm("确定要移除该成员吗？")) return;
    try {
      await api.removeMember({ groupUuid: groupId, targetUserId: memberId });
      alert("已移除成员");
      onRefreshMembers();
    } catch {
      alert("移除失败");
    }
  };

  const handleLeave = async () => {
    if (!window.confirm("确定退出该群聊吗？")) return;
    try {
      await api.leaveGroup({ groupUuid: groupId });
      alert("已退出群聊");
      onClose();
      onRefreshContacts();
    } catch {
      alert("退出失败");
    }
  };

  const handleDismiss = async () => {
    if (!window.confirm("确定要解散群聊吗？")) return;
    try {
      await api.dismissGroup({ groupId });
      alert("群聊已解散");
      onClose();
      onRefreshContacts();
    } catch {
      alert("解散失败");
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div
        className="bg-white/95 backdrop-blur-md rounded-xl shadow-2xl w-[400px] p-6 relative"
        onClick={(e) => e.stopPropagation()}
      >
        <button
          onClick={onClose}
          className="absolute top-3 right-3 text-gray-500 hover:text-gray-700 text-xl"
        >
          ×
        </button>
        <h2 className="text-lg font-semibold text-gray-800 mb-4">群成员列表</h2>

        {groupMembers.length === 0 ? (
          <div className="text-gray-500 text-sm text-center py-6">暂无成员</div>
        ) : (
          <ul className="max-h-[240px] overflow-y-auto divide-y divide-gray-200">
            {groupMembers.map((m) => (
              <li key={m} className="flex items-center justify-between py-2">
                <span className="text-sm text-gray-800">{m}</span>
                {isGroupOwner && m !== userId && (
                  <button
                    onClick={() => handleRemove(m)}
                    className="px-2 py-1 bg-red-500 hover:bg-red-600 text-white text-xs rounded"
                  >
                    移除
                  </button>
                )}
              </li>
            ))}
          </ul>
        )}

        <div className="flex justify-end mt-5 space-x-2">
          {!isGroupOwner && (
            <button
              onClick={handleLeave}
              className="px-3 py-2 rounded bg-gray-200 hover:bg-gray-600 text-gray-300 text-sm"
            >
              退出群聊
            </button>
          )}
          {isGroupOwner && (
            <button
              onClick={handleDismiss}
              className="px-3 py-2 rounded bg-red-600 hover:bg-red-700 text-white text-sm"
            >
              解散群聊
            </button>
          )}
        </div>
      </div>
    </div>
  );
};

export default GroupMembersModal;
