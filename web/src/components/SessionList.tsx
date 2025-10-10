import React, { useEffect, useState } from "react";
import axios from "axios";

interface UserSession {
  user_id: string;
  user_name: string;
  avatar: string;
}

interface GroupSession {
  group_id: string;
  group_name: string;
  avatar: string;
}

interface SessionListProps {
  userId: string; // 当前登录用户
  onSelect: (id: string, isGroup: boolean) => void;
}

const SessionList: React.FC<SessionListProps> = ({ userId, onSelect }) => {
  const [userSessions, setUserSessions] = useState<UserSession[]>([]);
  const [groupSessions, setGroupSessions] = useState<GroupSession[]>([]);

  const loadUserSessions = async () => {
    try {
      const res = await axios.post("http://localhost:8000/session/getUserSessionList", {
        owner_id: userId,
      });
      if (res.data.data) {
        const list = res.data.data.map((u: UserSession) => ({
          ...u,
          avatar: u.avatar.startsWith("http")
            ? u.avatar
            : "http://localhost:8000" + u.avatar,
        }));
        setUserSessions(list);
      }
    } catch (err) {
      console.error("加载用户会话失败", err);
    }
  };

  const loadGroupSessions = async () => {
    try {
      const res = await axios.post("http://localhost:8000/session/getGroupSessionList", {
        owner_id: userId,
      });
      if (res.data.data) {
        const list = res.data.data.map((g: GroupSession) => ({
          ...g,
          avatar: g.avatar.startsWith("http")
            ? g.avatar
            : "http://localhost:8000" + g.avatar,
        }));
        setGroupSessions(list);
      }
    } catch (err) {
      console.error("加载群聊会话失败", err);
    }
  };

  useEffect(() => {
    loadUserSessions();
    loadGroupSessions();
  }, []);

  return (
    <div className="p-4 space-y-6 w-64 bg-gray-100 border-r overflow-y-auto">
      {/* 用户会话 */}
      <div>
        <h3 className="font-bold text-lg mb-2">用户</h3>
        {userSessions.map((u) => (
          <div
            key={u.user_id}
            onClick={() => onSelect(u.user_id, false)}
            className="flex items-center gap-2 p-2 rounded cursor-pointer hover:bg-gray-200"
          >
            <img src={u.avatar} alt="" className="w-8 h-8 rounded-full" />
            <span>{u.user_name}</span>
          </div>
        ))}
      </div>

      {/* 群聊会话 */}
      <div>
        <h3 className="font-bold text-lg mb-2">群聊</h3>
        {groupSessions.map((g) => (
          <div
            key={g.group_id}
            onClick={() => onSelect(g.group_id, true)}
            className="flex items-center gap-2 p-2 rounded cursor-pointer hover:bg-gray-200"
          >
            <img src={g.avatar} alt="" className="w-8 h-8 rounded-md" />
            <span>{g.group_name}</span>
          </div>
        ))}
      </div>
    </div>
  );
};

export default SessionList;
