// ✅ socket.ts – 自动群订阅 + 重连恢复 + 实时群消息

import { getToken } from "../utils/session";

export interface ChatMessage {
  type: number;
  content: string;
  receiveId: string;
  url?: string;
  fileName?: string;
  fileType?: string;
  fileSize?: string;
}

export interface IncomingMessage {
  uuid?: string;
  type: number;
  content?: string;
  url?: string;
  sendId?: string;
  receiveId?: string;
  createdAt?: number;
  // ✅ 系统消息字段（解散群）
  action?: string;    // e.g. "group_dismiss"
  groupId?: string;
  message?: string;
}


interface ChatWebSocketProps {
  token: string;
  onMessage: (msg: IncomingMessage) => void;
  onOpen?: () => void;
  onClose?: () => void;
  reconnect?: boolean;
}

export class ChatWebSocket {
  private ws: WebSocket | null = null;
  private token: string;
  private onMessage: (msg: IncomingMessage) => void;
  private onOpen?: () => void;
  private onClose?: () => void;
  private reconnect: boolean;
  private reconnectTimer?: ReturnType<typeof setTimeout>;
  private subscribedGroups: Set<string> = new Set(); // ✅ 已订阅群ID缓存
  private pendingGroups: Set<string> = new Set(); // ✅ 连接建立后自动订阅用

  constructor({ token, onMessage, onOpen, onClose, reconnect = true }: ChatWebSocketProps) {
    this.token = token;
    this.onMessage = onMessage;
    this.onOpen = onOpen;
    this.onClose = onClose;
    this.reconnect = reconnect;
    this.connect();
  }

  private connect() {
    const tk = getToken() || this.token;
    const wsUrl = `ws://localhost:8000/wss?token=${tk}`;
    this.ws = new WebSocket(wsUrl);

    this.ws.onopen = () => {
      console.log("✅ WebSocket 连接成功");
      this.onOpen?.();
      this.flushGroupSubscriptions(); // ✅ 发送订阅
    };

    this.ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        this.onMessage(data);
      } catch {
        console.warn("收到非JSON消息:", event.data);
      }
    };

    this.ws.onclose = () => {
      console.warn("❌ WebSocket 连接关闭");
      this.onClose?.();
      if (this.reconnect) this.scheduleReconnect();
    };

    this.ws.onerror = (err) => {
      console.error("⚠️ WebSocket 出错:", err);
      this.ws?.close();
    };
  }

  private scheduleReconnect() {
    if (this.reconnectTimer) return;
    console.log("🔁 正在重连...");
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = undefined;
      this.connect();
    }, 3000);
  }

  // ✅ 设置需要订阅的群
  setGroups(groupIds: string[]) {
    groupIds.forEach(id => this.pendingGroups.add(id)); // 记录需要订阅的群
    this.flushGroupSubscriptions();
  }

  // ✅ flush 群订阅，连接成功后才发送 join_group
  private flushGroupSubscriptions() {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;

    this.pendingGroups.forEach(groupId => {
      if (!this.subscribedGroups.has(groupId)) {
        this.ws!.send(JSON.stringify({ action: "join_group", groupId }));
        this.subscribedGroups.add(groupId);
        console.log(`✅ 已订阅群 ${groupId}`);
      }
    });
    this.pendingGroups.clear();
  }

  send(msg: any) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      console.warn("⚠️ WebSocket 未连接，消息丢失:", msg);
      return;
    }
    this.ws.send(JSON.stringify(msg));
  }

  close() {
    this.reconnect = false;
    if (this.ws?.readyState === WebSocket.OPEN) this.ws.close();
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
  }
}

// ✅ 发送文本消息
export const sendTextMessage = (socket: ChatWebSocket, content: string, receiveId: string) => {
  const msg: ChatMessage = { type: 0, content, receiveId };
  socket.send(msg);
};

// ✅ 发送文件消息
export const sendFileMessage = (
  socket: ChatWebSocket,
  fileUrl: string,
  receiveId: string,
  fileName: string,
  fileType: string,
  fileSize: string
) => {
  const msg: ChatMessage = {
    type: 1,
    content: fileUrl,
    receiveId,
    url: fileUrl,
    fileName,
    fileType,
    fileSize,
  };
  socket.send(msg);
};
