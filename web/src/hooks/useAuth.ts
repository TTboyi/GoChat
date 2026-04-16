// ============================================================
// 文件：web/src/hooks/useAuth.ts
// 作用：一个独立的认证 Hook（与 AuthContext.tsx 中的 useAuth 并存）。
//       本文件的 useAuth 主要供 Login 页面使用（不依赖 AuthContext，独立管理状态）。
//
// 与 AuthContext 的 useAuth 的区别：
//   AuthContext.tsx 的 useAuth：消费全局 Context，适合在 Chat 等主界面使用
//   本文件的 useAuth：独立 Hook，维护自己的 user/loading state
//   两者逻辑相似，但本文件额外暴露了 setUser（供外部直接更新用户状态）
// ============================================================
import { useEffect, useState } from "react";
import api from "../api/api";
import { getToken, setToken, clearToken, setRefreshToken, clearRefreshToken } from "../utils/session";

interface UserInfo {
  uuid: string;
  nickname: string;
  avatar: string;
  signature?: string;
  telephone?: string;
  is_admin?: boolean;
}

/**
 * useAuth
 * 管理登录状态、用户信息、登出逻辑
 */
export const useAuth = () => {
  const [user, setUser] = useState<UserInfo | null>(null);
  const [loading, setLoading] = useState(true);

  // 初始化：加载用户信息
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
        console.error("获取用户信息失败:", err);
        clearToken();
        clearRefreshToken();
      } finally {
        setLoading(false);
      }
    };

    fetchUser();
  }, []);

  // 登录
  const login = async (nickname: string, password: string) => {
    try {
      const res = await api.login({ nickname, password });
      const token = res.data?.token || res.data?.data?.token;
      const refresh = res.data?.refresh || res.data?.data?.refresh;
      if (token) {
        setToken(token);
        if (refresh) setRefreshToken(refresh);
        // 从登录响应中获取用户信息，避免额外的API调用
        const userRes = await api.getUserInfo();
        const userData = userRes.data?.data || userRes.data;
        setUser(userData);
        return true;
      }
    } catch (err) {
      console.error("登录失败:", err);
    }
    return false;
  };

  // 登出
  const logout = async () => {
    try {
      await api.logout();
    } catch (_) {}
    clearToken();
    clearRefreshToken();
    setUser(null);
    window.location.href = "/";
  };

  return {
    user,
    loading,
    isLogin: !!user,
    login,
    logout,
    setUser,
  };
};
