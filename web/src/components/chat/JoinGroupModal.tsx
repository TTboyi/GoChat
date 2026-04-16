// ============================================================
// 文件：web/src/components/chat/JoinGroupModal.tsx
// 作用：通过群号（6位数字）搜索并加入群聊的弹窗。
// ============================================================
import React, { useState } from "react";
import api from "../../api/api";

interface JoinGroupModalProps {
  open: boolean;
  onClose: () => void;
  onJoined: () => void;
}

const JoinGroupModal: React.FC<JoinGroupModalProps> = ({
  open,
  onClose,
  onJoined,
}) => {
  const [groupId, setGroupId] = useState("");

  if (!open) return null;

  const handleJoin = async () => {
    if (!groupId.trim()) return alert("请输入群聊 UUID");
    try {
      const res = await api.enterGroup({ groupId: groupId.trim() });
      alert(res.data?.message || "申请成功");
      setGroupId("");
      onClose();
      onJoined();
    } catch {
      alert("加入失败");
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-[#2e2e2e] w-[360px] rounded-xl shadow-lg p-6 space-y-4">
        <h2 className="text-lg font-semibold text-gray-200">加入群聊</h2>
        <input
          placeholder="输入群聊 UUID"
          className="w-full bg-[#3a3b3d] rounded px-3 py-2 text-gray-200 outline-none"
          value={groupId}
          onChange={(e) => setGroupId(e.target.value)}
        />
        <div className="flex justify-end space-x-2 pt-2">
          <button
            onClick={onClose}
            className="px-3 py-2 rounded bg-gray-500 hover:bg-gray-600 text-sm text-white"
          >
            取消
          </button>
          <button
            onClick={handleJoin}
            className="px-3 py-2 rounded bg-green-600 hover:bg-green-700 text-sm text-white"
          >
            加入
          </button>
        </div>
      </div>
    </div>
  );
};

export default JoinGroupModal;
