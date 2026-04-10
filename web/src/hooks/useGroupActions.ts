// src/hooks/useGroupActions.ts
import api from "../api/api";

export const leaveGroup = async (groupId: string): Promise<boolean> => {
  try {
    await api.leaveGroup({ groupUuid: groupId });
    return true;
  } catch (error) {
    console.error("退出群聊失败", error);
    return false;
  }
};

export const dismissGroup = async (groupId: string): Promise<boolean> => {
  try {
    await api.dismissGroup({ groupId });
    return true;
  } catch (error) {
    console.error("解散群聊失败", error);
    return false;
  }
};
