import axios from "axios";
import { getToken, setToken, clearToken } from "../utils/session";
import { API_BASE } from "../config";

// 创建 axios 实例
const api = axios.create({
  baseURL: API_BASE,
  timeout: 10000,
});

// 请求拦截器：带上 JWT
api.interceptors.request.use((config: any) => {
  const token = getToken();
  if (token && config.headers) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// ✅ 响应拦截器：Token 过期时自动刷新（避免视频通话中途因 401 失败）
let isRefreshing = false;
let refreshPromise: Promise<string | null> | null = null;

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;
    // 只处理 401，且只重试一次，且不是 refresh 请求本身
    if (
      error.response?.status === 401 &&
      !originalRequest._retry &&
      !originalRequest.url?.includes("/auth/refresh")
    ) {
      originalRequest._retry = true;

      // 防止并发刷新（多个请求同时 401 时只刷新一次）
      if (!isRefreshing) {
        isRefreshing = true;
        refreshPromise = axios
          .post(`${API_BASE}/auth/refresh`, {}, { withCredentials: true })
          .then((res) => {
            const newToken = res.data?.token || res.data?.data?.token;
            if (newToken) {
              setToken(newToken);
              return newToken;
            }
            return null;
          })
          .catch(() => null)
          .finally(() => {
            isRefreshing = false;
            refreshPromise = null;
          });
      }

      const newToken = await refreshPromise;
      if (newToken) {
        // 用新 token 重试原请求
        originalRequest.headers.Authorization = `Bearer ${newToken}`;
        return api(originalRequest);
      } else {
        // 刷新失败 → 清除 token，跳回登录页
        clearToken();
        window.location.href = "/";
        return Promise.reject(error);
      }
    }
    return Promise.reject(error);
  }
);

