import { useRef, useState, useCallback, useEffect } from "react";
import type { RefObject } from "react";
import type { CallState, SessionItem } from "../types/chat";
import type { ChatWebSocket } from "../api/socket";
import { callCandidate, callEnd } from "../api/socket";
import api from "../api/api";

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

// ✅ 从后端获取动态 TURN 凭证，构建 ICE 服务器列表
async function fetchIceServers(): Promise<RTCIceServer[]> {
  // 基础 STUN（总是可用）
  const stunServers: RTCIceServer[] = [
    { urls: "stun:stun1.l.google.com:19302" },
    { urls: "stun:stun2.l.google.com:19302" },
  ];

  try {
    const res = await api.getTurnCredentials();
    const { username, password, uris } = res.data;
    if (username && password && Array.isArray(uris)) {
      return [
        ...stunServers,
        {
          urls: uris,
          username,
          credential: password,
        },
      ];
    }
  } catch (e) {
    console.warn("⚠️ 无法获取 TURN 凭证，仅使用 STUN（跨网络通话可能失败）:", e);
  }

  return stunServers;
}

// ✅ 将媒体流挂载到 video 元素（含重试保障）
function attachStream(
  videoRef: RefObject<HTMLVideoElement | null>,
  stream: MediaStream,
  retries = 5
) {
  const tryAttach = (remaining: number) => {
    if (videoRef.current) {
      videoRef.current.srcObject = stream;
      videoRef.current.play().catch(() => {}); // 忽略自动播放策略错误
      return;
    }
    if (remaining > 0) {
      setTimeout(() => tryAttach(remaining - 1), 80);
    }
  };
  tryAttach(retries);
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

  // ✅ 主叫：发起通话
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
            ? { video: { facingMode: "user" }, audio: true }
            : { audio: true };
        stream = await navigator.mediaDevices.getUserMedia(constraints);
      } catch (err: any) {
        console.error("🚫 无法访问麦克风或摄像头:", err);
        alert("无法访问麦克风或摄像头，请检查浏览器权限。");
        return;
      }

      localStreamRef.current = stream;

      // ✅ 立即挂载本地视频（不等 callState 变化）
      if (callType === "video") {
        attachStream(localVideoRef, stream);
      }

      // 设置状态（触发 CallWindow 渲染）
      setCallState({
        callId,
        peerId: active.id,
        status: "ringing",
        callType,
        isCaller: true,
      });

      // ✅ 获取动态 TURN 凭证
      const iceServers = await fetchIceServers();
      const pc = new RTCPeerConnection({ iceServers });
      peerRef.current = pc;

      // 添加本地轨道
      stream.getTracks().forEach((t) => pc.addTrack(t, stream));

      pc.ontrack = (event) => {
        const [remoteStream] = event.streams;
        if (callType === "video") {
          attachStream(remoteVideoRef, remoteStream);
        } else {
          // 语音通话：动态创建 audio 元素播放远端流
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

      pc.oniceconnectionstatechange = () => {
        console.log("📡 ICE 状态:", pc.iceConnectionState);
        if (
          pc.iceConnectionState === "disconnected" ||
          pc.iceConnectionState === "failed"
        ) {
          console.warn("⚠️ ICE 连接失败或断开");
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

  // ✅ 接听来电（被叫）
  const acceptIncomingCall = useCallback(async () => {
    const pending = incomingCall;
    if (!pending) return;
    const socket = wsRef.current;
    if (!socket) return;

    setIncomingCall(null);

    const { callId, from, callType, offer: offerStr } = pending;

    // 先更新状态（触发 CallWindow 渲染）
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
        callType === "video"
          ? { video: { facingMode: "user" }, audio: true }
          : { audio: true };
      stream = await navigator.mediaDevices.getUserMedia(constraints);
    } catch {
      alert("无法访问摄像头/麦克风");
      socket.send({ action: "call_answer", receiveId: from, callId, accept: false });
      cleanupCall();
      return;
    }

    localStreamRef.current = stream;

    // ✅ 被叫立即挂载本地视频（修复移动端/接收方看不到自己画面的 bug）
    if (callType === "video") {
      attachStream(localVideoRef, stream);
    }

    // ✅ 获取动态 TURN 凭证
    const iceServers = await fetchIceServers();
    const pc2 = new RTCPeerConnection({ iceServers });
    peerRef.current = pc2;

    // 添加本地所有轨道
    stream.getTracks().forEach((t) => pc2.addTrack(t, stream));

    pc2.ontrack = (event) => {
      const [remoteStream] = event.streams;
      if (callType === "video") {
        attachStream(remoteVideoRef, remoteStream);
      } else {
        const audioEl = document.createElement("audio");
        audioEl.srcObject = remoteStream;
        audioEl.autoplay = true;
        document.body.appendChild(audioEl);
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

    pc2.oniceconnectionstatechange = () => {
      console.log("📡 ICE 状态（被叫）:", pc2.iceConnectionState);
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
