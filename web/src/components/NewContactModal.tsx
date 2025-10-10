import React, { useEffect, useState } from "react";
import Modal from "./Modal";
import axios from "axios";

interface NewContact {
  contact_id: string;
  contact_name: string;
  contact_avatar: string;
  message: string;
}

interface NewContactModalProps {
  isOpen: boolean;
  onClose: () => void;
  userId: string;
}

const NewContactModal: React.FC<NewContactModalProps> = ({
  isOpen,
  onClose,
  userId,
}) => {
  const [newContacts, setNewContacts] = useState<NewContact[]>([]);

  // 加载新的好友申请
  const loadNewContacts = async () => {
    try {
      const res = await axios.post("http://localhost:8000/contact/getNewContactList", {
        owner_id: userId,
      });
      if (res.data.data) {
        const list = res.data.data.map((c: NewContact) => ({
          ...c,
          contact_avatar: c.contact_avatar.startsWith("http")
            ? c.contact_avatar
            : "http://localhost:8000" + c.contact_avatar,
        }));
        setNewContacts(list);
      } else {
        setNewContacts([]);
      }
    } catch (err) {
      console.error("加载新联系人失败", err);
    }
  };

  useEffect(() => {
    if (isOpen) {
      loadNewContacts();
    }
  }, [isOpen]);

  // 同意申请
  const handleAgree = async (contactId: string) => {
    try {
      const res = await axios.post("http://localhost:8000/contact/passContactApply", {
        owner_id: userId,
        contact_id: contactId,
      });
      alert(res.data.message || "已同意");
      setNewContacts((prev) => prev.filter((c) => c.contact_id !== contactId));
    } catch (err) {
      console.error("同意失败", err);
    }
  };

  // 拒绝申请
  const handleReject = async (contactId: string) => {
    try {
      const res = await axios.post("http://localhost:8000/contact/refuseContactApply", {
        owner_id: userId,
        contact_id: contactId,
      });
      alert(res.data.message || "已拒绝");
      setNewContacts((prev) => prev.filter((c) => c.contact_id !== contactId));
    } catch (err) {
      console.error("拒绝失败", err);
    }
  };

  // 拉黑
  const handleBlack = async (contactId: string) => {
    try {
      const res = await axios.post("http://localhost:8000/contact/blackApply", {
        owner_id: userId,
        contact_id: contactId,
      });
      alert(res.data.message || "已拉黑");
      setNewContacts((prev) => prev.filter((c) => c.contact_id !== contactId));
    } catch (err) {
      console.error("拉黑失败", err);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose}>
      <div className="p-6 w-[400px]">
        <h2 className="text-xl font-bold text-center mb-4">新的朋友</h2>
        {newContacts.length === 0 ? (
          <p className="text-gray-500 text-center">暂无新的好友申请</p>
        ) : (
          <ul className="space-y-3">
            {newContacts.map((c) => (
              <li
                key={c.contact_id}
                className="flex justify-between items-center border p-2 rounded"
              >
                <div className="flex items-center gap-2">
                  <img
                    src={c.contact_avatar}
                    alt="avatar"
                    className="w-8 h-8 rounded-full"
                  />
                  <div>
                    <p className="font-medium">{c.contact_name}</p>
                    <p className="text-xs text-gray-500">{c.message}</p>
                  </div>
                </div>
                <div className="flex flex-col space-y-1">
                  <button
                    onClick={() => handleAgree(c.contact_id)}
                    className="bg-green-500 text-white px-2 py-1 rounded text-sm hover:bg-green-600"
                  >
                    同意
                  </button>
                  <button
                    onClick={() => handleReject(c.contact_id)}
                    className="bg-gray-400 text-white px-2 py-1 rounded text-sm hover:bg-gray-500"
                  >
                    拒绝
                  </button>
                  <button
                    onClick={() => handleBlack(c.contact_id)}
                    className="bg-red-500 text-white px-2 py-1 rounded text-sm hover:bg-red-600"
                  >
                    拉黑
                  </button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </Modal>
  );
};

export default NewContactModal;
