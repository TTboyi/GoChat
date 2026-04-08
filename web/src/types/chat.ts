export interface SessionItem {
  id: string;
  name: string;
  avatar?: string;
  type: "user" | "group";
}

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

export interface GroupInfo {
  uuid: string;
  name: string;
  notice?: string;
  add_mode?: number;
  owner_id?: string;
  member_cnt?: number;
  avatar?: string;
}

export type GroupMember = {
  uuid: string;
  nickname?: string;
  avatar?: string;
};

export type CallState = {
  callId: string | null;
  peerId: string | null;
  status: "idle" | "ringing" | "in-call";
  callType: "audio" | "video" | null;
  isCaller: boolean;
};
