// src/components/NewFriendModal.tsx
import React, { useEffect, useState } from "react";
import api from "../api/api";

interface Props {
  open: boolean;
  onClose: () => void;
  onRefreshContacts?: () => void; // ✅ 新增
}

type ApplyView = {
  uuid: string;
  userId: string;
  nickname?: string;
  avatar?: string;
  message: string;
  lastApplyAt?: string;
};

interface Props {
  open: boolean;
  onClose: () => void;
}

// 统一把相对路径转成绝对路径，并可选加上防缓存参数
const toAbs = (rel?: string, bust = false) => {
  if (!rel) return "";
  const base = "http://localhost:8000";
  const url = rel.startsWith("http") ? rel : `${base}${rel}`;
  return bust ? `${url}${url.includes("?") ? "&" : "?"}v=${Date.now()}` : url;
};

// 兼容化：把后端“大小写两套字段”合并为前端一套字段
const normalize = (raw: any): ApplyView => ({
  uuid: raw?.uuid ?? raw?.Uuid ?? "",
  userId: raw?.userId ?? raw?.UserId ?? "",
  nickname: raw?.nickname ?? raw?.Nickname ?? undefined,
  avatar: raw?.avatar ?? raw?.Avatar ?? undefined,
  message: raw?.message ?? raw?.Message ?? "",
  lastApplyAt: raw?.lastApplyAt ?? raw?.LastApplyAt ?? undefined,
});

const NewFriendModal: React.FC<Props> = ({ open, onClose ,onRefreshContacts}) => {
  const [loading, setLoading] = useState(true);
  const [list, setList] = useState<ApplyView[]>([]);

  const loadApplies = async () => {
    setLoading(true);
    try {
      const res = await api.getNewContactList();
      const arr = (res?.data?.data ?? res?.data ?? []) as any[];
      const items = arr.map(normalize);
      setList(items);
    } catch (err) {
      console.error("获取好友申请失败:", err);
      alert("获取好友申请失败");
    } finally {
      setLoading(false);
    }
  };

  const handleApply = async (applyUuid: string, approve: boolean) => {
    try {
      // 后端结构：{ apply_uuid, approve }
      await api.handleContactApply({ applyUuid: applyUuid, approve });
      //setList((prev) => prev.filter((x) => x.uuid !== applyUuid));
      alert(approve ? "已同意好友申请" : "已拒绝好友申请");
      if (approve && onRefreshContacts) {
        onRefreshContacts();
      }
      
    } catch (err) {
      console.error("处理失败:", err);
      alert("操作失败");
    }
  };

  useEffect(() => {
    if (open) loadApplies();
  }, [open]);

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
      onClick={onClose}
    >
      <div
        className="bg-white rounded-2xl shadow-2xl w-[520px] max-h-[70vh] p-5 relative"
        onClick={(e) => e.stopPropagation()}
      >
        <button
          onClick={onClose}
          className="absolute top-3 right-3 w-8 h-8 rounded-lg bg-black/80 text-white text-lg leading-none"
        >
          ×
        </button>

        <h2 className="text-lg font-semibold mb-4 text-black ">新的朋友</h2>

        {loading ? (
          <div className="text-gray-500 text-center py-6">加载中...</div>
        ) : list.length === 0 ? (
          <div className="text-gray-400 text-center py-6">暂无新的申请</div>
        ) : (
          <div className="space-y-3 overflow-y-auto pr-1">
            {list.map((item) => {
              const name = item.nickname || item.userId || "好友";
              const avatarSrc = item.avatar ? toAbs(item.avatar, true) : ""; // bust cache
              return (
                <div
                  key={item.uuid}
                  className="flex items-center justify-between border rounded-xl px-4 py-3"
                >
                  <div className="flex items-center min-w-0">
                    {avatarSrc ? (
                      <img
                        src={avatarSrc}
                        alt={name}
                        className="w-10 h-10 rounded-lg object-cover flex-shrink-0"
                        onError={(e) => {
                          // 加个兜底，防止 404
                          (e.currentTarget as HTMLImageElement).src =
                            "https://via.placeholder.com/40?text=%F0%9F%91%A5";
                        }}
                      />
                    ) : (
                      <div className="w-10 h-10 rounded-lg bg-gray-200 flex items-center justify-center text-gray-600 text-sm">
                        {name.slice(0, 1)}
                      </div>
                    )}
                    <div className="ml-3 min-w-0">
                      <div className="text-sm font-medium text-black truncate">{name}</div>
                      <div className="text-xs text-gray-500 mt-1 truncate" title={item.message}>
                        {item.message}
                      </div>
                    </div>
                  </div>

                  <div className="flex space-x-2 flex-shrink-0">
                    <button
                      onClick={() => handleApply(item.uuid, true)}
                      className="px-3 py-1 text-xs bg-green-500 text-white rounded hover:bg-green-600"
                    >
                      同意
                    </button>
                    <button
                      onClick={() => handleApply(item.uuid, false)}
                      className="px-3 py-1 text-xs bg-gray-300 rounded hover:bg-gray-400"
                    >
                      拒绝
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
};

export default NewFriendModal;
