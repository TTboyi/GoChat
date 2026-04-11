// socket.ts 封装浏览器原生 WebSocket，补上了三个聊天应用常见能力：
// 1. 断线自动重连；
// 2. 群聊订阅状态恢复；
// 3. 统一的消息发送辅助函数。

import { getToken } from "../utils/session";
import { WS_BASE } from "../config";

// ChatMessage 是“普通聊天消息”的发送载荷。
export interface ChatMessage {
  type: number;
  content: string;
  receiveId: string;
  url?: string;
  fileName?: string;
  fileType?: string;
  fileSize?: string;
}

// IncomingMessage 是前端消费服务端推送时的宽松类型。
// 它既容纳普通聊天消息，也容纳系统事件和控制消息。
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

// ChatWebSocket 把“连接生命周期管理”封装成一个类，
// 这样页面层只需要关心 onMessage/onOpen/onClose，而不用反复处理底层细节。
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
    // 优先读取最新 token，避免刷新 token 后还沿用旧值重连。
    const tk = getToken() || this.token;
    const wsUrl = `${WS_BASE}/wss?token=${tk}`;
    this.ws = new WebSocket(wsUrl);

    // 重连时把“已经订阅过的群”重新放回待发送队列，
    // 等 onopen 之后再补发 join_group，恢复实时推送能力。
    this.subscribedGroups.forEach(id => this.pendingGroups.add(id));
    this.subscribedGroups.clear();

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

  // setGroups 记录当前用户需要订阅的所有群。
  // 如果连接已经建立，会马上 flush；否则等 onopen 后统一补发。
  setGroups(groupIds: string[]) {
    groupIds.forEach(id => this.pendingGroups.add(id)); // 记录需要订阅的群
    this.flushGroupSubscriptions();
  }

  // flushGroupSubscriptions 只在连接处于 OPEN 时发送 join_group，
  // 避免在尚未建连成功时调用 send 导致消息丢失。
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
    // 所有上行消息最终都会走这里，因此这里也是观察“WS 当前是否可用”的最佳位置。
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

// sendTextMessage / sendFileMessage / call* 系列函数，
// 目的是让页面层用“业务语义”发消息，而不是手写 JSON。
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




// 音视频信令发送
export const callInvite = (socket: ChatWebSocket, targetUserId: string, callType: "audio" | "video", callId: string) => {
  socket.send({
    action: "call_invite",
    receiveId: targetUserId,
    callType,
    callId,
  });
};

export const callAnswer = (socket: ChatWebSocket, targetUserId: string, callId: string, accept: boolean, sdp?: any) => {
  socket.send({
    action: "call_answer",
    receiveId: targetUserId,
    callId,
    accept,
    content: sdp ? JSON.stringify(sdp) : "",
  });
};

export const callCandidate = (socket: ChatWebSocket, targetUserId: string, callId: string, candidate: any) => {
  socket.send({
    action: "call_candidate",
    receiveId: targetUserId,
    callId,
    content: JSON.stringify(candidate),
  });
};

export const callEnd = (socket: ChatWebSocket, targetUserId: string, callId: string) => {
  socket.send({
    action: "call_end",
    receiveId: targetUserId,
    callId,
  });
};
