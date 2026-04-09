import { useRef, useState, useCallback, useEffect } from "react";
import type { RefObject } from "react";
import type { CallState, SessionItem } from "../types/chat";
import type { ChatWebSocket } from "../api/socket";
import { callCandidate, callEnd } from "../api/socket";

const IDLE_CALL_STATE: CallState = {
  callId: null,
  peerId: null,
  status: "idle",
  callType: null,
  isCaller: false,
};

export interface IncomingCall {
  callId: string;
  from: string;
  fromName?: string;
  callType: "audio" | "video";
  offer: string; // JSON stringified RTCSessionDescriptionInit
}

export function useWebRTC(
  wsRef: RefObject<ChatWebSocket | null>,
  userId: string | undefined
) {
  const [callState, setCallState] = useState<CallState>(IDLE_CALL_STATE);
  const callStateRef = useRef<CallState>(IDLE_CALL_STATE);

  // 待接听的来电
  const [incomingCall, setIncomingCall] = useState<IncomingCall | null>(null);

  const localVideoRef = useRef<HTMLVideoElement>(null);
  const remoteVideoRef = useRef<HTMLVideoElement>(null);
  const peerRef = useRef<RTCPeerConnection | null>(null);
  const localStreamRef = useRef<MediaStream | null>(null);

  // Keep ref in sync so signal handlers don't go stale
  useEffect(() => {
    callStateRef.current = callState;
  }, [callState]);

  // ✅ 当 callState 变为非 idle 时，将本地流挂载到 video 元素
  useEffect(() => {
    if (callState.status === "idle") return;
    if (!localStreamRef.current) return;

    const tryAttach = () => {
      if (localVideoRef.current && localStreamRef.current) {
        localVideoRef.current.srcObject = localStreamRef.current;
      }
    };
    tryAttach();
    // 等 DOM 更新完毕后再试一次
    const t = setTimeout(tryAttach, 80);
    return () => clearTimeout(t);
  }, [callState.status]);

  const cleanupCall = useCallback(() => {
    setCallState(IDLE_CALL_STATE);
    if (peerRef.current) {
      peerRef.current.close();
      peerRef.current = null;
    }
    if (localStreamRef.current) {
      localStreamRef.current.getTracks().forEach((t) => t.stop());
      localStreamRef.current = null;
    }
    if (localVideoRef.current) localVideoRef.current.srcObject = null;
    if (remoteVideoRef.current) remoteVideoRef.current.srcObject = null;
  }, []);

  const startCall = useCallback(
    async (callType: "audio" | "video", active: SessionItem) => {
      const ws = wsRef.current;
      if (!ws || !active?.id) return;

      const callId = Date.now().toString();

      try {
        const devices = await navigator.mediaDevices.enumerateDevices();
        const hasMic = devices.some((d) => d.kind === "audioinput");
        const hasCam = devices.some((d) => d.kind === "videoinput");
        if (callType === "video" && !hasCam) {
          alert("未检测到摄像头设备，请检查硬件或权限。");
          return;
        }
        if (!hasMic) {
          alert("未检测到麦克风设备，请检查硬件或权限。");
          return;
        }
      } catch (err) {
        console.warn("🎥 无法枚举设备:", err);
      }

      let stream: MediaStream;
      try {
        const constraints =
          callType === "video"
            ? { video: true, audio: true }
            : { audio: true };
        stream = await navigator.mediaDevices.getUserMedia(constraints);
      } catch (err: any) {
        console.error("🚫 无法访问麦克风或摄像头:", err);
        alert("无法访问麦克风或摄像头，请检查浏览器权限。");
        return;
      }

      localStreamRef.current = stream;

      // 先设置状态（触发 CallWindow 渲染 + useEffect 挂载 stream）
      setCallState({
        callId,
        peerId: active.id,
        status: "ringing",
        callType,
        isCaller: true,
      });

      const pc = new RTCPeerConnection({
        iceServers: [
          { urls: "stun:stun1.l.google.com:19302" },
          { urls: "stun:stun2.l.google.com:19302" },
        ],
      });
      peerRef.current = pc;

      if (callType === "video") {
        stream.getVideoTracks().forEach((t) => pc.addTrack(t, stream));
      }
      stream.getAudioTracks().forEach((t) => pc.addTrack(t, stream));

      pc.ontrack = (event) => {
        const [remoteStream] = event.streams;
        if (remoteVideoRef.current && callType === "video") {
          remoteVideoRef.current.srcObject = remoteStream;
        }
        if (callType === "audio") {
          const audioEl = document.createElement("audio");
          audioEl.srcObject = remoteStream;
          audioEl.autoplay = true;
          document.body.appendChild(audioEl);
        }
      };

      pc.onicecandidate = (event) => {
        if (event.candidate) {
          callCandidate(ws, active.id, callId, event.candidate);
        }
      };

      const offer = await pc.createOffer();
      await pc.setLocalDescription(offer);

      ws.send({
        action: "call_invite",
        receiveId: active.id,
        callType,
        callId,
        content: JSON.stringify(offer),
      });
    },
    [wsRef]
  );

  // ✅ 主叫取消呼叫 / 通话结束
  const endCall = useCallback(() => {
    const ws = wsRef.current;
    const { peerId, callId } = callStateRef.current;
    if (ws && peerId && callId) {
      callEnd(ws, peerId, callId);
    }
    cleanupCall();
  }, [wsRef, cleanupCall]);

  // ✅ 接受来电
  const acceptIncomingCall = useCallback(async () => {
    const pending = incomingCall;
    if (!pending) return;
    const socket = wsRef.current;
    if (!socket) return;

    setIncomingCall(null);

    const { callId, from, callType, offer: offerStr } = pending;

    setCallState({
      callId,
      peerId: from,
      status: "in-call",
      callType,
      isCaller: false,
    });

    let stream: MediaStream;
    try {
      const constraints =
        callType === "video" ? { video: true, audio: true } : { audio: true };
      stream = await navigator.mediaDevices.getUserMedia(constraints);
    } catch {
      alert("无法访问摄像头/麦克风");
      socket.send({ action: "call_answer", receiveId: from, callId, accept: false });
      cleanupCall();
      return;
    }

    localStreamRef.current = stream;

    const pc2 = new RTCPeerConnection({
      iceServers: [
        { urls: "stun:stun1.l.google.com:19302" },
        { urls: "stun:stun2.l.google.com:19302" },
      ],
    });
    peerRef.current = pc2;
    stream.getTracks().forEach((t) => pc2.addTrack(t, stream));

    pc2.ontrack = (event) => {
      if (remoteVideoRef.current) {
        remoteVideoRef.current.srcObject = event.streams[0];
      }
    };

    pc2.onicecandidate = (event) => {
      if (event.candidate) {
        socket.send({
          action: "call_candidate",
          receiveId: from,
          callId,
          content: JSON.stringify(event.candidate),
        });
      }
    };

    const remoteOffer = JSON.parse(offerStr);
    await pc2.setRemoteDescription(new RTCSessionDescription(remoteOffer));
    const answer = await pc2.createAnswer();
    await pc2.setLocalDescription(answer);

    socket.send({
      action: "call_answer",
      receiveId: from,
      callId,
      accept: true,
      content: JSON.stringify(answer),
    });
  }, [incomingCall, wsRef, cleanupCall]);

  // ✅ 拒绝来电
  const rejectIncomingCall = useCallback(() => {
    const pending = incomingCall;
    if (!pending) return;
    const socket = wsRef.current;
    socket?.send({
      action: "call_answer",
      receiveId: pending.from,
      callId: pending.callId,
      accept: false,
    });
    setIncomingCall(null);
  }, [incomingCall, wsRef]);

  const handleCallSignal = useCallback(
    async (msg: any) => {
      console.log("📨 收到信令:", msg);
      const { action, from, callType, callId, accept, content } = msg;
      const pc = peerRef.current;
      const socket = wsRef.current;
      const me = userId;

      switch (action) {
        case "call_invite": {
          if (from === me) return;
          if (callStateRef.current.status !== "idle") {
            // 忙线，自动拒绝
            socket?.send({
              action: "call_answer",
              receiveId: from,
              callId,
              accept: false,
            });
            return;
          }
          // ✅ 使用状态而非 confirm 阻塞
          setIncomingCall({
            callId,
            from,
            callType,
            offer: content || "",
          });
          break;
        }

        case "call_answer": {
          if (from === me) return;
          if (accept === false) {
            setCallState(IDLE_CALL_STATE);
            cleanupCall();
            alert("🚫 对方拒绝通话");
            return;
          }
          setCallState((prev) =>
            prev.status === "in-call"
              ? prev
              : { ...prev, callId, peerId: from, status: "in-call" }
          );
          if (!pc) {
            console.warn("⚠️ call_answer 收到时本地 peerRef 还不存在");
            return;
          }
          if (content) {
            const remoteAnswer = JSON.parse(content);
            if (pc.signalingState !== "stable") {
              await pc.setRemoteDescription(
                new RTCSessionDescription(remoteAnswer)
              );
            }
          }
          break;
        }

        case "call_candidate": {
          if (from === me) return;
          if (!pc || !content) return;
          const ice = JSON.parse(content);
          try {
            await pc.addIceCandidate(new RTCIceCandidate(ice));
          } catch (err) {
            console.error("❌ addIceCandidate 失败", err);
          }
          break;
        }

        case "call_end": {
          setIncomingCall(null);
          cleanupCall();
          break;
        }
      }
    },
    [wsRef, userId, cleanupCall]
  );

  return {
    callState,
    incomingCall,
    localVideoRef,
    remoteVideoRef,
    startCall,
    handleCallSignal,
    endCall,
    acceptIncomingCall,
    rejectIncomingCall,
  };
}
