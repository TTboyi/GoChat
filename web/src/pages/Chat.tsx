// Chat.tsx 是整个前端最核心的“聊天工作台”页面。
// 你可以把它理解成一个装配层：它自己不实现所有细节，
// 而是负责把“会话列表、消息列表、输入框、WebSocket、WebRTC、各种弹窗”
// 这些子系统拼成一个完整的聊天体验。
import React, { useEffect, useCallback, useRef, useState } from "react";
import { useAuth } from "../context/AuthContext";
import { ChatWebSocket, sendTextMessage, sendFileMessage } from "../api/socket";
import type { IncomingMessage } from "../api/socket";
import api from "../api/api";
import { getToken } from "../utils/session";

import type { SessionItem, Message, GroupInfo } from "../types/chat";
import {
  saveMessagesToStorage,
  loadMessagesFromStorage,
  saveActiveId,
  loadActiveId,
  saveRemark,
  loadRemarks,
  clearContactData,
} from "../utils/chatUtils";
import { useWebRTC } from "../hooks/useWebRTC";

import Sidebar from "../components/chat/Sidebar";
import ChatHeader from "../components/chat/ChatHeader";
import ChatMessages from "../components/chat/ChatMessages";
import ChatInput from "../components/chat/ChatInput";
import CallWindow from "../components/chat/CallWindow";
import IncomingCallModal from "../components/chat/IncomingCallModal";
import AddFriendModal from "../components/chat/AddFriendModal";
import ProfileModal from "../components/chat/ProfileModal";
import FriendProfileModal from "../components/chat/FriendProfileModal";
import CreateGroupModal from "../components/chat/CreateGroupModal";
import JoinGroupModal from "../components/chat/JoinGroupModal";
import GroupMembersModal from "../components/chat/GroupMembersModal";
import GroupInfoModal from "../components/chat/GroupInfoModal";
import SearchHistoryModal from "../components/chat/SearchHistoryModal";
import NewFriendModal from "../components/NewFriendModal";

const PAGE_SIZE = 50;
const MAX_FILE_SIZE = 30 * 1024 * 1024; // 30MB

