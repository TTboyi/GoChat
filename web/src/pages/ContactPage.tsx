import React, { useState } from "react";
import ContactList from "../components/ContactList";
import ContactInfoModal from "../components/ContactInfoModal";

const ContactPage: React.FC = () => {
  const userId = "user-uuid-xxx"; // 从登录态取
  const [selectedContact, setSelectedContact] = useState<any>(null);
  const [showInfo, setShowInfo] = useState(false);

  return (
    <div className="flex h-screen">
      <aside className="w-64 bg-gray-100 p-4 border-r">
        <ContactList
          userId={userId}
          onSelect={(c) => {
            setSelectedContact(c);
            setShowInfo(true);
          }}
        />
      </aside>
      <main className="flex-1 flex items-center justify-center">
        <p className="text-gray-400">选择联系人查看详情</p>
      </main>

      <ContactInfoModal
        isOpen={showInfo}
        onClose={() => setShowInfo(false)}
        contact={selectedContact}
      />
    </div>
  );
};

export default ContactPage;
