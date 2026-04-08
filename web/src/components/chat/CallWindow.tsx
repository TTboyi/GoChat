import React from "react";
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
  if (callState.status !== "in-call") return null;

  return (
    <div className="fixed bottom-5 right-5 bg-gray-800 text-white rounded-lg shadow-lg p-4 z-50">
      <div className="flex flex-col items-center space-y-2">
        {callState.callType === "video" ? (
          <div className="flex space-x-2">
            <video
              ref={localVideoRef}
              autoPlay
              playsInline
              muted
              className="w-40 h-32 bg-black rounded-md"
            />
            <video
              ref={remoteVideoRef}
              autoPlay
              playsInline
              className="w-40 h-32 bg-black rounded-md"
            />
          </div>
        ) : (
          <p className="text-sm text-gray-200">🎧 正在语音通话中...</p>
        )}
        <button
          onClick={onEndCall}
          className="mt-2 px-4 py-1 bg-red-500 hover:bg-red-600 rounded"
        >
          挂断
        </button>
      </div>
    </div>
  );
};

export default CallWindow;
