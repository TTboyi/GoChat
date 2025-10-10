import React, { useEffect, useState, useRef } from "react";
import axios from "axios";

interface Message {
  id: string;
  send_id: string;
  send_name: string;
  send_avatar: string;
  content: string;
  created_at: string;
}

interface MessageListProps {
  userId: string;      // 当前登录用户
  contactId: string;   // 会话 ID (Uxxx / Gxxx)
}

const MessageList: React.FC<MessageListProps> = ({ userId, contactId }) => {
  const [messages, setMessages] = useState<Message[]>([]);
  const scrollRef = useRef<HTMLDivElement>(null);

  const loadMessages = async () => {
    try {
      let res;
      if (contactId.startsWith("U")) {
        res = await axios.post("http://localhost:8000/message/getMessageList", {
          user_one_id: userId,
          user_two_id: contactId,
        });
      } else {
        res = await axios.post("http://localhost:8000/message/getGroupMessageList", {
          group_id: contactId,
        });
      }

      if (res.data.data) {
        const list = res.data.data.map((msg: Message) => ({
          ...msg,
          send_avatar: msg.send_avatar.startsWith("http")
            ? msg.send_avatar
            : "http://localhost:8000" + msg.send_avatar,
        }));
        setMessages(list);
      }
    } catch (err) {
      console.error("加载消息失败", err);
    }
  };

  useEffect(() => {
    loadMessages();
  }, [contactId]);

  useEffect(() => {
    // 自动滚动到底部
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [messages]);

  return (
    <div ref={scrollRef} className="flex-1 overflow-y-auto p-4 bg-white">
      {messages.map((msg) => {
        const isMine = msg.send_id === userId;
        return (
          <div
            key={msg.id}
            className={`flex mb-4 ${isMine ? "justify-end" : "justify-start"}`}
          >
            {!isMine && (
              <img
                src={msg.send_avatar}
                alt=""
                className="w-8 h-8 rounded-full mr-2"
              />
            )}
            <div
              className={`max-w-xs px-3 py-2 rounded-lg ${
                isMine ? "bg-blue-500 text-white" : "bg-gray-200 text-black"
              }`}
            >
              {!isMine && (
                <p className="text-xs text-gray-600 mb-1">{msg.send_name}</p>
              )}
              <p>{msg.content}</p>
              <p className="text-[10px] text-gray-500 mt-1 text-right">
                {new Date(msg.created_at).toLocaleTimeString()}
              </p>
            </div>
            {isMine && (
              <img
                src={msg.send_avatar}
                alt=""
                className="w-8 h-8 rounded-full ml-2"
              />
            )}
          </div>
        );
      })}
    </div>
  );
};

export default MessageList;
