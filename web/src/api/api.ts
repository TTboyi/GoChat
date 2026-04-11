import axios from "axios";
import { getToken, setToken, clearToken, getRefreshToken, setRefreshToken, clearRefreshToken } from "../utils/session";
import { API_BASE } from "../config";

// 这个文件相当于“前端访问后端的统一网关”：
// 1. 创建 axios 实例；
// 2. 在请求阶段自动带上 access token；
// 3. 在响应阶段统一处理 401 和 token 刷新；
// 4. 导出按业务域划分的接口方法，供页面/Hook 直接调用。
const api = axios.create({
  baseURL: API_BASE,
  timeout: 10000,
});

// 请求拦截器：给受保护接口自动补 Authorization 头。
api.interceptors.request.use((config: any) => {
  const token = getToken();
  if (token && config.headers) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// 响应拦截器：集中处理“access token 过期但 refresh token 还有效”的情况。
// 这里用 isRefreshing + refreshPromise 做并发收敛，避免多个 401 同时触发多次刷新请求。
let isRefreshing = false;
let refreshPromise: Promise<string | null> | null = null;

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;
    // 只处理受保护接口的 401（排除登录/注册/验证码/刷新等公开接口）
    if (
      error.response?.status === 401 &&
      !originalRequest._retry &&
      !originalRequest.url?.includes("/auth/refresh") &&
      !originalRequest.url?.includes("/login") &&
      !originalRequest.url?.includes("/register") &&
      !originalRequest.url?.includes("/captcha")
    ) {
      originalRequest._retry = true;

      // 没有 refresh token 时，说明无法自动续期，只能回登录页重新登录。
      const refreshToken = getRefreshToken();
      if (!refreshToken) {
        clearToken();
        window.location.href = "/";
        return Promise.reject(error);
      }

      // 多个请求同时 401 时，只允许第一个请求真正发起刷新，其它请求复用同一个 Promise。
      if (!isRefreshing) {
        isRefreshing = true;
        const accessToken = getToken();
        const refreshToken = getRefreshToken();
        refreshPromise = axios
          .post(`${API_BASE}/auth/refresh`, { access: accessToken, refresh: refreshToken })
          .then((res) => {
            const newToken = res.data?.token || res.data?.data?.token;
            const newRefresh = res.data?.refresh || res.data?.data?.refresh;
            if (newToken) {
              setToken(newToken);
              if (newRefresh) setRefreshToken(newRefresh);
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
        // 刷新成功后，用新 token 无感重试原请求。
        originalRequest.headers.Authorization = `Bearer ${newToken}`;
        return api(originalRequest);
      } else {
        // 刷新失败时，前端不能再假装自己已登录，必须清掉本地状态。
        clearToken();
        clearRefreshToken();
        window.location.href = "/";
        return Promise.reject(error);
      }
    }
    return Promise.reject(error);
  }
);

export default {
  // ================= 用户认证 =================
  // 这些方法刻意保持“薄封装”，让调用方一眼能看出它们对应哪个后端接口。
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

  // 退出群聊（作为普通成员）
  leaveGroup: (data: { groupUuid: string }) =>
    api.post("/group/quit", { groupId: data.groupUuid }),

  // 解散群聊（群主权限）
  dismissGroup: (data: { groupId: string }) =>
    api.post("/group/dismiss", data),

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
  // WebRTC Hook 会在真正发起/接听通话前调用它，动态获取 ICE server 配置。
  getTurnCredentials: () => api.get("/turn/credentials"),
};
