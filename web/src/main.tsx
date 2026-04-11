// src/main.tsx
import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import { AuthProvider } from "./context/AuthContext";
import "./index.css";
// src/main.tsx 或 index.tsx
import { getTabId } from "./utils/session";

// 入口文件只做三件事：
// 1. 提前生成当前标签页的 tabId，保证 token 能做“按标签页隔离”的存储；
// 2. 挂载全局 AuthProvider，让任意页面都能访问用户状态；
// 3. 把 App 渲染到 root 节点。
//
// 这一步之所以要最先调用 getTabId，是因为后面的登录、鉴权请求、WebSocket 建连
// 都会依赖这个 tabId 来区分“当前浏览器标签页”的身份。
getTabId();


ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <AuthProvider>
      <App />   {/* 之后任意页面里的 useAuth 都能拿到同一个认证上下文。 */}
    </AuthProvider>
  </React.StrictMode>
);
