// SessionItem 描述左侧会话列表中的一个条目。
export interface SessionItem {
  id: string;
  name: string;
  avatar?: string;
  type: "user" | "group";
}

// Message 是前端渲染消息气泡时使用的统一结构。
// 注意 createdAt 允许 number|string，是为了兼容接口返回和本地缓存两种来源。
export interface Message {
  uuid?: string;
  sendId: string;
  receiveId: string;
  content: string;
  type: number;
  createdAt?: number | string;
  sendName?: string;
  sendAvatar?: string;
  url?: string;
  fileName?: string;
  fileType?: string;
  fileSize?: string;
  isRecalled?: boolean;
  readAt?: string | null; // ISO string or null
}

// GroupInfo 保存群聊的展示信息和管理信息。
export interface GroupInfo {
  uuid: string;
  name: string;
  notice?: string;
  add_mode?: number;
  owner_id?: string;
  member_cnt?: number;
  avatar?: string;
}

// GroupMember 是群成员弹窗需要的最小信息集。
export type GroupMember = {
  uuid: string;
  nickname?: string;
  avatar?: string;
};

// CallState 描述当前通话状态机。
export type CallState = {
  callId: string | null;
  peerId: string | null;
  status: "idle" | "ringing" | "in-call";
  callType: "audio" | "video" | null;
  isCaller: boolean;
};
