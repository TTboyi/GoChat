// Chat.tsx
import React, { useEffect, useMemo, useRef, useState } from "react";
import { useAuth } from "../context/AuthContext";
import { ChatWebSocket, sendTextMessage } from "../api/socket";
import type { IncomingMessage } from "../api/socket";
import api from "../api/api";
import NewFriendModal from "../components/NewFriendModal";
import { getToken } from "../utils/session";

// ====== 类型 ======
interface SessionItem {
  id: string;
  name: string;
  avatar?: string;
  type: "user" | "group";
}

interface Message {
  uuid?: string;
  sendId: string;
  receiveId: string;
  content: string;
  type: number;
  createdAt?: number | string;
}

// 群聊类型定义
interface GroupInfo {
  uuid: string;
  name: string;
  notice?: string;
  add_mode?: number;
  owner_id?: string;
  member_cnt?: number;
  avatar?: string;
}

// ====== 工具 ======
const cn = (...a: Array<string | false | undefined>) => a.filter(Boolean).join(" ");
const toAbs = (rel?: string) => (rel ? `http://localhost:8000${rel}` : "");

// ====== 持久化工具函数 ======
const saveMessagesToStorage = (map: Record<string, Message[]>) => {
  localStorage.setItem("chat_messages", JSON.stringify(map));
};

const loadMessagesFromStorage = (): Record<string, Message[]> => {
  try {
    const raw = localStorage.getItem("chat_messages");
    return raw ? JSON.parse(raw) : {};
  } catch {
    return {};
  }
};



// ====== 主组件 ======
const Chat: React.FC = () => {
  const [showProfile, setShowProfile] = useState(false);
  const { user, refreshUser, logout } = useAuth();
  const [profileForm, setProfileForm] = useState({ nickname: "", email: "" });
  const [ws, setWs] = useState<ChatWebSocket | null>(null);
  const [sessions, setSessions] = useState<SessionItem[]>([]);
  const [activeId, setActiveId] = useState<string>("");
  const [input, setInput] = useState("");
  const listRef = useRef<HTMLDivElement>(null);
  // 群聊相关状态
// 群聊相关状态
const [showCreateGroup, setShowCreateGroup] = useState(false);
//const [showJoinGroup, setShowJoinGroup] = useState(false);
const [groupForm, setGroupForm] = useState({ name: "", notice: "", avatar: "" });
const [joinGroupId, setJoinGroupId] = useState("");
// 群成员模态框状态
const [showGroupMembers, setShowGroupMembers] = useState(false);
const [groupMembers, setGroupMembers] = useState<string[]>([]);
const [isGroupOwner, setIsGroupOwner] = useState(false);


  // ✅ 改造：按会话保存所有消息
  const [messagesMap, setMessagesMap] = useState<Record<string, Message[]>>({});

  // UI 控制
  const [showAddFriend, setShowAddFriend] = useState(false);
  const [showJoinGroup, setShowJoinGroup] = useState(false);
  const [inputValue, setInputValue] = useState("");
  const [avatarVersion, setAvatarVersion] = useState(Date.now());
  const [showNewFriend, setShowNewFriend] = useState(false);

  // 我的群聊列表
  const [myGroups, setMyGroups] = useState<GroupInfo[]>([]);  


  
// ===== 加载群成员 =====
const loadGroupMembers = async () => {
  if (!active || active.type !== "group") return;
  try {
    const res = await api.getGroupMembers(active.id);
    const members = res.data?.members || res.data?.data || [];
    setGroupMembers(members);
    // 判断是否为群主
    const myId = user?.uuid;
    const groupInfo = myGroups.find(g => g.uuid === active.id);
    setIsGroupOwner(groupInfo?.owner_id === myId);
  } catch (e) {
    console.error("加载群成员失败:", e);
    setGroupMembers([]);
  }
};

  // 上传头像
const onAvatarChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
  const file = e.target.files?.[0];
  if (!file) return;
  const fd = new FormData();
  fd.append("file", file);
  try {
    const res: any = await api.uploadAvatar(fd);
    if (res?.avatarUrl) {
      await refreshUser();
      setAvatarVersion(Date.now());
      alert("头像更新成功！");
    } else {
      alert("上传失败：服务端未返回 url");
    }
  } catch (err) {
    console.error("头像上传失败：", err);
    alert("头像上传失败");
  }
};

