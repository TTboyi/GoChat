import React from "react";
import Modal from "./Modal";
import { useForm } from "react-hook-form";
import axios from "axios";

interface ApplyGroupModalProps {
  isOpen: boolean;
  onClose: () => void;
  userId: string; // 当前用户 uuid
}

interface FormData {
  contactId: string;
  message: string;
}

const ApplyGroupModal: React.FC<ApplyGroupModalProps> = ({
  isOpen,
  onClose,
  userId,
}) => {
  const { register, handleSubmit, formState: { errors } } = useForm<FormData>();

  const onSubmit = async (data: FormData) => {
    try {
      // 第一步：先查入群方式
      const checkRes = await axios.post("http://localhost:8000/group/checkGroupAddMode", {
        uuid: data.contactId,
      });

      if (checkRes.data.add_mode === 0) {
        // 直接加入
        const res = await axios.post("http://localhost:8000/group/enterGroupDirectly", {
          groupUuid: data.contactId,
          message: data.message,
        });
        alert(res.data.message || "加入成功");
        onClose();
        return;
      }

      // 第二步：需要审核 → 走申请接口
      const res2 = await axios.post("http://localhost:8000/contact/applyContact", {
        contact_id: data.contactId,
        owner_id: userId,
        message: data.message,
      });

      if (res2.data.code === 200) {
        alert("申请成功");
        onClose();
      } else {
        alert(res2.data.message || "申请失败");
      }
    } catch (error: any) {
      console.error("申请失败", error);
      alert("操作失败");
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose}>
      <form
        onSubmit={handleSubmit(onSubmit)}
        className="w-[400px] p-6 flex flex-col space-y-4"
      >
        <h2 className="text-xl font-bold text-center">添加用户/群聊</h2>

        <div>
          <label className="block mb-1">用户/群聊 ID</label>
          <input
            {...register("contactId", { required: "请输入 ID" })}
            placeholder="请填写申请的用户/群聊 ID"
            className="w-full border rounded p-2"
          />
          {errors.contactId && (
            <p className="text-red-500 text-sm">{errors.contactId.message}</p>
          )}
        </div>

        <div>
          <label className="block mb-1">申请消息（选填）</label>
          <textarea
            {...register("message")}
            placeholder="填写更容易通过"
            className="w-full border rounded p-2"
            rows={3}
          />
        </div>

        <div className="text-right">
          <button
            type="submit"
            className="bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600"
          >
            完成
          </button>
        </div>
      </form>
    </Modal>
  );
};

export default ApplyGroupModal;
