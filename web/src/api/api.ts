import axios from "axios";
import { getToken } from "../utils/session"; 

// 创建 axios 实例
const api = axios.create({
  baseURL: "http://localhost:8000", // 后端服务地址
  timeout: 10000,
});

// 请求拦截器：带上 JWT
api.interceptors.request.use((config) => {
  const token = getToken();
  console.log("👉 请求URL:", config.url, "带token:", token ? token.slice(0, 15) + "..." : "无token");
  if (token && config.headers) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

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
  createGroup: (data: { name: string; notice?: string; avatar?: string; addMode?: number; ownerId: string }) =>
    api.post("/group/create", data),

    // 查询我创建的群
    loadMyGroup: () => api.get("/group/loadMyGroup"),

    // 查询我加入的群（如果你在后端定义了 /contact/joinedGroups）
    loadMyJoinedGroup: () => api.get("/contact/joinedGroups"),
  
    // 获取群成员
    getGroupMembers: (groupUuid: string) =>
      api.get("/group/getGroupMemberList", { params: { groupUuid } }),
  
    // 直接加入群聊
    enterGroup: (data: { groupId: string; message?: string }) => {
      const formData = new FormData();
      formData.append("groupId", data.groupId); // ✅ 对应 EnterGroupDirectly 的 c.PostForm("groupId")
      if (data.message) formData.append("message", data.message);
      return api.post("/group/enterGroupDirectly", formData);
    },
  
    // 退出群聊
    leaveGroup: (data: { groupUuid: string }) => {
      const formData = new FormData();
      formData.append("groupUuid", data.groupUuid);
      return api.post("/group/leaveGroup", formData);
    },
  
    // 移除群成员（群主权限）
    removeMember: (data: { groupUuid: string; targetUserId: string }) => {
      const formData = new FormData();
      formData.append("groupUuid", data.groupUuid);
      formData.append("targetUserId", data.targetUserId);
      return api.post("/group/removeGroupMember", formData);
    },
  
    // 解散群聊（群主权限）
    dismissGroup: (data: { groupUuid: string }) => {
      const formData = new FormData();
      formData.append("groupUuid", data.groupUuid);
      return api.post("/group/dismissGroup", formData);
    },
  
    // 获取群聊消息列表
    getGroupMessageList: (data: { groupId: string; limit?: number }) =>
      api.post("/message/groupList", data),

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
  getMessageList: (data: { targetId: string, limit: number}) =>
    api.post("/message/list", data),
  
  uploadAvatar: async(formData: FormData) =>{
    const res =  await api.post("/message/uploadAvatar", formData, {
      headers: { "Content-Type": "multipart/form-data" },
    });
        // ✅ 与后端一致：返回 { message, url }
    const relativeUrl = res.data?.url;
    const avatarUrl = relativeUrl ? `http://localhost:8000${relativeUrl}` : "";
  return { avatarUrl };
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
};
