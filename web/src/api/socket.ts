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
  uuid: string;
  type: number;
  content?: string;
  url?: string;
  sendId: string;
  receiveId: string;
  createdAt: number;
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

  constructor({ token, onMessage, onOpen, onClose, reconnect = true }: ChatWebSocketProps) {
    this.token = token;
    this.onMessage = onMessage;
    this.onOpen = onOpen;
    this.onClose = onClose;
    this.reconnect = reconnect;
    this.connect();
  }

  private connect() {
    // ✅ 每个标签页用自己 session 内的 token
    const tk = getToken() || this.token;
    const wsUrl = `ws://localhost:8000/wss?token=${tk}`;
    this.ws = new WebSocket(wsUrl);

    this.ws.onopen = () => {
      console.log("✅ WebSocket 已连接");
      this.onOpen?.();
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
      console.warn("❌ WebSocket 已关闭");
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
    console.log("🔁 正在尝试重连...");
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = undefined;
      this.connect();
    }, 3000);
  }

  send(msg: ChatMessage) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      console.warn("WebSocket未连接，消息未发送:", msg);
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

export const sendTextMessage = (socket: ChatWebSocket, content: string, receiveId: string) => {
  const msg: ChatMessage = { type: 0, content, receiveId };
  socket.send(msg);
};

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
