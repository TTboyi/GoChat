// ============================================================
// 文件：web/src/App.tsx
// 作用：前端路由配置的"总入口"，负责把 URL 路径映射到对应页面组件。
//
// React Router 核心概念：
//   BrowserRouter：使用 HTML5 History API（history.pushState）管理 URL，
//     用户感觉在浏览不同"页面"，但实际上没有真正刷新（单页应用 SPA）。
//   Routes / Route：声明式路由表，匹配当前 URL 并渲染对应组件。
//   Navigate：编程式跳转（相当于"强制重定向"）。
//
// AuthProvider 包裹：
//   把 AuthProvider 放在 Router 内部，使得路由钩子（useNavigate 等）
//   在 AuthProvider 内部也可以使用。
//
// 路由分类：
//   公开路由（/、/register、/captcha-login）：无需登录即可访问
//   受保护路由（/chat、/profile）：通过 ProtectedRoute 检查 token
//   管理员路由（/admin）：通过 AdminRoute 额外检查 is_admin 标志
// ============================================================
import React from "react";
import { BrowserRouter as Router, Routes, Route, Navigate } from "react-router-dom";
import { AuthProvider, useAuth } from "./context/AuthContext";
import ProtectedRoute from "./components/ProtectedRoute";
import { getToken } from "./utils/session";

import Login from "./pages/Login";
import Register from "./pages/Register";
import CaptchaLogin from "./pages/CaptchaLogin";
import Chat from "./pages/Chat";
import Profile from "./pages/Profile";
import Admin from "./pages/Admin";

// AdminRoute 在 ProtectedRoute 基础上额外校验 is_admin 标志。
const AdminRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const token = getToken();
  const { user } = useAuth();
  if (!token) return <Navigate to="/" replace />;
  // user 未加载完时先放行（避免闪屏跳转），加载后再校验
  if (user && !user.is_admin) return <Navigate to="/chat" replace />;
  return <>{children}</>;
};

// App 只关心"路由层面的页面切换"。
// 真正的业务逻辑会继续下沉到各个 page / component / hook 中。
function App() {
  return (
      <Router>
        <AuthProvider>
          <Routes>
            {/* 公开页面：未登录也能访问 */}
            <Route path="/" element={<Login />} />
            <Route path="/register" element={<Register />} />
            <Route path="/captcha-login" element={<CaptchaLogin />} />

            {/* 受保护页面：必须先通过 AuthContext 确认已登录 */}
            <Route
              path="/chat"
              element={
                <ProtectedRoute>
                  <Chat />
                </ProtectedRoute>
              }
            />
            <Route
              path="/profile"
              element={
                <ProtectedRoute>
                  <Profile />
                </ProtectedRoute>
              }
            />

            {/* 管理员专属页面 */}
            <Route
              path="/admin"
              element={
                <AdminRoute>
                  <Admin />
                </AdminRoute>
              }
            />
          </Routes>
        </AuthProvider>
      </Router>
  );
}

export default App;
