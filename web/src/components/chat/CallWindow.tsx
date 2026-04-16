// ============================================================
// 文件：web/src/components/chat/CallWindow.tsx
// 作用：通话中的窗口组件（显示本地/远端视频画面，提供挂断按钮）。
//
// localVideoRef / remoteVideoRef：
//   这两个 ref 由 useWebRTC Hook 传入，直接绑定到 <video> 元素。
//   当 WebRTC 连接建立后，useWebRTC 内部会把 MediaStream 赋值给
//   videoElement.srcObject，浏览器自动开始播放视频。
// ============================================================
import React, { useEffect } from "react";
import type { RefObject } from "react";
import type { CallState } from "../../types/chat";

interface CallWindowProps {
  callState: CallState;
  localVideoRef: RefObject<HTMLVideoElement | null>;
  remoteVideoRef: RefObject<HTMLVideoElement | null>;
  onEndCall: () => void;
}

const CallWindow: React.FC<CallWindowProps> = ({
  callState,
  localVideoRef,
  remoteVideoRef,
  onEndCall,
}) => {
  // 挂断时播放音效（可选）
  useEffect(() => {
    if (callState.status === "idle") return;
    // 通话窗口打开时锁定背景滚动
    document.body.style.overflow = "hidden";
    return () => { document.body.style.overflow = ""; };
  }, [callState.status]);

  if (callState.status === "idle") return null;

  const isVideo = callState.callType === "video";
  const isRinging = callState.status === "ringing";
  const isInCall = callState.status === "in-call";

  return (
    <div className="fixed inset-0 z-[999] flex flex-col items-center justify-center bg-gray-900/95">
      {/* ---- 视频区域 ---- */}
      {isVideo ? (
        <div className="relative w-full h-full max-w-2xl mx-auto flex flex-col items-center justify-center">
          {/* 远端视频（全屏背景） */}
          <video
            ref={remoteVideoRef}
            autoPlay
            playsInline
            className={`w-full h-full object-cover rounded-none ${isRinging ? "hidden" : "block"}`}
          />

          {/* 响铃时的等待界面 */}
          {isRinging && (
            <div className="flex flex-col items-center gap-4 text-white">
              <div className="w-20 h-20 rounded-full bg-white/20 flex items-center justify-center text-4xl animate-pulse">
                📹
              </div>
              <p className="text-xl font-semibold">等待对方接听...</p>
            </div>
          )}

          {/* 本地小窗（右上角） */}
          <div className="absolute top-4 right-4 w-28 h-20 sm:w-36 sm:h-28 rounded-xl overflow-hidden border-2 border-white/30 shadow-lg bg-black">
            <video
              ref={localVideoRef}
              autoPlay
              playsInline
              muted
              className="w-full h-full object-cover"
            />
          </div>
        </div>
      ) : (
        /* ---- 语音通话界面 ---- */
        <div className="flex flex-col items-center gap-6 text-white px-6">
          <div className="w-24 h-24 rounded-full bg-white/20 flex items-center justify-center text-5xl">
            🎧
          </div>
          {isRinging ? (
            <p className="text-xl font-semibold animate-pulse">呼叫中...</p>
          ) : (
            <p className="text-xl font-semibold">语音通话中</p>
          )}
          {/* 隐藏的 audio 播放远端流（由 useWebRTC 创建 audio 元素） */}
          <video ref={localVideoRef} autoPlay playsInline muted className="hidden" />
          <video ref={remoteVideoRef} autoPlay playsInline className="hidden" />
        </div>
      )}

      {/* ---- 操作按钮 ---- */}
      <div className="absolute bottom-10 left-0 right-0 flex justify-center">
        <button
          onClick={onEndCall}
          className="w-16 h-16 rounded-full bg-red-500 hover:bg-red-600 active:bg-red-700 flex items-center justify-center shadow-lg transition-colors"
          title="挂断"
        >
          <svg viewBox="0 0 24 24" fill="white" className="w-8 h-8">
            <path d="M6.6 10.8c1.4 2.8 3.8 5.1 6.6 6.6l2.2-2.2c.3-.3.7-.4 1-.2 1.1.4 2.3.6 3.6.6.6 0 1 .4 1 1V20c0 .6-.4 1-1 1-9.4 0-17-7.6-17-17 0-.6.4-1 1-1h3.5c.6 0 1 .4 1 1 0 1.3.2 2.5.6 3.6.1.3 0 .7-.2 1L6.6 10.8z"/>
          </svg>
        </button>
      </div>
    </div>
  );
};

export default CallWindow;
