// ============================================================
// 文件：web/src/context/AuthContext.tsx
// 作用：全局认证状态的"单一事实来源"（Single Source of Truth）。
//
// 什么是 React Context（上下文）？
//   Props 传递有个问题：如果 A 组件里有用户信息，而 D 组件（孙子辈）需要用，
//   就得 A→B→C→D 一层层传递（"prop drilling"，钻探问题）。
//   Context 解决了这个问题：把状态放进"上下文容器"，任何后代组件
//   都可以直接调用 useContext 取数据，不用关心中间层。
//
// AuthProvider / useAuth 的分工：
//   AuthProvider：包裹在所有页面外层，提供 user、login、logout、refreshUser
//   useAuth Hook：在任何组件里取出认证状态，一行代码：const { user } = useAuth()
//
// 初始化流程（页面刷新时）：
//   1. 读 sessionStorage 里的 token
//   2. 如果有，调 /user/info 恢复用户资料（因为内存里的 user 状态在刷新后清空了）
//   3. 如果失败（token 已失效），清掉本地 token，用户需重新登录
//   loading 标志防止在用户信息加载完成前就渲染受保护页面（避免闪屏跳转）
// ============================================================
import React, { createContext, useContext, useEffect, useState } from "react";
import api from "../api/api";

import { getToken, setToken, clearToken } from "../utils/session";

// UserInfo 描述前端真正关心的“当前登录用户最小画像”。
interface UserInfo {
  uuid: string;
  nickname: string;
  email: string;
  avatar: string;
  signature?: string;
  is_admin?: boolean;
}

// AuthContextType 定义了页面可消费的认证能力：
// - user / loading：当前状态；
// - login / logout：认证动作；
// - refreshUser：资料变更后的同步入口。
interface AuthContextType {
  user: UserInfo | null;
  loading: boolean;
  login: (username: string, password: string) => Promise<boolean>;
  logout: () => Promise<void>;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

// AuthProvider 是整个前端认证状态的“单一事实来源”。
// 页面组件不要自己直接维护“我是不是登录了”，而是统一从这里读取。
export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<UserInfo | null>(null);
  const [loading, setLoading] = useState(true);

  /**
   * 初始化阶段：
   * 1. 先从 sessionStorage 取当前标签页的 token；
   * 2. 如果有 token，就请求后端恢复用户资料；
   * 3. 如果失败，说明 token 无效或已过期，清掉本地状态。
   */
  useEffect(() => {
    const token = getToken();
    if (!token) {
      setLoading(false);
      return;
    }

    const fetchUser = async () => {
      try {
        const res = await api.getUserInfo();
        const data = res.data?.data || res.data;
        setUser(data);
      } catch (err) {
        console.error("加载用户信息失败:", err);
        clearToken();
      } finally {
        setLoading(false);
      }
    };

    setTimeout(fetchUser, 100);
  }, []);

  /**
   * login 只负责“完成登录动作并把用户状态装进上下文”。
   * 具体接口细节（请求地址、拦截器、刷新 token）都交给 api.ts。
   */
  const login = async (nickname: string, password: string) => {
  try {
    const res = await api.login({ nickname, password });
    console.log("🔑 登录响应:", res.data);

    const token = res.data?.token; // ✅ 后端直接返回 token
    if (!token) {
      console.error("❌ 未获取到 token:", res.data);
      return false;
    }

    // ✅ 存入 sessionStorage（每个标签页独立）
    setToken(token);

    // 测试打印：确保存成功
    console.log("✅ token 已存入 sessionStorage:", sessionStorage);

    const userRes = await api.getUserInfo();
    const userData = userRes.data?.data || userRes.data;
    setUser(userData);
    return true;
  } catch (err) {
    console.error("登录失败:", err);
  }
  return false;
};
  // logout 既通知后端失效 token，也把前端的内存态/存储态一起清空。
  const logout = async () => {
    try {
      await api.logout();
    } catch (_) {}
    clearToken();
    setUser(null);
    window.location.href = "/";
  };

  // refreshUser 常用于“个人资料更新成功后，把最新头像/昵称拉回来”。
  const refreshUser = async () => {
    try {
      const res = await api.getUserInfo();
      const data = res.data?.data || res.data;
      setUser(data);
    } catch (err) {
      console.error("刷新用户信息失败:", err);
    }
  };

  return (
    <AuthContext.Provider value={{ user, loading, login, logout, refreshUser }}>
      {children}
    </AuthContext.Provider>
  );
};

// useAuth 让页面以 Hook 的方式消费认证上下文，避免层层透传 props。
export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) throw new Error("useAuth 必须在 <AuthProvider> 内使用");
  return context;
};