export default {
  // ================= 用户认证 =================
  register: (data: { nickname: string; password: string }) =>
    api.post("/register", data),

  login: (data: { nickname: string; password: string }) =>
    api.post("/login", data),

  refreshToken: () => api.post("/auth/refresh"),

  logout: () => api.post("/auth/logout"),

  // ================= 邮箱验证码登录 =================
  sendEmailCaptcha: (data: { email: string }) =>
    api.post("/captcha/send_email", data),

  emailCaptchaLogin: (data: { email: string; code: string }) =>
    api.post("/captcha/login_email", data),

  // ================= 用户信息 =================
  getUserInfo: () => api.get("/api/user/info"),

  updateUser: (data: any) => api.post("/user/update", data),

  // ================= 群聊 =================
  createGroup: (data: {
    name: string;
    notice?: string;
    avatar?: string;
    addMode?: number;
    ownerId: string;
  }) => api.post("/group/create", data),

  // 查询我创建的群
  loadMyGroup: () => api.get("/group/loadMyGroup"),

  // 查询我加入的群
  loadMyJoinedGroup: () => api.get("/contact/joinedGroups"),

  // 获取群成员
  getGroupMembers: (groupUuid: string) =>
    api.get("/group/members", { params: { groupUuid: groupUuid } }),

  // 直接加入群聊
  enterGroup: (data: { groupId: string; message?: string }) => {
    const formData = new FormData();
    formData.append("groupId", data.groupId);
    if (data.message) formData.append("message", data.message);
    return api.post("/group/enter", formData);
  },

  // 退出群聊
  leaveGroup: (data: { groupUuid: string }) => {
    const formData = new FormData();
    formData.append("groupId", data.groupUuid);
    return api.post("/group/leave", formData);
  },

  // 移除群成员（群主权限）
  removeMember: (data: { groupUuid: string; targetUserId: string }) => {
    const formData = new FormData();
    formData.append("groupUuid", data.groupUuid);
    formData.append("targetUserId", data.targetUserId);
    return api.post("/group/removeMember", formData);
  },

  // 群聊详情与管理
  getGroupInfo: (groupId: string) =>
    api.get("/group/info", { params: { groupId } }),

  updateGroupName: (data: { groupId: string; name: string }) => {
    const formData = new FormData();
    formData.append("groupUuid", data.groupId);
    formData.append("name", data.name);
    return api.post("/group/updateName", formData);
  },

  updateGroupNotice: (data: { groupId: string; notice: string }) => {
    const formData = new FormData();
    formData.append("groupUuid", data.groupId);
    formData.append("notice", data.notice);
    return api.post("/group/updateNotice", formData);
  },

  updateGroupAvatar: (data: { groupUuid: string; avatar: string }) =>
    api.post("/group/updateAvatar", data),

  quitGroup: (data: { groupId: string; userId: string }) =>
    api.post("/group/quit", data),

  dismissGroup: (data: { groupId: string }) =>
    api.post("/group/dismiss", data),

  // ================= 联系人 =================
  applyContact: (data: { target: string; message: string }) =>
    api.post("/contact/apply", data),
  getContactList: () => api.get("/contact/list"),
  deleteContact: (data: { userId: string }) =>
    api.post("/contact/delete", data),
  blackContact: (data: { userId: string }) =>
    api.post("/contact/black", data),
  unblackContact: (data: { userId: string }) =>
    api.post("/contact/unblack", data),

  getNewContactList: () => api.get("/contact/newList"),
  handleContactApply: (data: { applyUuid: string; approve: boolean }) =>
    api.post("/contact/handle", data),

  // ================= 会话 =================
  openSession: (data: { targetId: string; type: "user" | "group" }) =>
    api.post("/session/open", data),
  getUserSessionList: () => api.get("/session/userList"),
  getGroupSessionList: () => api.get("/session/groupList"),
  deleteSession: (data: { sessionId: string }) =>
    api.post("/session/delete", data),

  // ================= 消息 =================
  getMessageList: (data: { targetId: string; limit?: number; beforeTime?: number }) =>
    api.post("/message/list", data),

  getGroupMessageList: (data: { groupId: string; limit?: number; beforeTime?: number }) =>
    api.post("/message/groupList", data),

  recallMessage: (data: { msgId: string; receiveId: string }) =>
    api.post("/message/recall", data),

  markMessagesRead: (data: { senderId: string }) =>
    api.post("/message/markRead", data),

  // ✅ 删除好友时清除双方的聊天记录（后端物理删除）
  clearConversation: (data: { targetId: string }) =>
    api.post("/message/clearConversation", data),

  uploadAvatar: async (formData: FormData) => {
    const res = await api.post("/message/uploadAvatar", formData, {
      headers: { "Content-Type": "multipart/form-data" },
    });
    const relativeUrl = res.data?.url;
    const avatarUrl = relativeUrl ? `${API_BASE}${relativeUrl}` : "";
    return { avatarUrl, relativeUrl };
  },

  uploadImage: async (formData: FormData) => {
    const res = await api.post("/message/uploadImage", formData, {
      headers: { "Content-Type": "multipart/form-data" },
    });
    const relativeUrl = res.data?.url;
    const imageUrl = relativeUrl ? `${API_BASE}${relativeUrl}` : "";
    return { imageUrl, relativeUrl };
  },

  uploadFile: (formData: FormData) =>
    api.post("/message/uploadFile", formData, {
      headers: { "Content-Type": "multipart/form-data" },
    }),

  // ================= 管理员 =================
  getAllUsers: () => api.get("/admin/users"),
  banUser: (id: string, status: boolean) =>
    api.put(`/admin/users/${id}/ban`, { status }),
  getAllGroups: () => api.get("/admin/groups"),
  adminDismissGroup: (id: string) => api.delete(`/admin/groups/${id}`),
  getSystemStats: () => api.get("/admin/stats"),

  // ================= TURN 服务器动态凭证 =================
  getTurnCredentials: () => api.get("/turn/credentials"),
};
