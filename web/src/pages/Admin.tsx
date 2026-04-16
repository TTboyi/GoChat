// ============================================================
// 文件：web/src/pages/Admin.tsx
// 作用：后台管理界面，仅管理员可访问（AdminRoute 守卫）。
//
// 功能区域：
//   - 系统概况卡片（总用户数、总群数、总消息数）
//   - 趋势图表（每日新增用户 & 消息量，使用 Chart.js 绘制折线图）
//   - 用户管理表格（查看所有用户、封禁/解封）
//   - 群聊管理表格（查看所有群聊、强制解散）
//
// Chart.js 使用方式：
//   通过 canvas 元素 + new Chart(canvas, config) 创建图表。
//   组件销毁时（useEffect 的 cleanup 函数）调用 chart.destroy()，
//   防止 React 重渲染时出现"canvas 已被占用"的报错。
// ============================================================
import React, { useEffect, useState, useCallback } from "react";
import api from "../api/api";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from "recharts";

// ─── 类型定义 ───────────────────────────────────────────────
interface SystemStats {
  total_users: number;
  total_groups: number;
  total_messages: number;
  today_messages: number;
  today_new_users: number;
  text_messages: number;
  file_messages: number;
  online_users: number;
}

interface DailyStat {
  date: string;
  messages: number;
  new_users: number;
}

interface UserInfo {
  uuid: string;
  nickname: string;
  telephone: string;
  email: string;
  is_admin: number;
  status: number;
  created_at: string;
}

interface GroupInfo {
  uuid: string;
  name: string;
  member_cnt: number;
  status: number;
  created_at: string;
}

// ─── 统计卡片 ─────────────────────────────────────────────────
const StatCard: React.FC<{
  label: string;
  value: number;
  color: string;
  icon: string;
}> = ({ label, value, color, icon }) => (
  <div className={`bg-white rounded-xl shadow p-5 flex items-center gap-4 border-l-4 ${color}`}>
    <span className="text-3xl">{icon}</span>
    <div>
      <p className="text-sm text-gray-500">{label}</p>
      <p className="text-2xl font-bold text-gray-800">{value.toLocaleString()}</p>
    </div>
  </div>
);

