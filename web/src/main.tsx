// src/main.tsx
import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import { AuthProvider } from "./context/AuthContext";
import "./index.css";
// src/main.tsx 或 index.tsx
import { getTabId } from "./utils/session";

// ✅ 页面一加载就生成独立 tabId
getTabId();


ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <AuthProvider>
      <App />   {/* 现在 App 里的 Login/Chat/Profile 用 useAuth 都安全了 */}
    </AuthProvider>
  </React.StrictMode>
);
