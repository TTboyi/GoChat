import React from "react";
import type { IncomingCall } from "../../hooks/useWebRTC";

interface IncomingCallModalProps {
  call: IncomingCall;
  onAccept: () => void;
  onReject: () => void;
}

const IncomingCallModal: React.FC<IncomingCallModalProps> = ({ call, onAccept, onReject }) => {
  const isVideo = call.callType === "video";

  return (
    <div className="fixed inset-0 z-[1000] flex items-center justify-center bg-black/60">
      <div className="bg-white rounded-2xl shadow-2xl p-8 flex flex-col items-center gap-5 w-72 sm:w-80 animate-scaleIn">
        {/* 图标 */}
        <div className="w-16 h-16 rounded-full bg-green-100 flex items-center justify-center text-3xl animate-bounce">
          {isVideo ? "📹" : "📞"}
        </div>

        {/* 来电信息 */}
        <div className="text-center">
          <p className="text-gray-500 text-sm mb-1">
            {isVideo ? "视频通话邀请" : "语音通话邀请"}
          </p>
          <p className="font-semibold text-gray-800 text-lg truncate max-w-[200px]">
            {call.fromName || call.from}
          </p>
        </div>

        {/* 操作按钮 */}
        <div className="flex gap-8 mt-2">
          {/* 拒绝 */}
          <button
            onClick={onReject}
            className="w-14 h-14 rounded-full bg-red-500 hover:bg-red-600 flex items-center justify-center shadow-md transition-colors"
            title="拒绝"
          >
            <svg viewBox="0 0 24 24" fill="white" className="w-7 h-7">
              <path d="M6.6 10.8c1.4 2.8 3.8 5.1 6.6 6.6l2.2-2.2c.3-.3.7-.4 1-.2 1.1.4 2.3.6 3.6.6.6 0 1 .4 1 1V20c0 .6-.4 1-1 1-9.4 0-17-7.6-17-17 0-.6.4-1 1-1h3.5c.6 0 1 .4 1 1 0 1.3.2 2.5.6 3.6.1.3 0 .7-.2 1L6.6 10.8z"/>
            </svg>
          </button>

          {/* 接受 */}
          <button
            onClick={onAccept}
            className="w-14 h-14 rounded-full bg-green-500 hover:bg-green-600 flex items-center justify-center shadow-md transition-colors"
            title="接听"
          >
            <svg viewBox="0 0 24 24" fill="white" className="w-7 h-7">
              <path d="M20.01 15.38c-1.23 0-2.42-.2-3.53-.56-.35-.12-.74-.03-1.01.24l-1.57 1.97c-2.83-1.35-5.48-3.9-6.89-6.83l1.95-1.66c.27-.28.35-.67.24-1.02-.37-1.11-.56-2.3-.56-3.53 0-.54-.45-.99-.99-.99H4.19C3.65 3 3 3.24 3 3.99 3 13.28 10.73 21 20.01 21c.71 0 .99-.63.99-1.18v-3.45c0-.54-.45-.99-.99-.99z"/>
            </svg>
          </button>
        </div>
      </div>
    </div>
  );
};

export default IncomingCallModal;
