import React, { createContext, useContext, useEffect, useState } from "react";
import api from "../api/api";

import { getToken, setToken, clearToken } from "../utils/session";

interface UserInfo {
  uuid: string;
  nickname: string;
  email: string;
  avatar: string;
  signature?: string;
  is_admin?: boolean;
}

interface AuthContextType {
  user: UserInfo | null;
  loading: boolean;
  login: (username: string, password: string) => Promise<boolean>;
  logout: () => Promise<void>;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<UserInfo | null>(null);
  const [loading, setLoading] = useState(true);

  /** ✅ 初始化：每个标签页独立加载自己的用户 */
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

  /** ✅ 登录逻辑 */
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
  /** ✅ 登出逻辑 */
  const logout = async () => {
    try {
      await api.logout();
    } catch (_) {}
    clearToken();
    setUser(null);
    window.location.href = "/";
  };

  /** ✅ 刷新用户信息（修改资料后） */
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

/** ✅ 导出 Hook */
export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) throw new Error("useAuth 必须在 <AuthProvider> 内使用");
  return context;
};