// 保存资料
const onSaveProfile = async (e: React.FormEvent) => {
  e.preventDefault();
  try {
    const res = await api.updateUser({
      nickname: profileForm.nickname.trim(),
      email: profileForm.email.trim(),
    });
    if (res.status === 200) {
      await refreshUser();
      alert("更新成功！");
      setShowProfile(false);
    } else {
      alert("更新失败：" + (res.data?.error || res.statusText));
    }
  } catch (err) {
    console.error("更新资料出错", err);
    alert("更新资料失败");
  }
};



  // 初始化加载缓存消息（刷新后恢复）
  useEffect(() => {
    const stored = loadMessagesFromStorage();
    setMessagesMap(stored);
  }, []);

  useEffect(() => {
    if (user?.avatar) setAvatarVersion(Date.now());
  }, [user?.avatar]);

  useEffect(() => {
    if (showProfile && user) {
      setProfileForm({
        nickname: user.nickname ?? "",
        email: user.email ?? "",
      });
    }
  }, [showProfile, user]);

  useEffect(() => {
    if (!user) refreshUser();
  }, []);

  // 自动滚动到底部
  useEffect(() => {
    listRef.current?.scrollTo({ top: listRef.current.scrollHeight, behavior: "smooth" });
  }, [messagesMap, activeId]);

  // ====== 拉取联系人 / 群聊 ======
  const loadContacts = async () => {
    try {
      const [userRes, groupRes] = await Promise.all([
        api.getContactList(),
        api.loadMyJoinedGroup(),
      ]);
      const contactList = (userRes.data?.data || userRes.data || []) as any[];
      const groupList = (groupRes.data?.data || groupRes.data || []) as any[];

      const contacts: SessionItem[] = contactList.map((it) => ({
        id: it.uuid || it.id,
        name: it.nickname || it.name || "联系人",
        avatar: it.avatar,
        type: "user",
      }));

      const groups: SessionItem[] = groupList.map((g) => ({
        id: g.uuid || g.id,
        name: g.name || "群聊",
        avatar: g.avatar,
        type: "group",
      }));

      const merged = [...contacts, ...groups];
      // ✅ 去重：相同 id 只留一个
      const unique = Array.from(new Map(merged.map(c => [c.id, c])).values());
      setSessions(unique);


      if (!activeId && merged.length > 0) setActiveId(merged[0].id);
    } catch (e) {
      console.error("加载联系人或群聊失败：", e);
    }
  };

  useEffect(() => {
    loadContacts();
  }, []);

  // ====== 加载历史消息（每次切换会话） ======
  useEffect(() => {
    if (!activeId) return;

    const loadHistory = async () => {
      try {
        const active = sessions.find((s) => s.id === activeId);
        if (!active) return;

        let res;
        if (active.type === "group") {
          res = await api.getGroupMessageList({ groupId: active.id, limit: 300 });
        } else {
          res = await api.getMessageList({ targetId: active.id, limit: 300 });
        }

        
        const raw = (res.data?.data || res.data || []) as any[];
const arr: Message[] = raw.map((m: any) => ({
  uuid:      m.uuid      ?? m.Uuid,
  sendId:    m.sendId    ?? m.SendId,
  receiveId: m.receiveId ?? m.ReceiveId,
  content:   m.content   ?? m.Content ?? "",
  type:      m.type      ?? m.Type ?? 0,
  createdAt: typeof m.createdAt === "number"
               ? m.createdAt
               : m.CreatedAt
                 ? Math.floor(Date.parse(m.CreatedAt) / 1000)
                 : Math.floor(Date.now() / 1000),
}));

setMessagesMap(prev => ({
  ...prev,
  [active.id]: arr,   // 覆盖为标准化后的数组
}));
saveMessagesToStorage({ ...messagesMap, [active.id]: arr });

        console.log("加载历史消息:", arr);

        setMessagesMap((prev) => {
          const existing = prev[active.id] || [];
          const merged = [...existing, ...arr];
          const unique = Array.from(new Map(merged.map((m) => [m.uuid, m])).values()).sort(
            (a, b) =>
              new Date(a.createdAt || 0).getTime() -
              new Date(b.createdAt || 0).getTime()
          );
          const updated = { ...prev, [active.id]: unique };
          saveMessagesToStorage(updated); // ✅ 写入缓存
          return updated;
        });
      } catch (e) {
        console.error("加载历史消息失败：", e);
      }
    };

    loadHistory();
  }, [activeId, sessions]);

  // ====== 建立 WebSocket 连接 ======
  useEffect(() => {
    const token = getToken();
    if (!token) return;

    const socket = new ChatWebSocket({
      token,
      onMessage: (msg: IncomingMessage) => {
        const newMsg: Message = {
          uuid: msg.uuid,
          sendId: msg.sendId,
          receiveId: msg.receiveId,
          content: msg.content ?? "",
          type: msg.type ?? 0,
          createdAt: msg.createdAt ?? Math.floor(Date.now() / 1000),
        };

          // ✅ 群聊消息：receiveId 以 "G" 开头
  if (msg.receiveId?.startsWith("G")) {
    setMessagesMap((prev) => {
      const existing = prev[msg.receiveId] || [];
      const merged = [...existing, newMsg];
      const unique = Array.from(new Map(merged.map(m => [m.uuid, m])).values());
      const newMap = { ...prev, [msg.receiveId]: unique };
      saveMessagesToStorage(newMap);
      return newMap;
    });











    return;
  }


        // 确定属于哪个会话
        const targetId = msg.sendId === user?.uuid ? msg.receiveId : msg.sendId;

        setMessagesMap((prev) => {
          const oldList = prev[targetId] || [];
          if (newMsg.uuid && oldList.some(m => m.uuid === newMsg.uuid)) return prev;
          const updatedList = [...oldList, newMsg];
          const newMap = { ...prev, [targetId]: updatedList };
          saveMessagesToStorage(newMap);
          return newMap;
        });
      },
      onOpen: () => console.log("✅ WebSocket 已连接"),
      onClose: () => console.log("❌ WebSocket 已关闭"),
    });

    setWs(socket);
    return () => socket.close();
  }, []);

  // ====== 发送文本 ======
  const doSend = () => {
    if (!ws || !input.trim() || !activeId) return;
    const text = input.trim();  
    sendTextMessage(ws, input.trim(), activeId);
     // ✅ 自己先插一条（乐观显示）
  const newMsg: Message = {
    uuid: `tmp_${Date.now()}`,
    sendId: user?.uuid || "",
    receiveId: activeId,
    content: text,
    type: 0,
    createdAt: Math.floor(Date.now() / 1000),
  };

  setMessagesMap(prev => {
    const oldList = prev[activeId] || [];
    const updated = { ...prev, [activeId]: [...oldList, newMsg] };
    saveMessagesToStorage(updated);
    return updated;
  });
    setInput("");
  };

  // 回车发送
  const onKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      doSend();
    }
  };

  // 当前会话消息（从 Map 获取）
  const viewMsgs = messagesMap[activeId] || [];

  // ====== 退出登录清理缓存 ======
  const handleLogout = () => {
    localStorage.removeItem("chat_messages");
    logout();
  };
  const active = sessions.find((s) => s.id === activeId);
  // ====== UI 渲染 ======
  return (
    <div className="h-screen w-screen overflow-hidden bg-[#ededed] flex">
      {/* 左：会话栏 */}
      {/* 左侧会话栏 */}
<aside className="w-[280px] bg-[#2e2e2e] text-gray-200 flex flex-col border-r border-black/20">
  {/* 顶部用户栏 */}
  <div className="h-16 px-4 flex items-center justify-between border-b border-black/20">
    <div className="flex items-center space-x-3">
      {user?.avatar ? (
        <img
          src={`${toAbs(user.avatar)}?v=${avatarVersion}`}
          alt="me"
          className="w-9 h-9 rounded-md object-cover"
        />
      ) : (
        <div className="w-9 h-9 rounded-md bg-white/20 flex items-center justify-center">
          {user?.nickname?.[0] || "我"}
        </div>
      )}
      <div className="leading-tight">
        <div className="font-semibold">{user?.nickname || "未登录用户"}</div>
        <div className="text-xs text-gray-400">在线</div>
      </div>
    </div>
  </div>

  {/* 操作按钮区 */}
  <div className="px-3 py-2 flex justify-between border-b border-black/20">
    <button
      onClick={() => setShowProfile(true)}
      className="px-2 py-1 text-xs rounded bg-white/10 hover:bg-white/20"
    >
      资料
    </button>
    <button
      onClick={handleLogout}
      className="px-2 py-1 text-xs rounded bg-red-500 hover:bg-red-600 text-white"
    >
      退出
    </button>
    <button
      onClick={() => setShowNewFriend(true)}
      className="px-2 py-1 text-xs rounded bg-yellow-500 hover:bg-yellow-600 text-white"
    >
      新朋友
    </button>
  </div>

  {/* 联系人列表 */}
  <div className="flex-1 overflow-y-auto">
    {sessions.length === 0 && (
      <div className="text-gray-400 text-sm p-4">暂无联系人</div>
    )}
    {sessions.map((s) => (
      <button
        key={s.id}
        onClick={() => setActiveId(s.id)}
        className={cn(
          "w-full flex items-center px-3 py-3 hover:bg-[#3a3b3d]",
          activeId === s.id && "bg-[#3a3b3d]"
        )}
      >
        {s.avatar ? (
          <img
            src={`${toAbs(s.avatar)}?v=${avatarVersion}`}
            alt={s.name}
            className="w-10 h-10 rounded-md object-cover"
          />
        ) : (
          <div className="w-10 h-10 rounded-md bg-white/20 flex items-center justify-center">
            {s.name[0] || "友"}
          </div>
        )}
        <div className="ml-3 min-w-0 flex-1">
          <div className="text-sm font-semibold truncate">{s.name}</div>
          <div className="text-xs text-gray-400 truncate">
            {s.type === "group" ? "群聊" : "好友"}
          </div>
        </div>
      </button>
    ))}
  </div>

  {/* 底部添加按钮 */}
  <div className="p-3 border-t border-black/20 flex space-x-2 bg-[#262626]">
    <button
      onClick={() => { setInputValue(""); setShowAddFriend(true); }}
      className="flex-1 px-3 py-2 bg-[#3a3b3d] hover:bg-[#4a4b4d] rounded text-sm text-gray-200"
    >
      添加好友
    </button>
    <button
    onClick={() => { setShowCreateGroup(true); }}
    className="flex-1 px-3 py-2 bg-[#3a3b3d] hover:bg-[#4a4b4d] rounded text-sm text-gray-200"
  >
    创建群聊
  </button>
    <button
      onClick={() => { setInputValue(""); setShowJoinGroup(true); }}
      className="flex-1 px-3 py-2 bg-[#3a3b3d] hover:bg-[#4a4b4d] rounded text-sm text-gray-200"
    >
      加入群聊
    </button>
  </div>
</aside>


      {/* 右：聊天区 */}
      <main className="flex-1 flex flex-col bg-[#f5f5f5]">
        <div className="h-16 bg-[#f0f0f0] border-b border-gray-200 px-5 flex items-center justify-between text-black">
  <div className="text-base font-semibold">
    {activeId ? sessions.find((s) => s.id === activeId)?.name : "请选择会话"}
  </div>

  {active?.type === "group" && (
    <button
      onClick={async () => {
        await loadGroupMembers();
        setShowGroupMembers(true);
      }}
      className="px-2 py-1 text-xs rounded bg-gray-300 hover:bg-gray-300 text-grey-200"
    >
      群成员
    </button>
  )}
</div>


        {/* 消息列表 */}
        <div ref={listRef} className="flex-1 overflow-y-auto px-6 py-4 bg-[#eaeaea]">
          {(!activeId || viewMsgs.length === 0) && (
            <div className="text-gray-400 text-sm text-center mt-10">暂无消息</div>
          )}

{viewMsgs.map((m, idx) => {
  const isSelf = m.sendId === user?.uuid || m.sendId === "me";

  return (
    <div key={idx} className="mb-3">
      {/* 群聊消息显示昵称 */}
      {active?.type === "group" && !isSelf && (
        <div className="text-xs text-gray-400 ml-12 mb-1">
          {  m.sendId}
        </div>
      )}

      <div className={cn("flex items-end", isSelf ? "justify-end" : "justify-start")}>
        {!isSelf && (
          <div className="mr-2">
            {active?.avatar ? (
              <img
                src={`${toAbs(active.avatar)}?v=${avatarVersion}`}
                alt="avatar"
                className="w-8 h-8 rounded-md object-cover"
              />
            ) : (
              <div className="w-8 h-8 rounded-md bg-gray-300" />
            )}
          </div>
        )}

        <div
          className={cn(
            "max-w-[60%] px-3 py-2 rounded-2xl text-sm shadow",
            isSelf
              ? "bg-[#95ec69] text-black rounded-br-none"
              : "bg-white text-gray-900 rounded-bl-none"
          )}
        >
          {m.type === 0 ? (
            <div style={{ whiteSpace: "pre-wrap", wordBreak: "break-word" }}>
              {m.content}
            </div>
          ) : (
            <a href={m.content} target="_blank" rel="noreferrer" className="underline">
              文件
            </a>
          )}
          <div className="text-[10px] text-right opacity-60 mt-1">
            {m.createdAt
              ? new Date(
                  typeof m.createdAt === "number"
                    ? m.createdAt * 1000
                    : Date.parse(m.createdAt)
                ).toLocaleTimeString()
              : ""}
          </div>
        </div>

        {isSelf && (
          <div className="ml-2 w-8 h-8 rounded-md bg-gray-300 overflow-hidden">
            {user?.avatar && (
              <img
                src={`${toAbs(user.avatar)}?v=${avatarVersion}`}
                className="w-8 h-8 object-cover"
              />
            )}
          </div>
        )}
      </div>
    </div>
  );
})}

        </div>

        {/* 输入框 */}
        <div className="border-t border-gray-200 bg-white px-5 py-3">
          <div className="rounded-lg border border-gray-300 bg-white">
            <textarea
              className="w-full resize-none outline-none p-3 h-24 text-sm text-black"
              placeholder={activeId ? "输入消息，Enter 发送 / Shift+Enter 换行" : "请选择左侧会话"}
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={onKeyDown}
              disabled={!activeId}
            />
          </div>
          <div className="mt-2 flex justify-end">
            <button
              onClick={doSend}
              disabled={!activeId || !input.trim()}
              className={cn(
                "px-4 py-2 rounded bg-[#07c160] text-sm text-white",
                (!activeId || !input.trim()) && "opacity-60 cursor-not-allowed"
              )}
            >
              发送
            </button>
          </div>
        </div>
        {showAddFriend && (
  <div
    className="fixed inset-0 bg-black/40 flex items-center justify-center z-50 animate-fadeIn"
    onClick={() => setShowAddFriend(false)}
  >
    <div
      className="bg-white/90 backdrop-blur-md rounded-2xl shadow-2xl w-[380px] p-6 relative animate-scaleIn"
      onClick={(e) => e.stopPropagation()}
    >
      <button
        onClick={() => setShowAddFriend(false)}
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
          onClick={() => setShowAddFriend(false)}
          className="px-4 py-2 rounded bg-gray-300 hover:bg-gray-400 text-sm"
        >
          取消
        </button>
        <button
          onClick={async () => {
            if (!inputValue.trim()) {
              alert("请输入好友邮箱或ID");
              return;
            }
            try {
              const defaultMessage = "你好，希望能添加你为好友";
              const res = await api.applyContact({
                target: inputValue.trim(),
                message: defaultMessage,
              });
              alert(res.data?.message || "申请成功");
              setShowAddFriend(false);
            } catch (err: any) {
              alert("申请失败：" + (err.response?.data?.error || err.message));
            }
          }}
          className="px-4 py-2 rounded bg-blue-500 hover:bg-blue-600 text-white text-sm"
        >
          确认添加
        </button>
      </div>
    </div>
  </div>
)}

{/* ============ 个人资料 模态框 ============ */}
{showProfile && (
  <div
    className="fixed inset-0 flex items-center justify-center z-50 bg-black/40 animate-fadeIn"
    onClick={() => setShowProfile(false)}
  >
    <div
      className="bg-white/90 backdrop-blur-md rounded-2xl shadow-2xl w-[420px] p-6 relative
                 transform transition-all duration-300 ease-out animate-scaleIn"
      onClick={(e) => e.stopPropagation()} // 防止点击内部关闭
    >
      {/* 关闭按钮 */}
      <button
        onClick={() => setShowProfile(false)}
        className="absolute top-3 right-3 text-gray-500 hover:text-gray-700 text-xl"
      >
        ×
      </button>

      <h2 className="text-xl font-semibold mb-4 text-gray-800">个人资料</h2>

      {/* 头像上传区域 */}
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
          onChange={onAvatarChange} // ✅ 直接调用已有函数
        />
      </div>

      {/* 修改资料表单 */}
      <form onSubmit={onSaveProfile} className="space-y-4">
        <div>
          <label className="text-sm text-gray-600">昵称</label>
          <input
            name="nickname"
            value={profileForm.nickname}
            onChange={(e) =>
              setProfileForm((v) => ({ ...v, nickname: e.target.value }))
            }
            className="w-full border border-gray-300 rounded-lg px-3 py-2 mt-1 bg-white/60
                     focus:outline-none focus:ring-2 focus:ring-blue-400 text-gray-900
                     placeholder-gray-400"
          />
        </div>

        <div>
          <label className="text-sm text-gray-600">邮箱</label>
          <input
            name="email"
            value={profileForm.email}
            onChange={(e) =>
              setProfileForm((v) => ({ ...v, email: e.target.value }))
            }
            className="w-full border border-gray-300 rounded-lg px-3 py-2 mt-1 bg-white/60
                     focus:outline-none focus:ring-2 focus:ring-blue-400 text-gray-900
                     placeholder-gray-400"
          />
        </div>

        <div className="flex justify-end space-x-3 mt-4">
          <button
            type="button"
            onClick={() => setShowProfile(false)}
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
)}

{showCreateGroup && (
  <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
    <div className="bg-[#2e2e2e] w-[360px] rounded-xl shadow-lg p-6 space-y-4">
      <h2 className="text-lg font-semibold text-gray-200">创建群聊</h2>

      <input
        placeholder="群聊名称"
        className="w-full bg-[#3a3b3d] rounded px-3 py-2 text-gray-200 outline-none"
        value={groupForm.name}
        onChange={(e) => setGroupForm(v => ({ ...v, name: e.target.value }))}
      />

      <input
        placeholder="群公告（可选）"
        className="w-full bg-[#3a3b3d] rounded px-3 py-2 text-gray-200 outline-none"
        value={groupForm.notice}
        onChange={(e) => setGroupForm(v => ({ ...v, notice: e.target.value }))}
      />

      <div className="flex justify-end space-x-2 pt-2">
        <button
          onClick={() => setShowCreateGroup(false)}
          className="px-3 py-2 rounded bg-gray-500 hover:bg-gray-600 text-sm text-white"
        >
          取消
        </button>
        <button
  onClick={async () => {
    if (!groupForm.name.trim()) return alert("请输入群聊名称");
    try {
      const res = await api.createGroup({
        name: groupForm.name,
        notice: groupForm.notice,
        avatar: groupForm.avatar,
        ownerId: user?.uuid || "",
      });

      const groupUUID = res.data?.group_uuid; // ✅ 后端返回的群ID
      alert(res.data?.message || "群聊创建成功");

      setShowCreateGroup(false);
      await loadContacts(); // 刷新联系人和群聊

      if (groupUUID) {
        // ✅ 自动进入新建群聊
        setActiveId(groupUUID);
      }
    } catch (e: any) {
      console.error(e);
      alert(e.response?.data?.error || "创建失败");
    }
  }}
  className="px-3 py-2 rounded bg-green-600 hover:bg-green-700 text-sm text-white"
>
  创建
</button>

      </div>
    </div>
  </div>
)}

{showJoinGroup && (
  <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
    <div className="bg-[#2e2e2e] w-[360px] rounded-xl shadow-lg p-6 space-y-4">
      <h2 className="text-lg font-semibold text-gray-200">加入群聊</h2>

      <input
        placeholder="输入群聊 UUID"
        className="w-full bg-[#3a3b3d] rounded px-3 py-2 text-gray-200 outline-none"
        value={joinGroupId}
        onChange={(e) => setJoinGroupId(e.target.value)}
      />

      <div className="flex justify-end space-x-2 pt-2">
        <button
          onClick={() => setShowJoinGroup(false)}
          className="px-3 py-2 rounded bg-gray-500 hover:bg-gray-600 text-sm text-white"
        >
          取消
        </button>
        <button
          onClick={async () => {
            if (!joinGroupId.trim()) return alert("请输入群聊 UUID");
            try {
              const res = await api.enterGroup({ groupId: joinGroupId });
              alert(res.data?.message || "申请成功");
              setShowJoinGroup(false);
              await loadContacts(); // 更新联系人
            } catch (e) {
              console.error(e);
              alert("加入失败");
            }
          }}
          className="px-3 py-2 rounded bg-green-600 hover:bg-green-700 text-sm text-white"
        >
          加入
        </button>
      </div>
    </div>
  </div>
)}

{/* ========== 群成员列表模态框 ========== */}
{showGroupMembers && (
  <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
    <div
      className="bg-white/95 backdrop-blur-md rounded-xl shadow-2xl w-[400px] p-6 relative"
      onClick={(e) => e.stopPropagation()}
    >
      <button
        onClick={() => setShowGroupMembers(false)}
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
              {isGroupOwner && m !== user?.uuid && (
                <button
                  onClick={async () => {
                    if (!window.confirm("确定要移除该成员吗？")) return;
                    try {
                      await api.removeMember({
                        groupUuid: active?.id!,
                        targetUserId: m,
                      });
                      alert("已移除成员");
                      await loadGroupMembers();
                    } catch (e) {
                      alert("移除失败");
                    }
                  }}
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
            onClick={async () => {
              if (!window.confirm("确定退出该群聊吗？")) return;
              try {
                await api.leaveGroup({ groupUuid: active?.id! });
                alert("已退出群聊");
                setShowGroupMembers(false);
                await loadContacts();
              } catch (e) {
                alert("退出失败");
              }
            }}
            className="px-3 py-2 rounded bg-gray-200 hover:bg-gray-600 text-gray-300 text-sm"
          >
            退出群聊
          </button>
        )}

        {isGroupOwner && (
          <button
            onClick={async () => {
              if (!window.confirm("确定要解散群聊吗？")) return;
              try {
                await api.dismissGroup({ groupUuid: active?.id! });
                alert("群聊已解散");
                setShowGroupMembers(false);
                await loadContacts();
              } catch (e) {
                alert("解散失败");
              }
            }}
            className="px-3 py-2 rounded bg-red-600 hover:bg-red-700 text-white text-sm"
          >
            解散群聊
          </button>
        )}
      </div>
    </div>
  </div>
)}




      </main>

      <NewFriendModal
        open={showNewFriend}
        onClose={() => setShowNewFriend(false)}
        onRefreshContacts={loadContacts}
      />
    </div>
  );
};

export default Chat;
