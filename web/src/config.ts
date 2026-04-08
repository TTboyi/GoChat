// 生产环境通过 .env.production 设置 VITE_API_BASE 和 VITE_WS_BASE
export const API_BASE: string =
  (import.meta.env.VITE_API_BASE as string) || "http://localhost:8000";

export const WS_BASE: string =
  (import.meta.env.VITE_WS_BASE as string) || "ws://localhost:8000";
