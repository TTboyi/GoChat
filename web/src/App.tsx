import React from "react";
import { BrowserRouter as Router, Routes, Route } from "react-router-dom";
import { AuthProvider } from "./context/AuthContext";
import ProtectedRoute from "./components/ProtectedRoute";

import Login from "./pages/Login";
import Register from "./pages/Register";
import CaptchaLogin from "./pages/CaptchaLogin";
import Chat from "./pages/Chat";
import Profile from "./pages/Profile";

// App 只关心“路由层面的页面切换”。
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
          </Routes>
        </AuthProvider>
      </Router>
  );
}

export default App;
