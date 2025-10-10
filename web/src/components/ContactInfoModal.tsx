import React from "react";
import Modal from "./Modal";

interface ContactInfo {
  contact_id: string;
  contact_name: string;
  contact_avatar: string;
  contact_gender: number;
  contact_phone: string;
  contact_email: string;
  contact_birthday: string;
  contact_signature: string;
}

interface ContactInfoModalProps {
  isOpen: boolean;
  onClose: () => void;
  contact: ContactInfo | null;
}

const ContactInfoModal: React.FC<ContactInfoModalProps> = ({
  isOpen,
  onClose,
  contact,
}) => {
  if (!contact) return null;

  return (
    <Modal isOpen={isOpen} onClose={onClose}>
      <div className="p-6 space-y-4">
        <h2 className="text-xl font-bold text-center">个人主页</h2>
        <div className="flex justify-center">
          <img
            src={contact.contact_avatar}
            alt="avatar"
            className="w-24 h-24 rounded-full"
          />
        </div>
        <p>ID: {contact.contact_id}</p>
        <p>昵称: {contact.contact_name}</p>
        <p>性别: {contact.contact_gender === 0 ? "男" : "女"}</p>
        <p>电话: {contact.contact_phone}</p>
        <p>邮箱: {contact.contact_email}</p>
        <p>生日: {contact.contact_birthday}</p>
        <p>个性签名: {contact.contact_signature}</p>
      </div>
    </Modal>
  );
};

export default ContactInfoModal;
