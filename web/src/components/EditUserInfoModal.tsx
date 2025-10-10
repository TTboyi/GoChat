// components/EditUserInfoModal.tsx
import React from "react";
import { useForm } from "react-hook-form";
import Modal from "./Modal";

interface Props {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (data: any) => void;
}

const EditUserInfoModal: React.FC<Props> = ({ isOpen, onClose, onSubmit }) => {
  const { register, handleSubmit, formState: { errors } } = useForm();

  return (
    <Modal isOpen={isOpen} onClose={onClose}>
      <form onSubmit={handleSubmit(onSubmit)} className="p-4 space-y-4 w-[400px]">
        <h3 className="text-xl font-bold text-center">修改主页</h3>

        <div>
          <label>昵称</label>
          <input {...register("nickname", {
            minLength: 3,
            maxLength: 10,
          })} className="w-full border p-2 rounded" placeholder="选填" />
        </div>

        <div>
          <label>邮箱</label>
          <input {...register("email")} className="w-full border p-2 rounded" placeholder="选填" />
        </div>

        <div>
          <label>生日</label>
          <input {...register("birthday")} className="w-full border p-2 rounded" placeholder="2024.1.1" />
        </div>

        <div>
          <label>个性签名</label>
          <input {...register("signature")} className="w-full border p-2 rounded" placeholder="选填" />
        </div>

        {/* 文件上传后续可用 react-dropzone 实现 */}
        <div>
          <label>头像上传（未实现）</label>
        </div>

        <div className="text-center">
          <button type="submit" className="bg-blue-500 text-white px-4 py-2 rounded">完成</button>
        </div>
      </form>
    </Modal>
  );
};

export default EditUserInfoModal;
