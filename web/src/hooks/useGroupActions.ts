// src/hooks/useGroupActions.ts
import axios from "axios";
import { API_BASE } from "../config";

export const leaveGroup = async (groupId: string) => {
  try {
    const res = await axios.post(
      `${API_BASE}/group/leaveGroup`,
      new URLSearchParams({ groupUuid: groupId }),
      { headers: { "Content-Type": "application/x-www-form-urlencoded" } }
    );
    alert(res.data.message || "退出成功");
  } catch (error) {
    console.error("退出群聊失败", error);
    alert("退出失败");
  }
};

export const dismissGroup = async (groupId: string) => {
  try {
    const res = await axios.post(
      `${API_BASE}/group/dismissGroup`,
      new URLSearchParams({ groupUuid: groupId }),
      { headers: { "Content-Type": "application/x-www-form-urlencoded" } }
    );
    alert(res.data.message || "解散成功");
  } catch (error) {
    console.error("解散群聊失败", error);
    alert("解散失败");
  }
};
