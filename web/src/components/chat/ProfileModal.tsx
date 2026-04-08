import React, { useState, useEffect } from "react";
import api from "../../api/api";
import { toAbs } from "../../utils/chatUtils";

interface ProfileModalProps {
  open: boolean;
  onClose: () => void;
  user: { nickname?: string; email?: string; avatar?: string } | null;
  avatarVersion: number;
  onRefreshUser: () => Promise<void>;
  onAvatarUpdated: () => void;
}

const ProfileModal: React.FC<ProfileModalProps> = ({
  open,
  onClose,
  user,
  avatarVersion,
  onRefreshUser,
  onAvatarUpdated,
}) => {
  const [form, setForm] = useState({ nickname: "", email: "" });

  useEffect(() => {
    if (open && user) {
      setForm({ nickname: user.nickname ?? "", email: user.email ?? "" });
    }
  }, [open, user]);

  if (!open) return null;

  const handleAvatarChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const fd = new FormData();
    fd.append("file", file);
    try {
      const res: any = await api.uploadAvatar(fd);
      if (res?.avatarUrl) {
        await onRefreshUser();
        onAvatarUpdated();
        alert("头像更新成功！");
      } else {
        alert("上传失败：服务端未返回 url");
      }
    } catch {
      alert("头像上传失败");
    }
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const res = await api.updateUser({
        nickname: form.nickname.trim(),
        email: form.email.trim(),
      });
      if (res.status === 200) {
        await onRefreshUser();
        alert("更新成功！");
        onClose();
      } else {
        alert("更新失败：" + (res.data?.error || res.statusText));
      }
    } catch {
      alert("更新资料失败");
    }
  };

  return (
    <div
      className="fixed inset-0 flex items-center justify-center z-50 bg-black/40 animate-fadeIn"
      onClick={onClose}
    >
      <div
        className="bg-white/90 backdrop-blur-md rounded-2xl shadow-2xl w-[420px] p-6 relative
           transform transition-all duration-300 ease-out animate-scaleIn"
        onClick={(e) => e.stopPropagation()}
      >
        <button
          onClick={onClose}
          className="absolute top-3 right-3 text-gray-500 hover:text-gray-700 text-xl"
        >
          ×
        </button>
        <h2 className="text-xl font-semibold mb-4 text-gray-800">个人资料</h2>

        <div className="flex flex-col items-center mb-5">
          <label htmlFor="avatarUpload" className="cursor-pointer group relative">
            {user?.avatar ? (
              <img
                src={`${toAbs(user.avatar)}?v=${avatarVersion}`}
                alt="avatar"
                className="w-24 h-24 rounded-full object-cover border border-gray-300 group-hover:brightness-75 transition"
              />
            ) : (
              <div className="w-24 h-24 rounded-full bg-gray-300 flex items-center justify-center text-gray-700 text-3xl">
                {user?.nickname?.[0] || "我"}
              </div>
            )}
            <div className="absolute bottom-0 w-full text-center text-xs bg-black/50 text-white py-1 opacity-0 group-hover:opacity-100 transition">
              更换头像
            </div>
          </label>
          <input
            id="avatarUpload"
            type="file"
            accept="image/*"
            className="hidden"
            onChange={handleAvatarChange}
          />
        </div>

        <form onSubmit={handleSave} className="space-y-4">
          <div>
            <label className="text-sm text-gray-600">昵称</label>
            <input
              value={form.nickname}
              onChange={(e) => setForm((v) => ({ ...v, nickname: e.target.value }))}
              className="w-full border border-gray-300 rounded-lg px-3 py-2 mt-1 bg-white/60
               focus:outline-none focus:ring-2 focus:ring-blue-400 text-gray-900"
            />
          </div>
          <div>
            <label className="text-sm text-gray-600">邮箱</label>
            <input
              value={form.email}
              onChange={(e) => setForm((v) => ({ ...v, email: e.target.value }))}
              className="w-full border border-gray-300 rounded-lg px-3 py-2 mt-1 bg-white/60
               focus:outline-none focus:ring-2 focus:ring-blue-400 text-gray-900"
            />
          </div>
          <div className="flex justify-end space-x-3 mt-4">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 rounded bg-gray-300 hover:bg-gray-400 text-sm"
            >
              取消
            </button>
            <button
              type="submit"
              className="px-4 py-2 rounded bg-blue-500 hover:bg-blue-600 text-white text-sm"
            >
              保存
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default ProfileModal;
