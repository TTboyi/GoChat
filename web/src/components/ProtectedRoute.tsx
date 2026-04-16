// ============================================================
// 文件：web/src/components/ProtectedRoute.tsx
// 作用：路由守卫组件，防止未登录用户直接访问 /chat 等受保护页面。
//
// 工作方式：
//   1. 从 sessionStorage 取 token（调用 getToken()）
//   2. 如果没有 token，立刻重定向到登录页（/）
//   3. 如果有 token，渲染子组件（children）
//
// 注意：这里只检查 token 是否存在，不验证 token 是否有效（过期等）。
// token 真正的有效性验证发生在 API 请求的响应拦截器里（api.ts 的 401 处理）。
// ============================================================
import React from "react";
import { Navigate } from "react-router-dom";
import { useAuth } from "../context/AuthContext";
import { getToken } from "../utils/session"; 

interface ProtectedRouteProps {
  children: React.ReactNode;
}

const ProtectedRoute: React.FC<ProtectedRouteProps> = ({ children }) => {
  const token =  getToken();  

  if (!token) {
    return <Navigate to="/" replace />;
  }

  return <>{children}</>;
};

export default ProtectedRoute;
