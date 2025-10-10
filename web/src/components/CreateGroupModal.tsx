import React, { useState } from "react";
import Modal from "./Modal";
import { useForm } from "react-hook-form";
import axios from "axios";

interface CreateGroupModalProps {
  isOpen: boolean;
  onClose: () => void;
  ownerId: string; // 当前用户 uuid
  onSuccess?: () => void;
}

interface FormData {
  name: string;
  notice: string;
  addMode: 0 | 1;
  avatar: File | null;
}

const CreateGroupModal: React.FC<CreateGroupModalProps> = ({
  isOpen,
  onClose,
  ownerId,
  onSuccess,
}) => {
  const { register, handleSubmit, setValue, formState: { errors } } = useForm<FormData>();
  const [previewUrl, setPreviewUrl] = useState<string>("");

  const onSubmit = async (data: any) => {
    try {
      let avatarUrl = "";

      // 如果选择了文件，先上传
      if (data.avatar) {
        const formData = new FormData();
        formData.append("file", data.avatar);

        const res = await axios.post(
          "http://localhost:8000/upload/avatar",
          formData,
          { headers: { "Content-Type": "multipart/form-data" } }
        );
        avatarUrl = res.data.url;
      }

      // 发送 JSON 请求
      const payload = {
        name: data.name,
        notice: data.notice || "",
        ownerId,
        addMode: Number(data.addMode),
        avatar: avatarUrl || "", // 如果没上传，用后端默认值
      };

      const res2 = await axios.post(
        "http://localhost:8000/group/createGroup",
        payload,
        { headers: { "Content-Type": "application/json" } }
      );

      console.log("创建群聊成功：", res2.data);
      if (onSuccess) onSuccess();
      onClose();
    } catch (error) {
      console.error("创建群聊失败", error);
    }
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0] || null;
    setValue("avatar", file);
    if (file) setPreviewUrl(URL.createObjectURL(file));
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose}>
      <form
        onSubmit={handleSubmit(onSubmit)}
        className="w-[400px] p-6 flex flex-col space-y-4"
      >
        <h2 className="text-xl font-bold text-center">创建群聊</h2>

        <div>
          <label className="block mb-1">群名称</label>
          <input
            {...register("name", { required: "请输入群名称" })}
            placeholder="例如：技术交流群"
            className="w-full border rounded p-2"
          />
          {errors.name && <p className="text-red-500">{errors.name.message}</p>}
        </div>

        <div>
          <label className="block mb-1">群公告</label>
          <textarea
            {...register("notice")}
            placeholder="选填"
            className="w-full border rounded p-2"
          />
        </div>

        <div>
          <label className="block mb-1">加群方式</label>
          <div className="flex gap-4">
            <label className="flex items-center gap-2">
              <input
                type="radio"
                value="0"
                {...register("addMode", { required: "请选择加群方式" })}
              />
              直接加入
            </label>
            <label className="flex items-center gap-2">
              <input
                type="radio"
                value="1"
                {...register("addMode", { required: "请选择加群方式" })}
              />
              群主审核
            </label>
          </div>
          {errors.addMode && (
            <p className="text-red-500">{errors.addMode.message}</p>
          )}
        </div>

        <div>
          <label className="block mb-1">群头像</label>
          <input type="file" accept="image/*" onChange={handleFileChange} />
          {previewUrl && (
            <img src={previewUrl} alt="预览" className="w-16 h-16 mt-2 rounded" />
          )}
        </div>

        <button
          type="submit"
          className="bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600"
        >
          完成
        </button>
      </form>
    </Modal>
  );
};

export default CreateGroupModal;
