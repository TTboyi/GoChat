import React, { useEffect, useState } from "react";
import Modal from "./Modal";
import axios from "axios";

interface Member {
  userId: string;
  nickname: string;
  avatar: string;
}

interface RemoveGroupMembersModalProps {
  isOpen: boolean;
  onClose: () => void;
  groupId: string;
  onSuccess?: () => void;
}

const RemoveGroupMembersModal: React.FC<RemoveGroupMembersModalProps> = ({
  isOpen,
  onClose,
  groupId,
  onSuccess,
}) => {
  const [members, setMembers] = useState<Member[]>([]);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);

  // 加载群成员
  const loadMembers = async () => {
    try {
      const res = await axios.get("http://localhost:8000/group/getGroupMemberList", {
        params: { groupUuid: groupId },
      });
      if (res.data.members) {
        setMembers(res.data.members);
      }
    } catch (error) {
      console.error("加载成员失败", error);
    }
  };

  useEffect(() => {
    if (isOpen) {
      loadMembers();
      setSelectedIds([]);
    }
  }, [isOpen]);

  // 处理勾选
  const toggleSelection = (userId: string) => {
    setSelectedIds((prev) =>
      prev.includes(userId) ? prev.filter((id) => id !== userId) : [...prev, userId]
    );
  };

  // 移除成员
  const handleRemove = async () => {
    try {
      for (const uid of selectedIds) {
        await axios.post(
          "http://localhost:8000/group/removeGroupMember",
          new URLSearchParams({
            groupUuid: groupId,
            targetUserId: uid,
          }),
          {
            headers: { "Content-Type": "application/x-www-form-urlencoded" },
          }
        );
      }
      alert("移除成功");
      if (onSuccess) onSuccess();
      loadMembers(); // 刷新成员列表
    } catch (error) {
      console.error("移除失败", error);
      alert("移除失败");
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose}>
      <div className="w-[400px] p-6 flex flex-col space-y-4">
        <h2 className="text-xl font-bold text-center">移除群组人员</h2>

        <div className="max-h-[300px] overflow-y-auto space-y-2">
          {members.map((m) => (
            <div
              key={m.userId}
              className="flex items-center justify-between border p-2 rounded"
            >
              <div className="flex items-center gap-2">
                <img src={m.avatar} alt="" className="w-8 h-8 rounded-full" />
                <span>{m.nickname}</span>
              </div>
              <input
                type="checkbox"
                checked={selectedIds.includes(m.userId)}
                onChange={() => toggleSelection(m.userId)}
              />
            </div>
          ))}
        </div>

        <div className="flex justify-end">
          <button
            onClick={handleRemove}
            className="bg-red-500 text-white px-4 py-2 rounded hover:bg-red-600"
            disabled={selectedIds.length === 0}
          >
            移除所选人员
          </button>
        </div>
      </div>
    </Modal>
  );
};

export default RemoveGroupMembersModal;
