import { useEffect, useState } from "react";
import api from "../api/api";
import { getToken } from "../utils/session";

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
        localStorage.removeItem("token");
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
      if (token) {
        localStorage.setItem("token", token);
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
    localStorage.removeItem("token");
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
