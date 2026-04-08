import React, { useState, useRef } from "react";
import api from "../../api/api";

interface CreateGroupModalProps {
  open: boolean;
  onClose: () => void;
  userId: string | undefined;
  onCreated: (groupUUID: string) => void;
}

const CreateGroupModal: React.FC<CreateGroupModalProps> = ({
  open,
  onClose,
  userId,
  onCreated,
}) => {
  const [form, setForm] = useState({ name: "", notice: "", avatar: "" });
  const [previewUrl, setPreviewUrl] = useState<string>("");
  const [uploading, setUploading] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  if (!open) return null;

  const handleAvatarSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    // 本地预览
    setPreviewUrl(URL.createObjectURL(file));
    setUploading(true);
    try {
      const formData = new FormData();
      formData.append("file", file);
      const { relativeUrl } = await api.uploadImage(formData);
      setForm((v) => ({ ...v, avatar: relativeUrl || "" }));
    } catch {
      alert("头像上传失败");
      setPreviewUrl("");
    } finally {
      setUploading(false);
      e.target.value = "";
    }
  };

  const handleCreate = async () => {
    if (!form.name.trim()) return alert("请输入群聊名称");
    try {
      const res = await api.createGroup({
        name: form.name,
        notice: form.notice,
        avatar: form.avatar,
        ownerId: userId || "",
      });
      const groupUUID = res.data?.group_uuid;
      setForm({ name: "", notice: "", avatar: "" });
      setPreviewUrl("");
      onClose();
      if (groupUUID) onCreated(groupUUID);
    } catch (e: any) {
      alert(e.response?.data?.error || "创建失败");
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-[#2e2e2e] w-[360px] rounded-xl shadow-lg p-6 space-y-4">
        <h2 className="text-lg font-semibold text-gray-200">创建群聊</h2>

        {/* 群头像 */}
        <div className="flex flex-col items-center">
          <div
            className="w-20 h-20 rounded-xl bg-[#3a3b3d] flex items-center justify-center cursor-pointer overflow-hidden border-2 border-dashed border-gray-500 hover:border-gray-400 transition"
            onClick={() => fileInputRef.current?.click()}
            title="点击上传群头像"
          >
            {previewUrl ? (
              <img src={previewUrl} className="w-full h-full object-cover" alt="预览" />
            ) : (
              <div className="text-center text-gray-400 text-xs">
                <div className="text-2xl mb-1">📷</div>
                <div>{uploading ? "上传中..." : "群头像"}</div>
              </div>
            )}
          </div>
          <input
            ref={fileInputRef}
            type="file"
            accept="image/*"
            className="hidden"
            onChange={handleAvatarSelect}
          />
          <p className="text-xs text-gray-500 mt-1">点击设置群头像（可选）</p>
        </div>

        <input
          placeholder="群聊名称 *"
          className="w-full bg-[#3a3b3d] rounded px-3 py-2 text-gray-200 outline-none"
          value={form.name}
          onChange={(e) => setForm((v) => ({ ...v, name: e.target.value }))}
        />
        <input
          placeholder="群公告（可选）"
          className="w-full bg-[#3a3b3d] rounded px-3 py-2 text-gray-200 outline-none"
          value={form.notice}
          onChange={(e) => setForm((v) => ({ ...v, notice: e.target.value }))}
        />
        <div className="flex justify-end space-x-2 pt-2">
          <button
            onClick={onClose}
            className="px-3 py-2 rounded bg-gray-500 hover:bg-gray-600 text-sm text-white"
          >
            取消
          </button>
          <button
            onClick={handleCreate}
            disabled={uploading}
            className="px-3 py-2 rounded bg-green-600 hover:bg-green-700 text-sm text-white disabled:opacity-60"
          >
            创建
          </button>
        </div>
      </div>
    </div>
  );
};

export default CreateGroupModal;
