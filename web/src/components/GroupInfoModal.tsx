import React, { useEffect, useState } from "react";
import api from "../api/api";
import { useAuth } from "../context/AuthContext";

interface GroupInfoModalProps {
  open: boolean;
  onClose: () => void;
  groupId: string;
}

const GroupInfoModal: React.FC<GroupInfoModalProps> = ({ open, onClose, groupId }) => {
  const { user } = useAuth();
  const [groupInfo, setGroupInfo] = useState<any>(null);
  const [members, setMembers] = useState<string[]>([]);
  const [editingNotice, setEditingNotice] = useState(false);
  const [editingName, setEditingName] = useState(false);
  const [newNotice, setNewNotice] = useState("");
  const [newName, setNewName] = useState("");
  const [isOwner, setIsOwner] = useState(false);

  useEffect(() => {
    if (open && groupId) {
      loadGroupInfo();
    }
  }, [open, groupId]);

  const loadGroupInfo = async () => {
    try {
      const res = await api.getGroupInfo(groupId);
      const data = res.data?.data || res.data;
      setGroupInfo(data);
      setMembers(data?.members || []);
      setIsOwner(data?.ownerId === user?.uuid);
      setNewNotice(data?.notice || "");
      setNewName(data?.name || "群聊");
    } catch (e) {
      console.error("获取群信息失败", e);
    }
  };

  if (!open) return null;

  return (
    <div
    className="fixed inset-0 bg-black/40 flex items-center justify-center z-50"
    onClick={onClose}  // ✅ 点击遮罩关闭
  >
    <div
      className="bg-white w-[380px] rounded-xl shadow-xl p-6 relative"
      onClick={(e) => e.stopPropagation()}  // ✅ 阻止点击内容关闭
    >
      <button onClick={onClose} className="absolute top-3 right-3 text-xl">×</button>
      <h2 className="text-lg font-bold mb-4">群资料</h2>

        {/* 群名称 */}
        <div className="mb-3">
          <label className="text-sm text-gray-500">群名称</label>
          {!editingName ? (
            <div className="flex justify-between items-center">
              <span>{groupInfo?.name}</span>
              {isOwner && (
                <button onClick={() => setEditingName(true)} className="text-blue-600 text-sm">修改</button>
              )}
            </div>
          ) : (
            <div className="flex space-x-2">
              <input
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                className="border p-1 flex-1"
              />
              <button
                onClick={async () => {
                  await api.updateGroupName({ groupId: groupId, name: newName });
                  setEditingName(false);
                  loadGroupInfo();
                }}
                className="bg-blue-600 text-white px-2 rounded"
              >保存</button>
            </div>
          )}
        </div>

        {/* 群公告 */}
        <div className="mb-3">
          <label className="text-sm text-gray-500">群公告</label>
          {!editingNotice ? (
            <div className="flex justify-between items-center">
              <span className="text-gray-700">{groupInfo?.notice || "暂无公告"}</span>
              {isOwner && (
                <button onClick={() => setEditingNotice(true)} className="text-blue-600 text-sm">编辑</button>
              )}
            </div>
          ) : (
            <div>
              <textarea
                value={newNotice}
                onChange={(e) => setNewNotice(e.target.value)}
                className="border w-full p-2 h-20"
              />
              <div className="flex justify-end space-x-2">
                <button onClick={() => setEditingNotice(false)}>取消</button>
                <button
                  onClick={async () => {
                    await api.updateGroupNotice({ groupId: groupId, notice: newNotice });
                    setEditingNotice(false);
                    loadGroupInfo();
                  }}
                  className="bg-green-600 text-white px-2 rounded"
                >保存</button>
              </div>
            </div>
          )}
        </div>

        {/* 群ID */}
        <div className="mb-3">
          <label className="text-sm text-gray-500" >群ID</label>
          <div className="flex justify-between items-center">
            <span className="text-black">{groupInfo?.uuid}</span>
            <button
              onClick={() => navigator.clipboard.writeText(groupInfo?.uuid)}
              className="text-blue-600 text-sm"
            >
              复制
            </button>
          </div>
        </div>

        {/* 成员数量 */}
        <div className="mb-3">
          <label className="text-sm text-gray-500">成员数量</label>
          <div>{members.length} 人</div>
        </div>

        {/* 退出/解散按钮 */}
        <div className="mt-4 flex justify-end space-x-3">
          {!isOwner ? (
            <button className="bg-red-500 text-white px-3 py-1 rounded"
              onClick={async () => {
                await api.leaveGroup({ groupUuid: groupId });
                alert("已退出群聊");
                onClose();
              }}>
              退出群聊
            </button>
          ) : (
            <button className="bg-red-600 text-white px-3 py-1 rounded"
              onClick={async () => {
                var ownerId = groupInfo.ownerId
                await api.dismissGroup({ groupId });
                alert("群已解散");
                onClose();
              }}>
              解散群聊
            </button>
          )}
        </div>
      </div>
    </div>
  );
};

export default GroupInfoModal;