const Chat: React.FC = () => {
  const { user, refreshUser, logout } = useAuth();

  // ===== 响应式：是否移动端 =====
  const [isMobile, setIsMobile] = useState(() => window.innerWidth < 768);
  const [mobileView, setMobileView] = useState<"sidebar" | "chat">("sidebar");

  useEffect(() => {
    const update = () => setIsMobile(window.innerWidth < 768);
    window.addEventListener("resize", update);
    return () => window.removeEventListener("resize", update);
  }, []);

  // ===== 会话状态 =====
  // sessions：左侧列表的所有项；
  // activeId：当前正在看的会话；
  // sessionIndex / groupIdSet：帮助快速判断某个 id 是用户还是群。
  const [sessions, setSessions] = useState<SessionItem[]>([]);
  const [activeId, setActiveId] = useState<string>(loadActiveId());
  const [sessionIndex, setSessionIndex] = useState<Record<string, "user" | "group">>({});
  const [groupIdSet, setGroupIdSet] = useState<Set<string>>(new Set());
  const [myGroups, setMyGroups] = useState<GroupInfo[]>([]);

  useEffect(() => { if (activeId) saveActiveId(activeId); }, [activeId]);

  // ===== 消息状态 =====
  // messagesMap 采用“bucketId -> 消息数组”的结构，
  // 这样私聊和群聊都能用统一方式按会话分桶管理。
  const [messagesMap, setMessagesMap] = useState<Record<string, Message[]>>({});
  const [hasMoreMap, setHasMoreMap] = useState<Record<string, boolean>>({});
  const [input, setInput] = useState("");
  const listRef = useRef<HTMLDivElement>(null);
  const lastActiveId = useRef<string>("");

  // ===== 未读消息数 =====
  const [unreadCounts, setUnreadCounts] = useState<Record<string, number>>({});

  // ===== 在线状态 =====
  const [onlineUsers, setOnlineUsers] = useState<Set<string>>(new Set());

  const activeIdRef = useRef<string>("");
  useEffect(() => { activeIdRef.current = activeId; }, [activeId]);

  // ===== WebSocket =====
  // ws 用于触发重渲染，wsRef 用于在回调/Effect 中安全拿到最新连接实例。
  const [ws, setWs] = useState<ChatWebSocket | null>(null);
  const wsRef = useRef<ChatWebSocket | null>(null);
  // ✅ 始终指向最新的 handleIncomingMessage，避免 WS 闭包过期
  const handleIncomingMessageRef = useRef<((msg: any) => void) | null>(null);

  // ===== 群聊信息 =====
  const [groupMembers, setGroupMembers] = useState<string[]>([]);
  const [isGroupOwner, setIsGroupOwner] = useState(false);
  const [groupNotice, setGroupNotice] = useState<string>("");

  // ===== UI 控制 =====
  const [avatarVersion, setAvatarVersion] = useState(Date.now());
  const [showProfile, setShowProfile] = useState(false);
  const [showAddFriend, setShowAddFriend] = useState(false);
  const [showNewFriend, setShowNewFriend] = useState(false);
  const [showCreateGroup, setShowCreateGroup] = useState(false);
  const [showJoinGroup, setShowJoinGroup] = useState(false);
  const [showGroupMembers, setShowGroupMembers] = useState(false);
  const [showGroupInfo, setShowGroupInfo] = useState(false);
  const [showSearch, setShowSearch] = useState(false);

  // ===== 备注 =====
  const [remarks, setRemarks] = useState<Record<string, string>>(() => loadRemarks());

  // ===== 好友资料弹窗 =====
  const [friendProfile, setFriendProfile] = useState<{
    uuid: string; nickname: string; email?: string; avatar?: string; remark?: string;
  } | null>(null);

  // ===== 好友申请角标 =====
  const [newApplyCount, setNewApplyCount] = useState(0);

  // 好友申请自动弹窗控制：同一批次只弹一次
  const newApplyAutoShownRef = useRef(false);
  const showNewFriendRef = useRef(false);
  useEffect(() => { showNewFriendRef.current = showNewFriend; }, [showNewFriend]);

  // ===== 界面风格（深色/浅色）=====
  const [isDark, setIsDark] = useState<boolean>(() => {
    const saved = localStorage.getItem("chat_theme");
    return saved !== null ? saved === "dark" : true; // 默认深色
  });
  const handleToggleTheme = () => {
    setIsDark((prev) => {
      const next = !prev;
      localStorage.setItem("chat_theme", next ? "dark" : "light");
      return next;
    });
  };

  // ===== WebRTC =====
  // 通话相关复杂状态被抽到了 useWebRTC 里，Chat 页面只负责把 active 会话传进去。
  const {
    callState,
    incomingCall,
    localVideoRef,
    remoteVideoRef,
    startCall,
    handleCallSignal,
    endCall,
    acceptIncomingCall,
    rejectIncomingCall,
  } = useWebRTC(wsRef, user?.uuid);

  const active = sessions.find((s) => s.id === activeId);

  // ===== 备注 → 会话名显示（用备注替换真实昵称）=====
  const sessionsWithRemark = sessions.map((s) =>
    s.type === "user" && remarks[s.id] ? { ...s, name: remarks[s.id] } : s
  );

  // loadGroupMembers 专门服务于“查看群成员 / 群资料”等弹窗。
  const loadGroupMembers = useCallback(async () => {
    if (!active || active.type !== "group") return;
    try {
      const res = await api.getGroupMembers(active.id);
      const members = res.data?.members || res.data?.data || [];
      setGroupMembers(members);
      const groupInfo = myGroups.find((g) => g.uuid === active.id);
      setIsGroupOwner(groupInfo?.owner_id === user?.uuid);
    } catch {
      setGroupMembers([]);
    }
  }, [active, myGroups, user?.uuid]);

  // ===== 滚动到底部 =====
  const scrollToBottom = useCallback((smooth = false) => {
    if (!listRef.current) return;
    if (smooth) {
      listRef.current.scrollTo({ top: listRef.current.scrollHeight, behavior: "smooth" });
    } else {
      listRef.current.scrollTop = listRef.current.scrollHeight;
    }
  }, []);

  // parseMessages 用于兼容后端返回的大小写差异和字段差异，
  // 让后续渲染层永远消费统一的 Message 结构。
  const parseMessages = (raw: any[]): Message[] =>
    raw.map((m: any) => ({
      uuid: m.uuid ?? m.Uuid,
      sendId: m.sendId ?? m.SendId ?? "",
      receiveId: m.receiveId ?? m.ReceiveId ?? "",
      content: (m.isRecalled || m.IsRecalled) ? "" : (m.content ?? m.Content ?? m.url ?? m.Url ?? ""),
      type: m.type ?? m.Type ?? 0,
      createdAt:
        typeof m.createdAt === "number"
          ? m.createdAt
          : (m.createdAt || m.CreatedAt)
          ? Math.floor(Date.parse(m.createdAt || m.CreatedAt) / 1000)
          : Math.floor(Date.now() / 1000),
      sendName: m.sendName ?? m.SendName ?? "",
      sendAvatar: m.sendAvatar ?? m.SendAvatar ?? "",
      url: m.url ?? m.Url ?? "",
      fileName: m.fileName ?? m.FileName ?? "",
      fileType: m.fileType ?? m.FileType ?? "",
      fileSize: m.fileSize ?? m.FileSize ?? "",
      isRecalled: !!(m.isRecalled || m.IsRecalled),
      readAt: m.readAt ?? m.ReadAt ?? null,
    }));

  // loadMessages 负责拉历史消息，并处理“首次进入会话”和“上拉加载更多”两条路径。
  const loadMessages = useCallback(
    async (sessionId: string, isLoadMore = false) => {
      const currentActive = sessions.find((s) => s.id === sessionId);
      if (!currentActive) return;
      try {
        const existingMsgs = isLoadMore ? (messagesMap[sessionId] || []) : [];
        const oldestTs = isLoadMore && existingMsgs.length > 0
          ? Math.min(...existingMsgs.map((m) =>
              typeof m.createdAt === "number" ? m.createdAt : Math.floor(Date.parse(m.createdAt as string) / 1000)
            ))
          : 0;
        const beforeTime = isLoadMore && oldestTs > 0 ? oldestTs : 0;

        let res;
        if (currentActive.type === "group") {
          res = await api.getGroupMessageList({ groupId: currentActive.id, limit: PAGE_SIZE, beforeTime });
        } else {
          res = await api.getMessageList({ targetId: currentActive.id, limit: PAGE_SIZE, beforeTime });
        }
        const raw = (res.data?.data || res.data || []) as any[];
        const arr = parseMessages(raw);
        arr.reverse();

        setHasMoreMap((prev) => ({ ...prev, [sessionId]: arr.length === PAGE_SIZE }));

        setMessagesMap((prev) => {
          if (isLoadMore) {
            const merged = [...arr, ...(prev[sessionId] || [])];
            const unique = Array.from(new Map(merged.map((m) => [m.uuid, m])).values())
              .sort((a, b) => {
                const ta = typeof a.createdAt === "number" ? a.createdAt : Date.parse(a.createdAt as string) / 1000;
                const tb = typeof b.createdAt === "number" ? b.createdAt : Date.parse(b.createdAt as string) / 1000;
                return ta - tb;
              });
            const updated = { ...prev, [sessionId]: unique };
            saveMessagesToStorage(updated);
            return updated;
          } else {
            const updated = { ...prev, [sessionId]: arr };
            saveMessagesToStorage(updated);
            return updated;
          }
        });

        if (!isLoadMore && currentActive.type === "user" && user?.uuid) {
          setMessagesMap((prev) => {
            const list = prev[sessionId] || [];
            const updated = list.map((m) =>
              m.sendId === currentActive.id && !m.readAt
                ? { ...m, readAt: new Date().toISOString() }
                : m
            );
            const next = { ...prev, [sessionId]: updated };
            saveMessagesToStorage(next);
            return next;
          });
          api.markMessagesRead({ senderId: currentActive.id }).catch(() => {});
        }

        if (!isLoadMore) {
          setTimeout(() => scrollToBottom(false), 0);
        }
      } catch (e) {
        console.error("加载历史消息失败：", e);
      }
    },
    [sessions, messagesMap, user?.uuid, scrollToBottom]
  );

  // handleIncomingMessage 是前端实时消息总入口。
  // 所有服务端推送——包括普通聊天消息、系统事件、已读回执、群变更、通话信令——
  // 都会先进入这里，再按 action/type 分流到不同状态更新逻辑。
  const handleIncomingMessage = useCallback(
    (msg: IncomingMessage) => {
      if (["call_invite", "call_answer", "call_candidate", "call_end"].includes((msg as any).action)) {
        handleCallSignal(msg);
        return;
      }

      const anyMsg = msg as any;

      // 在线状态事件：由后端在用户连接/断开时主动广播。
      if (anyMsg.action === "online_users" && Array.isArray(anyMsg.userIds)) {
        setOnlineUsers(new Set(anyMsg.userIds as string[]));
        return;
      }
      if (anyMsg.action === "user_online" && anyMsg.userId) {
        setOnlineUsers((prev) => new Set([...prev, anyMsg.userId as string]));
        return;
      }
      if (anyMsg.action === "user_offline" && anyMsg.userId) {
        setOnlineUsers((prev) => {
          const next = new Set(prev);
          next.delete(anyMsg.userId as string);
          return next;
        });
        return;
      }

      // ✅ 收到好友申请通知 - 更新角标并在本批次未自动弹出过的情况下弹窗
      if (anyMsg.action === "new_contact_apply") {
        setNewApplyCount((prev) => prev + 1);
        if (!newApplyAutoShownRef.current && !showNewFriendRef.current) {
          newApplyAutoShownRef.current = true;
          setShowNewFriend(true);
        }
        return;
      }

      // ✅ 好友申请被接受（申请方收到）- 刷新联系人列表
      if (anyMsg.action === "contact_apply_accepted") {
        loadContacts();
        return;
      }

      // ✅ 联系人列表更新（接受方收到）- 刷新联系人列表
      if (anyMsg.action === "contact_list_updated") {
        loadContacts();
        return;
      }

      if (anyMsg.action === "group_dismissed" && anyMsg.groupId) {
        const gid = String(anyMsg.groupId);
        setSessions((prev) => prev.filter((s) => s.id !== gid));
        setMyGroups((prev) => prev.filter((g) => g.uuid !== gid));
        setMessagesMap((prev) => { const next = { ...prev }; delete next[gid]; saveMessagesToStorage(next); return next; });
        setActiveId((prev: string) => prev === gid ? "" : prev);
        return;
      }

      if (anyMsg.action === "group_join" && anyMsg.groupId) {
        if (anyMsg.member_cnt !== undefined) {
          setMyGroups((prev) => prev.map((g) => g.uuid === anyMsg.groupId ? { ...g, member_cnt: anyMsg.member_cnt } : g));
        }
        if (activeId === anyMsg.groupId) loadGroupMembers?.();
        return;
      }

      if (anyMsg.action === "group_quit" && anyMsg.groupId) {
        const gid = String(anyMsg.groupId);
        if (anyMsg.userId === user?.uuid) {
          setSessions((prev) => prev.filter((s) => s.id !== gid));
          setMyGroups((prev) => prev.filter((g) => g.uuid !== gid));
          setMessagesMap((prev) => { const next = { ...prev }; delete next[gid]; saveMessagesToStorage(next); return next; });
          setActiveId((prev: string) => prev === gid ? "" : prev);
        } else {
          setGroupMembers((prev) => prev.filter((uid) => uid !== anyMsg.userId));
        }
        return;
      }

      if (anyMsg.action === "msg_recall" && anyMsg.msgId) {
        setMessagesMap((prev) => {
          const updated = { ...prev };
          for (const bucketId of Object.keys(updated)) {
            const list = updated[bucketId];
            const idx = list.findIndex((m) => m.uuid === anyMsg.msgId);
            if (idx !== -1) {
              const newList = [...list];
              newList[idx] = { ...newList[idx], isRecalled: true, content: "" };
              updated[bucketId] = newList;
              saveMessagesToStorage(updated);
              break;
            }
          }
          return updated;
        });
        return;
      }

      if (anyMsg.action === "msg_read" && anyMsg.receiverId) {
        const readerId = String(anyMsg.receiverId);
        setMessagesMap((prev) => {
          const bucket = prev[readerId];
          if (!bucket) return prev;
          const updated = {
            ...prev,
            [readerId]: bucket.map((m) =>
              m.sendId === user?.uuid && !m.readAt ? { ...m, readAt: new Date().toISOString() } : m
            ),
          };
          saveMessagesToStorage(updated);
          return updated;
        });
        return;
      }

      const newMsg: Message = {
        uuid: anyMsg.uuid,
        sendId: anyMsg.sendId ?? "",
        receiveId: anyMsg.receiveId ?? "",
        content: anyMsg.content ?? "",
        type: msg.type ?? 0,
        createdAt: msg.createdAt ?? Math.floor(Date.now() / 1000),
        sendName: anyMsg.sendName ?? "",
        sendAvatar: anyMsg.sendAvatar ?? "",
        url: anyMsg.url ?? "",
        fileName: anyMsg.fileName ?? "",
        fileType: anyMsg.fileType ?? "",
        fileSize: anyMsg.fileSize ?? "",
        isRecalled: false,
        readAt: null,
      };

      // bucketId 决定这条消息应该归到哪个会话桶里。
      // 私聊时，自己发出的消息按对方 id 分桶；收到的消息按发送方 id 分桶；
      // 群聊时，统一按群 id 分桶。
      const isGroupMsg =
        (anyMsg.receiveId && sessionIndex[anyMsg.receiveId] === "group") ||
        (anyMsg.receiveId && groupIdSet.has(anyMsg.receiveId));

      const bucketId: string = isGroupMsg
        ? anyMsg.receiveId
        : anyMsg.sendId === user?.uuid
        ? anyMsg.receiveId
        : anyMsg.sendId;

      setMessagesMap((prev) => {
        const list = prev[bucketId] || [];
        if (newMsg.uuid && list.some((m) => m.uuid === newMsg.uuid)) return prev;
        const next = { ...prev, [bucketId]: [...list, newMsg] };
        saveMessagesToStorage(next);
        return next;
      });

      const isMine = anyMsg.sendId === user?.uuid;
      const isCurrentSession = bucketId === activeIdRef.current;
      if (!isMine && !isCurrentSession) {
        setUnreadCounts((prev) => ({ ...prev, [bucketId]: (prev[bucketId] || 0) + 1 }));
      }

      if (!isGroupMsg && isCurrentSession && !isMine) {
        setMessagesMap((prev) => {
          const list = prev[bucketId] || [];
          const updated = list.map((m) =>
            m.sendId !== user?.uuid && !m.readAt ? { ...m, readAt: new Date().toISOString() } : m
          );
          const next = { ...prev, [bucketId]: updated };
          saveMessagesToStorage(next);
          return next;
        });
        api.markMessagesRead({ senderId: anyMsg.sendId }).catch(() => {});
      }

      if (isCurrentSession) {
        setTimeout(() => scrollToBottom(true), 50);
      }
    },
    [groupIdSet, sessionIndex, user?.uuid, loadGroupMembers, handleCallSignal, scrollToBottom, activeId]
  );

  // loadContacts 会并行拉取“好友列表 + 已加入群聊”，然后组装成左侧会话栏数据源。
  const loadContacts = useCallback(async () => {
    try {
      const [userRes, groupRes] = await Promise.all([
        api.getContactList(),
        api.loadMyJoinedGroup(),
      ]);

      const contactList = (userRes.data?.data || userRes.data || []) as any[];
      const groupList = (groupRes.data?.data || groupRes.data || []) as any[];

      const groups: SessionItem[] = groupList.map((g: any) => ({
        id: g.uuid || g.Uuid,
        name: g.name || g.Name || "群聊",
        avatar: g.avatar || g.Avatar || "",
        type: "group",
      }));

      setMyGroups(groupList.map((g: any) => ({
        uuid: g.uuid || g.Uuid,
        name: g.name || g.Name || "群聊",
        avatar: g.avatar || g.Avatar || "",
        owner_id: g.owner_id || g.OwnerId,
        notice: g.notice || g.Notice || "",
        member_cnt: g.member_cnt || g.MemberCnt,
      })));

      const contacts: SessionItem[] = contactList.map((it: any) => ({
        id: it.uuid,
        name: it.nickname || "好友",
        avatar: it.avatar || "",
        type: "user",
      }));

      const merged = [...groups, ...contacts];
      const unique = Array.from(new Map(merged.map((x) => [x.id, x])).values());
      setSessions(unique);

      const idx: Record<string, "user" | "group"> = {};
      groups.forEach((g) => { if (g.id) idx[g.id] = "group"; });
      contacts.forEach((c) => { if (c.id) idx[c.id] = "user"; });
      setSessionIndex(idx);
      setGroupIdSet(new Set(groups.map((g) => g.id)));
      // ✅ 不在这里调用 setActiveId，由独立的 sessions effect 负责，避免每次刷新联系人都重置当前对话
    } catch (e) {
      console.error("加载联系人或群聊失败：", e);
    }
  }, []);

  // ===== 发送文本消息 =====
  const doSend = () => {
    if (!ws || !input.trim() || !activeId) return;
    sendTextMessage(ws, input.trim(), activeId);
    setInput("");
    setTimeout(() => scrollToBottom(true), 50);
  };

  const onKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      doSend();
    }
  };

  // ===== 发送文件（30MB 限制） =====
  const handleSendFile = async (file: File) => {
    if (!ws || !activeId) return;
    if (file.size > MAX_FILE_SIZE) {
      alert("文件大小不能超过 30MB");
      return;
    }
    try {
      const formData = new FormData();
      formData.append("file", file);
      const res = await api.uploadFile(formData);
      const fileUrl = res.data?.url || "";
      sendFileMessage(ws, fileUrl, activeId, file.name, file.type, String(file.size));
    } catch (e: any) {
      alert(e.response?.data?.error || "文件上传失败");
    }
  };

  // ===== 撤回消息 =====
  const handleRecall = async (msg: Message) => {
    if (!msg.uuid) return;
    try {
      await api.recallMessage({ msgId: msg.uuid, receiveId: msg.receiveId });
    } catch (e: any) {
      alert(e.response?.data?.error || "撤回失败");
    }
  };

  // ===== 切换会话 =====
  const handleSelectSession = useCallback((id: string) => {
    setActiveId(id);
    setUnreadCounts((prev) => ({ ...prev, [id]: 0 }));
    setMobileView("chat");
  }, []);

  // ===== 加载更多历史消息 =====
  const handleLoadMore = useCallback(() => {
    if (activeId) loadMessages(activeId, true);
  }, [activeId, loadMessages]);

  // ✅ 每次渲染都同步最新的 handleIncomingMessage 到 ref（必须在 effects 之前赋值）
  handleIncomingMessageRef.current = handleIncomingMessage;

  // ===== Effects =====
  // 这一段可以重点学习：Chat 页面很多“初始化 / 同步 / 副作用”都是靠多个 useEffect 配合完成的。
  useEffect(() => {
    loadContacts();
    api.getNewContactList().then((res) => {
      const list = res.data?.data || [];
      setNewApplyCount(list.length);
    }).catch(() => {});
  }, []);

  // ✅ sessions 变化时，只在未初始化或当前会话失效时才调整 activeId
  const activeIdInitialized = useRef(false);
  useEffect(() => {
    if (sessions.length === 0) return;
    setActiveId((prev) => {
      // 当前已有有效会话，保持不变
      if (prev && sessions.find((s) => s.id === prev)) {
        activeIdInitialized.current = true;
        return prev;
      }
      // 初次加载或当前会话已失效，回退到第一个
      activeIdInitialized.current = true;
      return sessions[0]?.id || "";
    });
  }, [sessions]);

  useEffect(() => {
    const stored = loadMessagesFromStorage();
    setMessagesMap(stored);
  }, []);

  useEffect(() => {
    if (user?.avatar) setAvatarVersion(Date.now());
  }, [user?.avatar]);

  useEffect(() => {
    if (!user) refreshUser();
  }, []);

  useEffect(() => {
    if (!activeId || sessions.length === 0) return;
    if (lastActiveId.current === activeId) return;
    lastActiveId.current = activeId;

    loadMessages(activeId, false);

    const cur = sessions.find((s) => s.id === activeId);
    if (cur?.type === "group") {
      api.getGroupMembers(activeId).then((res) => {
        setGroupMembers(res.data?.members || res.data?.data || []);
      }).catch(() => setGroupMembers([]));
      api.getGroupInfo(activeId).then((res) => {
        setGroupNotice(res.data?.notice || "");
        const groupInfo = myGroups.find((g) => g.uuid === activeId);
        setIsGroupOwner(groupInfo?.owner_id === user?.uuid);
      }).catch(() => {});
    } else {
      setGroupMembers([]);
      setGroupNotice("");
    }
  }, [activeId, sessions.length]);

  // WebSocket 只在首次进入页面或 token 变化时建立连接。
  // 如果把 myGroups 也放进依赖数组，会导致“群列表变化 -> 重建整条连接”的副作用。
  useEffect(() => {
    const token = getToken();
    if (!token) return;
    const socket = new ChatWebSocket({
      token,
      // ✅ 始终通过 ref 派发，保证调用的是最新版 handleIncomingMessage
      onMessage: (msg) => handleIncomingMessageRef.current?.(msg),
      onClose: () => console.log("WebSocket 已关闭"),
    });
    setWs(socket);
    wsRef.current = socket;
    return () => socket.close();
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []); // 只在挂载时建立一次

  // 群订阅与“建连”拆开处理：连接复用，订阅单独补发。
  useEffect(() => {
    if (!wsRef.current || myGroups.length === 0) return;
    myGroups.forEach((g) => wsRef.current!.send({ action: "join_group", groupId: g.uuid }));
  }, [myGroups]);

  const handleLogout = () => {
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    localStorage.removeItem("chat_messages");
    localStorage.removeItem("chat_activeId");
    logout();
  };

  // ===== 布局计算 =====
  // 桌面端同时显示左侧会话栏和右侧聊天区；
  // 移动端只显示一个视图，通过 mobileView 切换。
  // 桌面：始终显示两栏；移动：按 mobileView 切换
  const showSidebar = !isMobile || mobileView === "sidebar";
  const showMain = !isMobile || mobileView === "chat";

  const viewMsgs = messagesMap[activeId] || [];
  const hasMore = hasMoreMap[activeId] || false;

  return (
    <div className={`h-screen w-screen overflow-hidden flex ${isDark ? "bg-[#242424]" : "bg-gray-100"}`}>
      {/* ===== Sidebar ===== */}
      <div
        style={{
          display: showSidebar ? "flex" : "none",
          width: isMobile ? "100%" : "260px",
          flexShrink: 0,
          flexDirection: "column",
        }}
      >
        <Sidebar
          user={user}
          avatarVersion={avatarVersion}
          sessions={sessionsWithRemark}
          activeId={activeId}
          unreadCounts={unreadCounts}
          onlineUsers={onlineUsers}
          isDark={isDark}
          newApplyCount={newApplyCount}
          onToggleTheme={handleToggleTheme}
          onSelectSession={handleSelectSession}
          onShowProfile={() => setShowProfile(true)}
          onLogout={handleLogout}
          onShowNewFriend={() => {
            newApplyAutoShownRef.current = false;
            setNewApplyCount(0);
            setShowNewFriend(true);
          }}
          onShowAddFriend={() => setShowAddFriend(true)}
          onShowCreateGroup={() => setShowCreateGroup(true)}
          onShowJoinGroup={() => setShowJoinGroup(true)}
        />
      </div>

      {/* ===== 主聊天区域 ===== */}
      <main
        style={{ display: showMain ? "flex" : "none" }}
        className={`flex-1 flex-col min-w-0 ${isDark ? "bg-[#1e1e1e]" : "bg-gray-50"}`}
      >
        {/* 移动端顶部返回栏 */}
        {isMobile && (
          <div className={`flex items-center h-11 px-3 border-b flex-shrink-0 ${
            isDark ? "bg-[#2e2e2e] border-black/20 text-gray-200" : "bg-white border-gray-200 text-gray-800"
          }`}>
            <button
              onClick={() => setMobileView("sidebar")}
              className={`flex items-center gap-1 text-sm py-1 px-1 -ml-1 bg-transparent border-none ${
                isDark ? "text-blue-400" : "text-blue-500"
              }`}
            >
              <svg viewBox="0 0 24 24" className="w-5 h-5" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                <polyline points="15 18 9 12 15 6" />
              </svg>
              返回
            </button>
            <span className="mx-auto font-semibold text-sm truncate pr-10">{active?.name || ""}</span>
          </div>
        )}

        <ChatHeader
          active={active && active.type === "user" && remarks[active.id]
            ? { ...active, name: remarks[active.id] }
            : active}
          avatarVersion={avatarVersion}
          groupMemberCount={groupMembers.length}
          groupNotice={groupNotice}
          isDark={isDark}
          onShowGroupMembers={() => { setShowGroupMembers(true); loadGroupMembers(); }}
          onShowGroupInfo={() => setShowGroupInfo(true)}
          onShowFriendProfile={() => {
            if (!active || active.type !== "user") return;
            setFriendProfile({
              uuid: active.id,
              nickname: sessions.find((s) => s.id === active.id)?.name || active.name,
              avatar: active.avatar,
              remark: remarks[active.id],
            });
          }}
        />

        <ChatMessages
          messages={viewMsgs}
          activeId={activeId}
          active={active}
          userId={user?.uuid}
          avatarVersion={avatarVersion}
          userAvatar={user?.avatar}
          listRef={listRef}
          hasMore={hasMore}
          isDark={isDark}
          onLoadMore={handleLoadMore}
          onRecall={handleRecall}
        />

        <ChatInput
          input={input}
          activeId={activeId}
          active={active}
          isDark={isDark}
          onChange={setInput}
          onKeyDown={onKeyDown}
          onSend={doSend}
          onStartCall={(type) => active && startCall(type, active)}
          onSendFile={handleSendFile}
          onOpenSearch={() => setShowSearch(true)}
        />

        <CallWindow
          callState={callState}
          localVideoRef={localVideoRef}
          remoteVideoRef={remoteVideoRef}
          onEndCall={endCall}
        />
      </main>

      {/* ===== 来电提示 ===== */}
      {incomingCall && (
        <IncomingCallModal
          call={incomingCall}
          onAccept={acceptIncomingCall}
          onReject={rejectIncomingCall}
        />
      )}

      {/* ===== Modals ===== */}
      <AddFriendModal open={showAddFriend} onClose={() => setShowAddFriend(false)} />

      <ProfileModal
        open={showProfile}
        onClose={() => setShowProfile(false)}
        user={user}
        avatarVersion={avatarVersion}
        onRefreshUser={refreshUser}
        onAvatarUpdated={() => setAvatarVersion(Date.now())}
      />

      <CreateGroupModal
        open={showCreateGroup}
        onClose={() => setShowCreateGroup(false)}
        userId={user?.uuid}
        onCreated={async (groupUUID) => {
          await loadContacts();
          setActiveId(groupUUID);
          setMobileView("chat");
        }}
      />

      <JoinGroupModal
        open={showJoinGroup}
        onClose={() => setShowJoinGroup(false)}
        onJoined={loadContacts}
      />

      <GroupMembersModal
        open={showGroupMembers}
        onClose={() => setShowGroupMembers(false)}
        groupId={activeId}
        groupMembers={groupMembers}
        isGroupOwner={isGroupOwner}
        userId={user?.uuid}
        onRefreshMembers={loadGroupMembers}
        onRefreshContacts={loadContacts}
      />

      <GroupInfoModal
        open={showGroupInfo}
        onClose={() => setShowGroupInfo(false)}
        groupId={activeId}
        onAvatarUpdated={() => setAvatarVersion(Date.now())}
      />

      <SearchHistoryModal
        open={showSearch}
        onClose={() => setShowSearch(false)}
        messages={viewMsgs}
        userId={user?.uuid}
      />

      <NewFriendModal
        open={showNewFriend}
        onClose={() => setShowNewFriend(false)}
        onRefreshContacts={loadContacts}
      />

      {/* ===== 好友资料弹窗 ===== */}
      <FriendProfileModal
        open={!!friendProfile}
        friend={friendProfile}
        avatarVersion={avatarVersion}
        onClose={() => setFriendProfile(null)}
        onRemarkSaved={(friendId, remark) => {
          saveRemark(friendId, remark);
          setRemarks(loadRemarks());
          setFriendProfile((prev) => prev ? { ...prev, remark } : null);
        }}
        onDeleteFriend={async (friendId) => {
          try {
            await api.deleteContact({ userId: friendId });
            // 尝试清除服务端消息（接口可能不存在时静默失败）
            await api.clearConversation({ targetId: friendId }).catch(() => {});
          } catch {}
          // 清除本地数据
          const newMap = clearContactData(friendId, messagesMap);
          setMessagesMap(newMap);
          saveMessagesToStorage(newMap);
          setRemarks(loadRemarks());
          // 从会话列表移除
          setSessions((prev) => prev.filter((s) => s.id !== friendId));
          if (activeId === friendId) setActiveId("");
          setFriendProfile(null);
        }}
      />
    </div>
  );
};

export default Chat;
