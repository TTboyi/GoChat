import React, { useEffect, useState, useRef } from "react";
import api from "../../api/api";
import { useAuth } from "../../context/AuthContext";
import { API_BASE } from "../../config";

interface GroupInfoModalProps {
  open: boolean;
  onClose: () => void;
  groupId: string;
  onAvatarUpdated?: () => void;
}

const GroupInfoModal: React.FC<GroupInfoModalProps> = ({
  open,
  onClose,
  groupId,
  onAvatarUpdated,
}) => {
  const { user } = useAuth();
  const [groupInfo, setGroupInfo] = useState<any>(null);
  const [members, setMembers] = useState<string[]>([]);
  const [editingNotice, setEditingNotice] = useState(false);
  const [editingName, setEditingName] = useState(false);
  const [newNotice, setNewNotice] = useState("");
  const [newName, setNewName] = useState("");
  const [isOwner, setIsOwner] = useState(false);
  const [uploading, setUploading] = useState(false);
  const avatarInputRef = useRef<HTMLInputElement>(null);

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

  const handleAvatarUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setUploading(true);
    try {
      const formData = new FormData();
      formData.append("file", file);
      const { relativeUrl } = await api.uploadImage(formData);
      if (!relativeUrl) throw new Error("上传失败");
      await api.updateGroupAvatar({ groupUuid: groupId, avatar: relativeUrl });
      await loadGroupInfo();
      onAvatarUpdated?.();
      alert("群头像已更新");
    } catch (e: any) {
      alert(e.response?.data?.error || "上传失败");
    } finally {
      setUploading(false);
      e.target.value = "";
    }
  };

  const avatarSrc = (url?: string) => {
    if (!url) return null;
    if (url.startsWith("http")) return url;
    return `${API_BASE}${url}`;
  };

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 bg-black/40 flex items-center justify-center z-50"
      onClick={onClose}
    >
      <div
        className="bg-white w-[380px] rounded-xl shadow-xl p-6 relative"
        onClick={(e) => e.stopPropagation()}
      >
        <button onClick={onClose} className="absolute top-3 right-3 text-xl text-gray-500 hover:text-black">
          ×
        </button>
        <h2 className="text-lg font-bold mb-4">群资料</h2>

        {/* 群头像 */}
        <div className="mb-4 flex flex-col items-center">
          <div
            className={`relative cursor-pointer group ${isOwner ? "cursor-pointer" : ""}`}
            onClick={() => isOwner && avatarInputRef.current?.click()}
            title={isOwner ? "点击修改群头像" : ""}
          >
            {groupInfo?.avatar ? (
              <img
                src={avatarSrc(groupInfo.avatar)!}
                className="w-20 h-20 rounded-xl object-cover border-2 border-gray-200"
                alt="群头像"
                onError={(e) => { (e.target as HTMLImageElement).style.display = "none"; }}
              />
            ) : (
              <div className="w-20 h-20 rounded-xl bg-gray-200 flex items-center justify-center text-2xl font-bold text-gray-500">
                {groupInfo?.name?.[0] || "群"}
              </div>
            )}
            {isOwner && (
              <div className="absolute inset-0 rounded-xl bg-black/30 opacity-0 group-hover:opacity-100 flex items-center justify-center transition">
                <span className="text-white text-xs">{uploading ? "上传中..." : "修改头像"}</span>
              </div>
            )}
          </div>
          <input
            ref={avatarInputRef}
            type="file"
            accept="image/*"
            className="hidden"
            onChange={handleAvatarUpload}
          />
        </div>

        {/* 群名称 */}
        <div className="mb-3">
          <label className="text-sm text-gray-500">群名称</label>
          {!editingName ? (
            <div className="flex justify-between items-center">
              <span className="text-gray-700">{groupInfo?.name}</span>
              {isOwner && (
                <button onClick={() => setEditingName(true)} className="text-blue-600 text-sm">
                  修改
                </button>
              )}
            </div>
          ) : (
            <div className="flex space-x-2">
              <input
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                className="border p-1 flex-1 rounded text-sm"
              />
              <button
                onClick={async () => {
                  await api.updateGroupName({ groupId, name: newName });
                  setEditingName(false);
                  loadGroupInfo();
                }}
                className="bg-blue-600 text-white px-2 rounded text-sm"
              >
                保存
              </button>
              <button onClick={() => setEditingName(false)} className="text-gray-400 text-sm">取消</button>
            </div>
          )}
        </div>

        {/* 群公告 */}
        <div className="mb-3">
          <label className="text-sm text-gray-500">群公告</label>
          {!editingNotice ? (
            <div className="flex justify-between items-center">
              <span className="text-gray-600 text-sm">
                {groupInfo?.notice || "暂无公告"}
              </span>
              {isOwner && (
                <button onClick={() => setEditingNotice(true)} className="text-blue-600 text-sm">
                  编辑
                </button>
              )}
            </div>
          ) : (
            <div>
              <textarea
                value={newNotice}
                onChange={(e) => setNewNotice(e.target.value)}
                className="border w-full p-2 h-20 rounded text-sm"
              />
              <div className="flex justify-end space-x-2 mt-1">
                <button onClick={() => setEditingNotice(false)} className="text-gray-400 text-sm">取消</button>
                <button
                  onClick={async () => {
                    await api.updateGroupNotice({ groupId, notice: newNotice });
                    setEditingNotice(false);
                    loadGroupInfo();
                  }}
                  className="bg-green-600 text-white px-2 rounded text-sm"
                >
                  保存
                </button>
              </div>
            </div>
          )}
        </div>

        {/* 群ID */}
        <div className="mb-3">
          <label className="text-sm text-gray-500">群ID</label>
          <div className="flex justify-between items-center">
            <span className="text-gray-600 text-sm font-mono">{groupInfo?.uuid}</span>
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
          <div className="text-gray-600 text-sm">{members.length} 人</div>
        </div>

        {/* 退出/解散按钮 */}
        <div className="mt-4 flex justify-end space-x-3">
          {!isOwner ? (
            <button
              className="bg-red-500 text-white px-3 py-1 rounded text-sm"
              onClick={async () => {
                if (!confirm("确定退出群聊？")) return;
                await api.leaveGroup({ groupUuid: groupId });
                onClose();
              }}
            >
              退出群聊
            </button>
          ) : (
            <button
              className="bg-red-600 text-white px-3 py-1 rounded text-sm"
              onClick={async () => {
                if (!confirm("确定解散群聊？")) return;
                await api.dismissGroup({ groupId });
                onClose();
              }}
            >
              解散群聊
            </button>
          )}
        </div>
      </div>
    </div>
  );
};

export default GroupInfoModal;
