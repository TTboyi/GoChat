// ============================================================
// 文件：web/src/components/chat/AddFriendModal.tsx
// 作用：搜索并添加好友的弹窗（通过邮箱或用户 UUID 搜索，然后发起好友申请）。
// ============================================================
import React, { useState } from "react";
import api from "../../api/api";

interface AddFriendModalProps {
  open: boolean;
  onClose: () => void;
}

const AddFriendModal: React.FC<AddFriendModalProps> = ({ open, onClose }) => {
  const [inputValue, setInputValue] = useState("");

  if (!open) return null;

  const handleConfirm = async () => {
    if (!inputValue.trim()) {
      alert("请输入好友邮箱或ID");
      return;
    }
    try {
      const res = await api.applyContact({
        target: inputValue.trim(),
        message: "你好，希望能添加你为好友",
      });
      alert(res.data?.message || "申请成功");
      setInputValue("");
      onClose();
    } catch (err: any) {
      alert("申请失败：" + (err.response?.data?.error || err.message));
    }
  };

  return (
    <div
      className="fixed inset-0 bg-black/40 flex items-center justify-center z-50 animate-fadeIn"
      onClick={onClose}
    >
      <div
        className="bg-white/90 backdrop-blur-md rounded-2xl shadow-2xl w-[380px] p-6 relative animate-scaleIn"
        onClick={(e) => e.stopPropagation()}
      >
        <button
          onClick={onClose}
          className="absolute top-3 right-3 text-gray-500 hover:text-gray-700 text-xl"
        >
          ×
        </button>
        <h2 className="text-lg font-semibold mb-4 text-gray-800">添加好友</h2>
        <p className="text-sm text-gray-600 mb-3">输入对方邮箱或用户ID：</p>
        <input
          value={inputValue}
          onChange={(e) => setInputValue(e.target.value)}
          placeholder="例如：12345678 或 test@example.com"
          className="w-full border border-gray-300 rounded-lg px-3 py-2 mt-1 bg-white/60
             focus:outline-none focus:ring-2 focus:ring-blue-400 text-gray-900
             placeholder-gray-400"
        />
        <div className="flex justify-end space-x-3 mt-4">
          <button
            onClick={onClose}
            className="px-4 py-2 rounded bg-gray-300 hover:bg-gray-400 text-sm"
          >
            取消
          </button>
          <button
            onClick={handleConfirm}
            className="px-4 py-2 rounded bg-blue-500 hover:bg-blue-600 text-white text-sm"
          >
            确认添加
          </button>
        </div>
      </div>
    </div>
  );
};

export default AddFriendModal;
