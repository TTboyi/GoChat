import React, { useRef } from "react";
import type { SessionItem } from "../../types/chat";

const PhoneIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="w-[18px] h-[18px]">
    <path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07A19.5 19.5 0 0 1 4.69 13.5 19.79 19.79 0 0 1 1.61 4.87 2 2 0 0 1 3.6 2.69h3a2 2 0 0 1 2 1.72c.127.96.361 1.903.7 2.81a2 2 0 0 1-.45 2.11L7.91 10.3a16 16 0 0 0 6 6l.91-.91a2 2 0 0 1 2.11-.45c.907.339 1.85.573 2.81.7a2 2 0 0 1 1.72 2.02z"/>
  </svg>
);

const VideoIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="w-[18px] h-[18px]">
    <polygon points="23 7 16 12 23 17 23 7"/>
    <rect x="1" y="5" width="15" height="14" rx="2" ry="2"/>
  </svg>
);

const FileIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="w-[18px] h-[18px]">
    <path d="M21.44 11.05l-9.19 9.19a6 6 0 0 1-8.49-8.49l9.19-9.19a4 4 0 0 1 5.66 5.66l-9.2 9.19a2 2 0 0 1-2.83-2.83l8.49-8.48"/>
  </svg>
);

const SearchIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="w-[18px] h-[18px]">
    <circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/>
  </svg>
);

const SendIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" className="w-4 h-4">
    <path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z"/>
  </svg>
);

interface ChatInputProps {
  input: string;
  activeId: string;
  active: SessionItem | undefined;
  onChange: (value: string) => void;
  onKeyDown: (e: React.KeyboardEvent<HTMLTextAreaElement>) => void;
  onSend: () => void;
  onStartCall?: (type: "audio" | "video") => void;
  onSendFile?: (file: File) => void;
  onOpenSearch?: () => void;
}

const ChatInput: React.FC<ChatInputProps> = ({
  input,
  activeId,
  active,
  onChange,
  onKeyDown,
  onSend,
  onStartCall,
  onSendFile,
  onOpenSearch,
}) => {
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file && onSendFile) {
      onSendFile(file);
      e.target.value = "";
    }
  };

  const toolBtn = "p-1 rounded text-gray-400 hover:text-gray-600 hover:bg-gray-100 transition flex items-center justify-center";
  const disabled = !activeId;

  return (
    <div className="border-t border-gray-200 bg-white">
      {/* 紧凑工具栏 */}
      {activeId && (
        <div className="flex items-center px-4 pt-1.5 pb-0.5 gap-0.5">
          {/* 文件上传 */}
          <button className={toolBtn} title="发送文件" onClick={() => fileInputRef.current?.click()}>
            <FileIcon />
          </button>
          <input ref={fileInputRef} type="file" className="hidden" onChange={handleFileChange} />

          {/* 历史搜索 */}
          <button className={toolBtn} title="搜索历史消息" onClick={onOpenSearch}>
            <SearchIcon />
          </button>

          {/* 通话（仅单聊） */}
          {active?.type === "user" && onStartCall && (
            <>
              <div className="w-px h-4 bg-gray-200 mx-0.5" />
              <button className={toolBtn} title="语音通话" onClick={() => onStartCall("audio")}>
                <PhoneIcon />
              </button>
              <button className={toolBtn} title="视频通话" onClick={() => onStartCall("video")}>
                <VideoIcon />
              </button>
            </>
          )}
        </div>
      )}

      {/* 输入框（发送按钮融合在右下角） */}
      <div className="px-4 py-2">
        <div className="relative border border-gray-300 rounded-xl bg-white focus-within:border-gray-400 transition overflow-hidden">
          <textarea
            className="w-full resize-none outline-none px-3 pt-2.5 pb-8 text-sm text-gray-800 bg-transparent"
            style={{ minHeight: 72, maxHeight: 140 }}
            placeholder={disabled ? "请选择左侧会话" : "输入消息…  Enter 发送，Shift+Enter 换行"}
            value={input}
            onChange={(e) => onChange(e.target.value)}
            onKeyDown={onKeyDown}
            disabled={disabled}
          />
          {/* 融合的发送按钮 */}
          <button
            onClick={onSend}
            disabled={disabled || !input.trim()}
            className={
              "absolute bottom-2 right-2 flex items-center gap-1 px-3 py-1.5 rounded-lg text-xs font-medium transition " +
              (disabled || !input.trim()
                ? "bg-gray-100 text-gray-400 cursor-not-allowed"
                : "bg-[#07c160] text-white hover:bg-green-500 shadow-sm")
            }
          >
            <SendIcon />
            <span>发送</span>
          </button>
        </div>
      </div>
    </div>
  );
};

export default ChatInput;
