// src/hooks/useContactActions.ts
import axios from "axios";

// 删除联系人
export const deleteContact = async (ownerId: string, contactId: string) => {
  try {
    const res = await axios.post("http://localhost:8080/contact/deleteContact", {
      owner_id: ownerId,
      contact_id: contactId,
    });
    alert(res.data.message || "删除成功");
    return true;
  } catch (err) {
    console.error("删除联系人失败", err);
    alert("删除失败");
    return false;
  }
};

// 拉黑联系人
export const blackContact = async (ownerId: string, contactId: string) => {
  try {
    const res = await axios.post("http://localhost:8080/contact/blackContact", {
      owner_id: ownerId,
      contact_id: contactId,
    });
    alert(res.data.message || "已拉黑");
    return true;
  } catch (err) {
    console.error("拉黑联系人失败", err);
    alert("拉黑失败");
    return false;
  }
};

// 取消拉黑
export const cancelBlackContact = async (ownerId: string, contactId: string) => {
  try {
    const res = await axios.post(
      "http://localhost:8080/contact/cancelBlackContact",
      {
        owner_id: ownerId,
        contact_id: contactId,
      }
    );
    if (res.data.code === 200) {
      alert(res.data.message || "已取消拉黑");
      return true;
    } else {
      alert(res.data.message || "操作失败");
      return false;
    }
  } catch (err) {
    console.error("取消拉黑失败", err);
    alert("取消拉黑失败");
    return false;
  }
};
