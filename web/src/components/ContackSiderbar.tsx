// components/ContactSidebar.tsx
import React from "react";

type User = {
  user_id: string;
  user_name: string;
  avatar: string;
};

type Group = {
  group_id: string;
  group_name: string;
  avatar: string;
};

interface Props {
  contactUserList: User[];
  myGroupList: Group[];
  myJoinedGroupList: Group[];
  onChatUser: (user: User) => void;
  onChatGroup: (group: Group) => void;
}

const ContactSidebar: React.FC<Props> = ({
  contactUserList,
  myGroupList,
  myJoinedGroupList,
  onChatUser,
  onChatGroup,
}) => {
  return (
    <aside className="w-64 bg-white/60 h-full overflow-y-auto p-4 space-y-6 shadow-lg">
      <h3 className="text-lg font-semibold text-gray-800">联系人</h3>
      <div className="space-y-3">
        {contactUserList.map((user) => (
          <div
            key={user.user_id}
            className="flex items-center gap-2 cursor-pointer hover:bg-gray-200 p-2 rounded-md"
            onClick={() => onChatUser(user)}
          >
            <img src={user.avatar} alt="" className="w-8 h-8 rounded-full" />
            <span>{user.user_name}</span>
          </div>
        ))}
      </div>

      <div>
        <h3 className="mt-4 mb-2 text-lg font-semibold text-gray-800">我创建的群聊</h3>
        {myGroupList.map((group) => (
          <div
            key={group.group_id}
            className="flex items-center gap-2 cursor-pointer hover:bg-gray-200 p-2 rounded-md"
            onClick={() => onChatGroup(group)}
          >
            <img src={group.avatar} className="w-8 h-8 rounded-md" />
            <span>{group.group_name}</span>
          </div>
        ))}
      </div>

      <div>
        <h3 className="mt-4 mb-2 text-lg font-semibold text-gray-800">我加入的群聊</h3>
        {myJoinedGroupList.map((group) => (
          <div
            key={group.group_id}
            className="flex items-center gap-2 cursor-pointer hover:bg-gray-200 p-2 rounded-md"
            onClick={() => onChatGroup(group)}
          >
            <img src={group.avatar} className="w-8 h-8 rounded-md" />
            <span>{group.group_name}</span>
          </div>
        ))}
      </div>
    </aside>
  );
};

export default ContactSidebar;