// ─── 主页面 ───────────────────────────────────────────────────
const Admin: React.FC = () => {
  const [stats, setStats] = useState<SystemStats | null>(null);
  const [daily, setDaily] = useState<DailyStat[]>([]);
  const [users, setUsers] = useState<UserInfo[]>([]);
  const [groups, setGroups] = useState<GroupInfo[]>([]);
  const [tab, setTab] = useState<"users" | "groups">("users");
  const [loading, setLoading] = useState(true);
  const [dailyDays, setDailyDays] = useState(7);

  const loadStats = useCallback(async () => {
    try {
      const [statsRes, dailyRes] = await Promise.all([
        api.getSystemStats(),
        api.getAdminDailyStats(dailyDays),
      ]);
      setStats(statsRes.data?.data ?? statsRes.data);
      setDaily(dailyRes.data?.data ?? dailyRes.data ?? []);
    } catch (e) {
      console.error("获取统计失败", e);
    }
  }, [dailyDays]);

  const loadUsers = useCallback(async () => {
    try {
      const res = await api.getAllUsers();
      setUsers(res.data?.data ?? res.data ?? []);
    } catch (e) {
      console.error("获取用户列表失败", e);
    }
  }, []);

  const loadGroups = useCallback(async () => {
    try {
      const res = await api.getAllGroups();
      setGroups(res.data?.data ?? res.data ?? []);
    } catch (e) {
      console.error("获取群组列表失败", e);
    }
  }, []);

  useEffect(() => {
    setLoading(true);
    Promise.all([loadStats(), loadUsers(), loadGroups()]).finally(() =>
      setLoading(false)
    );
  }, [loadStats, loadUsers, loadGroups]);

  const handleBan = async (userId: string, currentStatus: number) => {
    const ban = currentStatus === 0;
    if (!window.confirm(ban ? `确认封禁该用户？` : `确认解封该用户？`)) return;
    try {
      await api.banUser(userId, ban);
      loadUsers();
    } catch (e) {
      alert("操作失败");
    }
  };

  const handleDismissGroup = async (groupId: string) => {
    if (!window.confirm("确认解散该群？此操作不可撤销。")) return;
    try {
      await api.adminDismissGroup(groupId);
      loadGroups();
    } catch (e) {
      alert("操作失败");
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <p className="text-gray-400 text-lg">加载中...</p>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 p-6">
      <div className="max-w-6xl mx-auto">
        {/* 标题 */}
        <div className="flex items-center justify-between mb-6">
          <h1 className="text-2xl font-bold text-gray-800">🛠 管理后台</h1>
          <button
            onClick={() => loadStats()}
            className="text-sm bg-white border rounded-lg px-3 py-1.5 text-gray-600 hover:bg-gray-100"
          >
            ↺ 刷新统计
          </button>
        </div>

        {/* 统计卡片 */}
        {stats && (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
            <StatCard label="在线用户" value={stats.online_users} color="border-green-400" icon="🟢" />
            <StatCard label="总用户数" value={stats.total_users} color="border-blue-400" icon="👥" />
            <StatCard label="今日消息" value={stats.today_messages} color="border-purple-400" icon="💬" />
            <StatCard label="总消息数" value={stats.total_messages} color="border-orange-400" icon="📊" />
            <StatCard label="总群组数" value={stats.total_groups} color="border-yellow-400" icon="👫" />
            <StatCard label="今日新用户" value={stats.today_new_users} color="border-teal-400" icon="🆕" />
            <StatCard label="文字消息" value={stats.text_messages} color="border-indigo-400" icon="📝" />
            <StatCard label="文件消息" value={stats.file_messages} color="border-pink-400" icon="📎" />
          </div>
        )}

        {/* 折线图 */}
        <div className="bg-white rounded-xl shadow p-5 mb-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-base font-semibold text-gray-700">近期趋势</h2>
            <select
              value={dailyDays}
              onChange={(e) => setDailyDays(Number(e.target.value))}
              className="text-sm border rounded px-2 py-1 text-gray-600"
            >
              <option value={7}>最近 7 天</option>
              <option value={14}>最近 14 天</option>
              <option value={30}>最近 30 天</option>
            </select>
          </div>
          <ResponsiveContainer width="100%" height={240}>
            <LineChart data={daily} margin={{ top: 4, right: 20, left: 0, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
              <XAxis dataKey="date" tick={{ fontSize: 11 }} />
              <YAxis tick={{ fontSize: 11 }} />
              <Tooltip />
              <Legend />
              <Line
                type="monotone"
                dataKey="messages"
                name="消息数"
                stroke="#6366f1"
                strokeWidth={2}
                dot={false}
              />
              <Line
                type="monotone"
                dataKey="new_users"
                name="新用户"
                stroke="#10b981"
                strokeWidth={2}
                dot={false}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>

        {/* 用户/群组管理 Tab */}
        <div className="bg-white rounded-xl shadow">
          <div className="flex border-b">
            {(["users", "groups"] as const).map((t) => (
              <button
                key={t}
                onClick={() => setTab(t)}
                className={`px-6 py-3 text-sm font-medium transition-colors ${
                  tab === t
                    ? "border-b-2 border-indigo-500 text-indigo-600"
                    : "text-gray-500 hover:text-gray-700"
                }`}
              >
                {t === "users" ? `👥 用户管理 (${users.length})` : `👫 群组管理 (${groups.length})`}
              </button>
            ))}
          </div>

          <div className="overflow-auto">
            {tab === "users" && (
              <table className="w-full text-sm">
                <thead className="bg-gray-50 text-gray-600">
                  <tr>
                    <th className="text-left px-4 py-3">昵称</th>
                    <th className="text-left px-4 py-3">手机号</th>
                    <th className="text-left px-4 py-3">注册时间</th>
                    <th className="text-left px-4 py-3">状态</th>
                    <th className="text-left px-4 py-3">操作</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {users.map((u) => (
                    <tr key={u.uuid} className="hover:bg-gray-50">
                      <td className="px-4 py-3 font-medium text-gray-800">{u.nickname}</td>
                      <td className="px-4 py-3 text-gray-600">{u.telephone}</td>
                      <td className="px-4 py-3 text-gray-500">
                        {new Date(u.created_at).toLocaleDateString()}
                      </td>
                      <td className="px-4 py-3">
                        {u.is_admin === 1 ? (
                          <span className="px-2 py-0.5 bg-indigo-100 text-indigo-700 rounded-full text-xs">管理员</span>
                        ) : u.status === 1 ? (
                          <span className="px-2 py-0.5 bg-red-100 text-red-700 rounded-full text-xs">已封禁</span>
                        ) : (
                          <span className="px-2 py-0.5 bg-green-100 text-green-700 rounded-full text-xs">正常</span>
                        )}
                      </td>
                      <td className="px-4 py-3">
                        {u.is_admin !== 1 && (
                          <button
                            onClick={() => handleBan(u.uuid, u.status)}
                            className={`text-xs px-3 py-1 rounded-lg ${
                              u.status === 1
                                ? "bg-green-100 text-green-700 hover:bg-green-200"
                                : "bg-red-100 text-red-700 hover:bg-red-200"
                            }`}
                          >
                            {u.status === 1 ? "解封" : "封禁"}
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}

            {tab === "groups" && (
              <table className="w-full text-sm">
                <thead className="bg-gray-50 text-gray-600">
                  <tr>
                    <th className="text-left px-4 py-3">群名称</th>
                    <th className="text-left px-4 py-3">成员数</th>
                    <th className="text-left px-4 py-3">创建时间</th>
                    <th className="text-left px-4 py-3">状态</th>
                    <th className="text-left px-4 py-3">操作</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {groups.map((g) => (
                    <tr key={g.uuid} className="hover:bg-gray-50">
                      <td className="px-4 py-3 font-medium text-gray-800">{g.name}</td>
                      <td className="px-4 py-3 text-gray-600">{g.member_cnt}</td>
                      <td className="px-4 py-3 text-gray-500">
                        {new Date(g.created_at).toLocaleDateString()}
                      </td>
                      <td className="px-4 py-3">
                        {g.status === 2 ? (
                          <span className="px-2 py-0.5 bg-gray-100 text-gray-500 rounded-full text-xs">已解散</span>
                        ) : (
                          <span className="px-2 py-0.5 bg-green-100 text-green-700 rounded-full text-xs">正常</span>
                        )}
                      </td>
                      <td className="px-4 py-3">
                        {g.status !== 2 && (
                          <button
                            onClick={() => handleDismissGroup(g.uuid)}
                            className="text-xs px-3 py-1 rounded-lg bg-red-100 text-red-700 hover:bg-red-200"
                          >
                            解散
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default Admin;
