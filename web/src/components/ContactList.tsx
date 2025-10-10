import React, { useEffect, useState } from "react";
import axios from "axios";

interface Contact {
  user_id: string;
  user_name: string;
  avatar: string;
}

interface ContactListProps {
  onSelect: (contact: Contact) => void;
  userId: string; // 当前登录用户
}

const ContactList: React.FC<ContactListProps> = ({ onSelect, userId }) => {
  const [contacts, setContacts] = useState<Contact[]>([]);

  const loadContacts = async () => {
    try {
      const res = await axios.post("http://localhost:8000/contact/getUserList", {
        owner_id: userId,
      });
      if (res.data.data) {
        const list = res.data.data.map((c: Contact) => ({
          ...c,
          avatar: c.avatar.startsWith("http")
            ? c.avatar
            : "http://localhost:8000" + c.avatar,
        }));
        setContacts(list);
      }
    } catch (err) {
      console.error("加载联系人失败", err);
    }
  };

  useEffect(() => {
    loadContacts();
  }, []);

  return (
    <div className="space-y-2">
      <h3 className="font-bold text-lg">联系人</h3>
      {contacts.map((c) => (
        <div
          key={c.user_id}
          onClick={() => onSelect(c)}
          className="flex items-center gap-2 p-2 rounded cursor-pointer hover:bg-gray-200"
        >
          <img src={c.avatar} alt="" className="w-8 h-8 rounded-full" />
          <span>{c.user_name}</span>
        </div>
      ))}
    </div>
  );
};

export default ContactList;
