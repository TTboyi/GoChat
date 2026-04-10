// src/hooks/useContactActions.ts
import api from "../api/api";

// 删除联系人
export const deleteContact = async (contactId: string): Promise<boolean> => {
  try {
    await api.deleteContact({ userId: contactId });
    return true;
  } catch (err) {
    console.error("删除联系人失败", err);
    return false;
  }
};

// 拉黑联系人
export const blackContact = async (contactId: string): Promise<boolean> => {
  try {
    await api.blackContact({ userId: contactId });
    return true;
  } catch (err) {
    console.error("拉黑联系人失败", err);
    return false;
  }
};

// 取消拉黑
export const cancelBlackContact = async (contactId: string): Promise<boolean> => {
  try {
    await api.unblackContact({ userId: contactId });
    return true;
  } catch (err) {
    console.error("取消拉黑失败", err);
    return false;
  }
};
