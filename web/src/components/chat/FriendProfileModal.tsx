// 好友个人信息弹窗 + 修改备注（拟态玻璃卡片风格）
import React, { useState } from "react";
import { toAbs } from "../../utils/chatUtils";

interface FriendInfo {
  uuid: string;
  nickname: string;
  email?: string;
  avatar?: string;
  remark?: string;
}

interface FriendProfileModalProps {
  open: boolean;
  friend: FriendInfo | null;
  avatarVersion: number;
  onClose: () => void;
  onDeleteFriend: (friendId: string) => void;
  onRemarkSaved: (friendId: string, remark: string) => void;
}

const FriendProfileModal: React.FC<FriendProfileModalProps> = ({
  open,
  friend,
  avatarVersion,
  onClose,
  onDeleteFriend,
  onRemarkSaved,
}) => {
  const [remarkMode, setRemarkMode] = useState(false);
  const [remarkInput, setRemarkInput] = useState("");
  const [confirmDelete, setConfirmDelete] = useState(false);

  if (!open || !friend) return null;

  const handleOpenRemark = () => {
    setRemarkInput(friend.remark || "");
    setRemarkMode(true);
  };

  const handleSaveRemark = () => {
    onRemarkSaved(friend.uuid, remarkInput.trim());
    setRemarkMode(false);
  };

  const handleDelete = () => {
    onDeleteFriend(friend.uuid);
    setConfirmDelete(false);
    onClose();
  };

  return (
    <div
      className="fixed inset-0 flex items-center justify-center z-[200] bg-black/50 animate-fadeIn"
      onClick={onClose}
    >
      <div
        className="relative bg-[#1e1e2e]/95 backdrop-blur-xl border border-white/10 rounded-2xl shadow-2xl w-[360px] max-w-[95vw] overflow-hidden animate-scaleIn"
        onClick={(e) => e.stopPropagation()}
      >
        {/* 顶部渐变装饰 */}
        <div className="h-24 bg-gradient-to-br from-indigo-600/40 to-purple-700/30 flex items-end px-5 pb-0">
          <div className="relative mb-[-28px]">
            {friend.avatar ? (
              <img
                src={`${toAbs(friend.avatar)}?v=${avatarVersion}`}
                alt={friend.nickname}
                className="w-14 h-14 rounded-xl object-cover border-2 border-white/20 shadow-lg"
              />
            ) : (
              <div className="w-14 h-14 rounded-xl bg-indigo-500/60 flex items-center justify-center text-xl font-bold text-white border-2 border-white/20">
                {(friend.nickname || "?")[0].toUpperCase()}
              </div>
            )}
          </div>
        </div>

        <div className="pt-10 px-5 pb-5">
          {/* 名称 + 备注 */}
          <div className="mb-4">
            <div className="text-lg font-semibold text-white">
              {friend.remark ? (
                <>
                  <span>{friend.remark}</span>
                  <span className="text-sm text-gray-400 ml-2 font-normal">({friend.nickname})</span>
                </>
              ) : (
                friend.nickname
              )}
            </div>
            {friend.email && (
              <div className="text-xs text-gray-500 mt-0.5">{friend.email}</div>
            )}
          </div>

          {/* 信息列表 */}
          <div className="space-y-2 mb-5">
            <InfoRow label="用户 ID" value={friend.uuid} mono />
            <InfoRow label="昵称" value={friend.nickname} />
            {friend.remark && <InfoRow label="我的备注" value={friend.remark} />}
            {friend.email && <InfoRow label="邮箱" value={friend.email} />}
          </div>

          {/* 操作按钮 */}
          <div className="flex gap-2">
            <button
              onClick={handleOpenRemark}
              className="flex-1 py-2 rounded-xl text-sm font-medium bg-white/10 hover:bg-white/20 text-white transition"
            >
              {friend.remark ? "修改备注" : "添加备注"}
            </button>
            <button
              onClick={() => setConfirmDelete(true)}
              className="flex-1 py-2 rounded-xl text-sm font-medium bg-red-500/20 hover:bg-red-500/30 text-red-400 transition"
            >
              删除好友
            </button>
          </div>
        </div>

        {/* 关闭按钮 */}
        <button
          onClick={onClose}
          className="absolute top-3 right-3 w-7 h-7 flex items-center justify-center rounded-full bg-white/10 hover:bg-white/20 text-gray-300 text-sm transition"
        >
          ✕
        </button>

        {/* 修改备注子弹窗 */}
        {remarkMode && (
          <div
            className="absolute inset-0 flex items-center justify-center bg-black/60 rounded-2xl"
            onClick={() => setRemarkMode(false)}
          >
            <div
              className="bg-[#2a2a3e] border border-white/10 rounded-xl p-5 w-64 shadow-xl"
              onClick={(e) => e.stopPropagation()}
            >
              <h3 className="text-base font-semibold text-white mb-3">修改备注</h3>
              <input
                autoFocus
                value={remarkInput}
                onChange={(e) => setRemarkInput(e.target.value)}
                onKeyDown={(e) => { if (e.key === "Enter") handleSaveRemark(); }}
                placeholder="输入备注名称..."
                className="w-full bg-white/10 border border-white/20 rounded-lg px-3 py-2 text-white text-sm placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500"
              />
              <div className="flex gap-2 mt-3">
                <button
                  onClick={() => setRemarkMode(false)}
                  className="flex-1 py-2 rounded-lg text-sm bg-white/10 text-gray-300 hover:bg-white/20 transition"
                >
                  取消
                </button>
                <button
                  onClick={handleSaveRemark}
                  className="flex-1 py-2 rounded-lg text-sm bg-indigo-600 text-white hover:bg-indigo-700 transition font-medium"
                >
                  保存
                </button>
              </div>
            </div>
          </div>
        )}

        {/* 删除确认子弹窗 */}
        {confirmDelete && (
          <div
            className="absolute inset-0 flex items-center justify-center bg-black/60 rounded-2xl"
            onClick={() => setConfirmDelete(false)}
          >
            <div
              className="bg-[#2a2a3e] border border-white/10 rounded-xl p-5 w-64 shadow-xl"
              onClick={(e) => e.stopPropagation()}
            >
              <h3 className="text-base font-semibold text-white mb-2">确认删除</h3>
              <p className="text-sm text-gray-400 mb-4">
                删除후双方聊天记录将永久清除，无法恢复，确定要删除 <span className="text-white font-medium">{friend.remark || friend.nickname}</span> 吗？
              </p>
              <div className="flex gap-2">
                <button
                  onClick={() => setConfirmDelete(false)}
                  className="flex-1 py-2 rounded-lg text-sm bg-white/10 text-gray-300 hover:bg-white/20 transition"
                >
                  取消
                </button>
                <button
                  onClick={handleDelete}
                  className="flex-1 py-2 rounded-lg text-sm bg-red-600 text-white hover:bg-red-700 transition font-medium"
                >
                  确认删除
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

const InfoRow: React.FC<{ label: string; value: string; mono?: boolean }> = ({ label, value, mono }) => (
  <div className="flex items-center bg-white/5 rounded-lg px-3 py-2 gap-3">
    <span className="text-xs text-gray-500 w-14 flex-shrink-0">{label}</span>
    <span className={`text-sm text-gray-200 truncate select-all ${mono ? "font-mono text-xs" : ""}`}>
      {value}
    </span>
  </div>
);

export default FriendProfileModal;
