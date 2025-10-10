import React, { useState } from "react";
import Modal from "./Modal";
import { useForm } from "react-hook-form";
import axios from "axios";

interface UpdateGroupInfoModalProps {
  isOpen: boolean;
  onClose: () => void;
  groupId: string; // 当前群聊 uuid
  onSuccess?: () => void;
}

interface FormData {
  name: string;
  notice: string;
  addMode: 0 | 1 | -1;
  avatar: File | null;
}

const UpdateGroupInfoModal: React.FC<UpdateGroupInfoModalProps> = ({
  isOpen,
  onClose,
  groupId,
  onSuccess,
}) => {
  const { register, handleSubmit, setValue } = useForm<FormData>({
    defaultValues: {
      addMode: -1, // 默认不修改
    },
  });
  const [previewUrl, setPreviewUrl] = useState<string>("");

  const onSubmit = async (data: FormData) => {
    try {
      let avatarUrl = "";

      // 如果上传了新头像，先传 /upload/avatar
      if (data.avatar) {
        const formData = new FormData();
        formData.append("file", data.avatar);

        const res = await axios.post("http://localhost:8000/upload/avatar", formData, {
          headers: { "Content-Type": "multipart/form-data" },
        });
        avatarUrl = res.data.url;
      }

      // 组装 payload
      const payload = {
        uuid: groupId,
        name: data.name || "",
        notice: data.notice || "",
        addMode: data.addMode === -1 ? undefined : Number(data.addMode),
        avatar: avatarUrl || "",
      };

      const res2 = await axios.post("http://localhost:8000/group/updateGroupInfo", payload, {
        headers: { "Content-Type": "application/json" },
      });

      alert(res2.data.message || "更新成功");
      if (onSuccess) onSuccess();
      onClose();
    } catch (error) {
      console.error("更新群聊失败", error);
      alert("更新失败");
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
        <h2 className="text-xl font-bold text-center">修改群资料</h2>

        <div>
          <label className="block mb-1">群名称</label>
          <input
            {...register("name", {
              minLength: { value: 3, message: "群名称至少 3 个字符" },
              maxLength: { value: 10, message: "群名称最多 10 个字符" },
            })}
            placeholder="选填"
            className="w-full border rounded p-2"
          />
        </div>

        <div>
          <label className="block mb-1">入群方式</label>
          <div className="flex gap-4">
            <label className="flex items-center gap-2">
              <input type="radio" value="0" {...register("addMode")} />
              直接加入
            </label>
            <label className="flex items-center gap-2">
              <input type="radio" value="1" {...register("addMode")} />
              群主审核
            </label>
          </div>
        </div>

        <div>
          <label className="block mb-1">群公告</label>
          <textarea
            {...register("notice")}
            placeholder="选填，最多 500 字"
            maxLength={500}
            rows={3}
            className="w-full border rounded p-2"
          />
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

export default UpdateGroupInfoModal;
