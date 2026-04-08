// Chat.tsx
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
} from "../utils/chatUtils";
import { useWebRTC } from "../hooks/useWebRTC";

import Sidebar from "../components/chat/Sidebar";
import ChatHeader from "../components/chat/ChatHeader";
import ChatMessages from "../components/chat/ChatMessages";
import ChatInput from "../components/chat/ChatInput";
import CallWindow from "../components/chat/CallWindow";
import AddFriendModal from "../components/chat/AddFriendModal";
import ProfileModal from "../components/chat/ProfileModal";
import CreateGroupModal from "../components/chat/CreateGroupModal";
import JoinGroupModal from "../components/chat/JoinGroupModal";
import GroupMembersModal from "../components/chat/GroupMembersModal";
import GroupInfoModal from "../components/chat/GroupInfoModal";
import SearchHistoryModal from "../components/chat/SearchHistoryModal";
import NewFriendModal from "../components/NewFriendModal";

const PAGE_SIZE = 50;

const Chat: React.FC = () => {
  const { user, refreshUser, logout } = useAuth();

  // ===== 会话状态 =====
  const [sessions, setSessions] = useState<SessionItem[]>([]);
  const [activeId, setActiveId] = useState<string>(loadActiveId());
  const [sessionIndex, setSessionIndex] = useState<Record<string, "user" | "group">>({});
  const [groupIdSet, setGroupIdSet] = useState<Set<string>>(new Set());
  const [myGroups, setMyGroups] = useState<GroupInfo[]>([]);

  // 持久化 activeId 到 localStorage
  useEffect(() => { if (activeId) saveActiveId(activeId); }, [activeId]);

  // ===== 消息状态 =====
  const [messagesMap, setMessagesMap] = useState<Record<string, Message[]>>({});
  const [hasMoreMap, setHasMoreMap] = useState<Record<string, boolean>>({});
  const [input, setInput] = useState("");
  const listRef = useRef<HTMLDivElement>(null);
  const lastActiveId = useRef<string>("");

  // ===== 未读消息数 =====
  const [unreadCounts, setUnreadCounts] = useState<Record<string, number>>({});

  // ===== 在线状态 =====
  const [onlineUsers, setOnlineUsers] = useState<Set<string>>(new Set());

  // activeId ref（避免 stale closure 问题）
  const activeIdRef = useRef<string>("");
  useEffect(() => { activeIdRef.current = activeId; }, [activeId]);

  // ===== WebSocket =====
  const [ws, setWs] = useState<ChatWebSocket | null>(null);
  const wsRef = useRef<ChatWebSocket | null>(null);

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

  // ===== WebRTC =====
  const { callState, localVideoRef, remoteVideoRef, startCall, handleCallSignal, endCall } =
    useWebRTC(wsRef, user?.uuid);

  const active = sessions.find((s) => s.id === activeId);

  // ===== 加载群成员 =====
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

  // ===== 工具：解析消息数组 =====
  const parseMessages = (raw: any[]): Message[] =>
    raw.map((m: any) => ({
      uuid: m.uuid ?? m.Uuid,
      sendId: m.sendId ?? m.SendId ?? "",
      receiveId: m.receiveId ?? m.ReceiveId ?? "",
      content: (m.isRecalled || m.IsRecalled) ? "" : (m.content ?? m.Content ?? ""),
      type: m.type ?? m.Type ?? 0,
      createdAt:
        typeof m.createdAt === "number"
          ? m.createdAt
          : m.CreatedAt
          ? Math.floor(Date.parse(m.CreatedAt) / 1000)
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

  // ===== 加载历史消息 =====
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
        arr.reverse(); // 后端 DESC → 前端 ASC

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

        // 首次加载后标记已读（单聊）
        if (!isLoadMore && currentActive.type === "user" && user?.uuid) {
          // 立即在本地标记为已读（即时性）
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

        // 首次加载：直接跳到底部
        if (!isLoadMore) {
          // 等 DOM 更新完毕后跳底部（用 setTimeout 0）
          setTimeout(() => scrollToBottom(false), 0);
        }
      } catch (e) {
        console.error("加载历史消息失败：", e);
      }
    },
    [sessions, messagesMap, user?.uuid, scrollToBottom]
  );

  // ===== WS 消息处理 =====
  const handleIncomingMessage = useCallback(
    (msg: IncomingMessage) => {
      if (["call_invite", "call_answer", "call_candidate", "call_end"].includes((msg as any).action)) {
        handleCallSignal(msg);
        return;
      }

      const anyMsg = msg as any;

      // 在线状态事件
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

      // 群解散
      if (anyMsg.action === "group_dismissed" && anyMsg.groupId) {
        const gid = String(anyMsg.groupId);
        setSessions((prev) => prev.filter((s) => s.id !== gid));
        setMyGroups((prev) => prev.filter((g) => g.uuid !== gid));
        setMessagesMap((prev) => { const next = { ...prev }; delete next[gid]; saveMessagesToStorage(next); return next; });
        setActiveId((prev: string) => prev === gid ? "" : prev);
        return;
      }

      // 新成员加入
      if (anyMsg.action === "group_join" && anyMsg.groupId) {
        if (anyMsg.member_cnt !== undefined) {
          setMyGroups((prev) => prev.map((g) => g.uuid === anyMsg.groupId ? { ...g, member_cnt: anyMsg.member_cnt } : g));
        }
        if (activeId === anyMsg.groupId) loadGroupMembers?.();
        return;
      }

      // 退群
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

      // 消息撤回
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

      // 已读通知（发送方收到：消息被读了）
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

      // 普通消息
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

      // 未读计数：排除自己发的、排除当前活跃会话中对方发的
      const isMine = anyMsg.sendId === user?.uuid;
      const isCurrentSession = bucketId === activeIdRef.current;
      if (!isMine && !isCurrentSession) {
        setUnreadCounts((prev) => ({ ...prev, [bucketId]: (prev[bucketId] || 0) + 1 }));
      }

      // 若当前会话有新消息，且是对方发的，立即标记已读
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

      // 新消息到达当前会话，平滑滚到底部
      if (isCurrentSession) {
        setTimeout(() => scrollToBottom(true), 50);
      }
    },
    [groupIdSet, sessionIndex, user?.uuid, loadGroupMembers, handleCallSignal, scrollToBottom]
  );

  // ===== 加载联系人/群聊 =====
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

      // 首次加载：恢复上次的会话，若没有则选第一个
      setActiveId((prev) => {
        if (prev && unique.find((s) => s.id === prev)) return prev;
        return unique[0]?.id || "";
      });
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

  // ===== 发送文件 =====
  const handleSendFile = async (file: File) => {
    if (!ws || !activeId) return;
    try {
      const formData = new FormData();
      formData.append("file", file);
      const res = await api.uploadFile(formData);
      const fileUrl = res.data?.url || "";
      sendFileMessage(ws, fileUrl, activeId, file.name, file.type, String(file.size));
    } catch (e) {
      console.error("文件上传失败", e);
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
  }, []);

  // ===== 加载更多历史消息 =====
  const handleLoadMore = useCallback(() => {
    if (activeId) loadMessages(activeId, true);
  }, [activeId, loadMessages]);

  // ===== Effects =====
  useEffect(() => { loadContacts(); }, []);

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

  // 切换会话时：加载消息、拉群信息
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

  // WebSocket 连接
  useEffect(() => {
    const token = getToken();
    if (!token) return;
    const socket = new ChatWebSocket({
      token,
      onOpen: () => {
        myGroups.forEach((g) => socket.send({ action: "join_group", groupId: g.uuid }));
      },
      onMessage: handleIncomingMessage,
      onClose: () => console.log("❌ WebSocket 已关闭"),
    });
    setWs(socket);
    wsRef.current = socket;
    return () => socket.close();
  }, [myGroups]);

  const handleLogout = () => {
    localStorage.removeItem("chat_messages");
    localStorage.removeItem("chat_activeId");
    logout();
  };

  const viewMsgs = messagesMap[activeId] || [];
  const hasMore = hasMoreMap[activeId] || false;

  return (
    <div className="h-screen w-screen overflow-hidden bg-[#ededed] flex">
      <Sidebar
        user={user}
        avatarVersion={avatarVersion}
        sessions={sessions}
        activeId={activeId}
        unreadCounts={unreadCounts}
        onlineUsers={onlineUsers}
        onSelectSession={handleSelectSession}
        onShowProfile={() => setShowProfile(true)}
        onLogout={handleLogout}
        onShowNewFriend={() => setShowNewFriend(true)}
        onShowAddFriend={() => setShowAddFriend(true)}
        onShowCreateGroup={() => setShowCreateGroup(true)}
        onShowJoinGroup={() => setShowJoinGroup(true)}
      />

      <main className="flex-1 flex flex-col bg-[#f5f5f5] min-w-0">
        <ChatHeader
          active={active}
          avatarVersion={avatarVersion}
          groupMemberCount={groupMembers.length}
          groupNotice={groupNotice}
          onShowGroupMembers={() => { setShowGroupMembers(true); loadGroupMembers(); }}
          onShowGroupInfo={() => setShowGroupInfo(true)}
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
          onLoadMore={handleLoadMore}
          onRecall={handleRecall}
        />

        <ChatInput
          input={input}
          activeId={activeId}
          active={active}
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

      {/* Modals */}
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
    </div>
  );
};

export default Chat;
