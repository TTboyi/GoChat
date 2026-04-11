import { useRef, useState, useCallback, useEffect } from "react";
import type { RefObject } from "react";
import type { CallState, SessionItem } from "../types/chat";
import type { ChatWebSocket } from "../api/socket";
import { callCandidate, callEnd } from "../api/socket";
import api from "../api/api";

// 空闲态是这个 Hook 的“复位快照”，通话结束后会回到这里。
const IDLE_CALL_STATE: CallState = {
  callId: null,
  peerId: null,
  status: "idle",
  callType: null,
  isCaller: false,
};

// IncomingCall 描述来电弹窗真正需要的信息。
export interface IncomingCall {
  callId: string;
  from: string;
  fromName?: string;
  callType: "audio" | "video";
  offer: string;
}

// fetchIceServers 先准备公开 STUN，再尝试向后端要动态 TURN 凭证。
// 这样就算 TURN 服务暂时不可用，局域网/简单网络条件下仍有机会建立直连。
async function fetchIceServers(): Promise<RTCIceServer[]> {
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
        { urls: uris, username, credential: password },
      ];
    }
  } catch (e) {
    console.warn("⚠️ 无法获取 TURN 凭证，仅使用 STUN（跨网络通话可能失败）:", e);
  }
  return stunServers;
}

// attachStream 负责把 MediaStream 挂到 video/audio 元素上。
// 之所以带重试，是因为 React 渲染和 WebRTC 回调并不是严格同步的：
// ontrack 触发时，video 节点未必已经出现在 DOM 中。
function attachStream(
  videoRef: RefObject<HTMLVideoElement | null>,
  stream: MediaStream,
  retries = 8
) {
  const tryAttach = (remaining: number) => {
    if (videoRef.current) {
      videoRef.current.srcObject = stream;
      videoRef.current.play().catch(() => {});
      return;
    }
    if (remaining > 0) setTimeout(() => tryAttach(remaining - 1), 100);
  };
  tryAttach(retries);
}

// useWebRTC 把一整套“音视频通话状态机”封装进 Hook。
// 页面只需要提供当前 WS 引用和 userId，就能得到：
// - 发起/接听/拒绝/挂断通话的方法
// - 本地/远端视频节点引用
// - 当前通话状态和来电信息
export function useWebRTC(
  wsRef: RefObject<ChatWebSocket | null>,
  userId: string | undefined
) {
  const [callState, setCallState] = useState<CallState>(IDLE_CALL_STATE);
  const callStateRef = useRef<CallState>(IDLE_CALL_STATE);

  const [incomingCall, setIncomingCall] = useState<IncomingCall | null>(null);

  const localVideoRef = useRef<HTMLVideoElement>(null);
  const remoteVideoRef = useRef<HTMLVideoElement>(null);
  const peerRef = useRef<RTCPeerConnection | null>(null);
  const localStreamRef = useRef<MediaStream | null>(null);

  // ICE candidate 可能早于 RemoteDescription 到达。
  // 如果直接 addIceCandidate，会因为时序不对而失败，所以先暂存，等时机成熟再 flush。
  const pendingCandidatesRef = useRef<RTCIceCandidateInit[]>([]);
  // 只有 setRemoteDescription 完成后，addIceCandidate 才是安全的。
  const remoteDescReadyRef = useRef(false);

  useEffect(() => {
    callStateRef.current = callState;
  }, [callState]);

  // flushPendingCandidates 把前面积压的 ICE 候选补到 PeerConnection 上。
  const flushPendingCandidates = useCallback(async (pc: RTCPeerConnection) => {
    const pending = [...pendingCandidatesRef.current];
    pendingCandidatesRef.current = [];
    console.log(`📦 flush ${pending.length} 个缓冲 ICE 候选`);
    for (const cand of pending) {
      try {
        await pc.addIceCandidate(new RTCIceCandidate(cand));
      } catch (err) {
        console.warn("⚠️ flush addIceCandidate 失败:", err);
      }
    }
  }, []);

  // cleanupCall 用于“彻底收尾”：清状态、关连接、停本地采集、清空视频元素。
  const cleanupCall = useCallback(() => {
    setCallState(IDLE_CALL_STATE);
    pendingCandidatesRef.current = [];
    remoteDescReadyRef.current = false;
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

  // startCall 是主叫路径：
  // 1. 申请媒体权限；
  // 2. 建立 RTCPeerConnection；
  // 3. 生成 offer；
  // 4. 通过 WebSocket 把 offer 发给对方。
  const startCall = useCallback(
    async (callType: "audio" | "video", active: SessionItem) => {
      const ws = wsRef.current;
      if (!ws || !active?.id) return;
      // 防止重复发起
      if (callStateRef.current.status !== "idle") return;

      const callId = Date.now().toString();

      let stream: MediaStream;
      try {
        const constraints =
          callType === "video"
            ? { video: { facingMode: "user" }, audio: true }
            : { audio: true };
        stream = await navigator.mediaDevices.getUserMedia(constraints);
      } catch (err: any) {
        console.error("🚫 无法访问媒体设备:", err);
        alert("无法访问麦克风或摄像头，请检查浏览器权限。");
        return;
      }

      localStreamRef.current = stream;
      if (callType === "video") attachStream(localVideoRef, stream);

      setCallState({ callId, peerId: active.id, status: "ringing", callType, isCaller: true });

      const iceServers = await fetchIceServers();
      const pc = new RTCPeerConnection({ iceServers });
      peerRef.current = pc;
      remoteDescReadyRef.current = false;
      pendingCandidatesRef.current = [];

      stream.getTracks().forEach((t) => pc.addTrack(t, stream));

      pc.ontrack = (event) => {
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

      pc.onicecandidate = (event) => {
        if (event.candidate) {
          console.log(`🌐 ICE候选 [caller] type=${event.candidate.type} protocol=${event.candidate.protocol} address=${event.candidate.address}`);
          callCandidate(ws, active.id, callId, event.candidate);
        } else {
          console.log("🌐 ICE候选收集完毕 [caller]");
        }
      };

      pc.oniceconnectionstatechange = () => {
        console.log("📡 ICE 状态:", pc.iceConnectionState);
        if (pc.iceConnectionState === "disconnected" || pc.iceConnectionState === "failed") {
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

  // endCall 既通知对端挂断，也做本地清理。
  const endCall = useCallback(() => {
    const ws = wsRef.current;
    const { peerId, callId } = callStateRef.current;
    if (ws && peerId && callId) callEnd(ws, peerId, callId);
    cleanupCall();
  }, [wsRef, cleanupCall]);

  // acceptIncomingCall 是被叫路径：
  // 先设置远端 offer，再创建 answer 回给主叫。
  const acceptIncomingCall = useCallback(async () => {
    const pending = incomingCall;
    if (!pending) return;
    const socket = wsRef.current;
    if (!socket) return;

    setIncomingCall(null);
    const { callId, from, callType, offer: offerStr } = pending;

    setCallState({ callId, peerId: from, status: "in-call", callType, isCaller: false });

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
    if (callType === "video") attachStream(localVideoRef, stream);

    const iceServers = await fetchIceServers();
    const pc2 = new RTCPeerConnection({ iceServers });
    peerRef.current = pc2;
    remoteDescReadyRef.current = false;

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
        console.log(`🌐 ICE候选 [callee] type=${event.candidate.type} protocol=${event.candidate.protocol} address=${event.candidate.address}`);
        socket.send({
          action: "call_candidate",
          receiveId: from,
          callId,
          content: JSON.stringify(event.candidate),
        });
      } else {
        console.log("🌐 ICE候选收集完毕 [callee]");
      }
    };

    pc2.oniceconnectionstatechange = () => {
      console.log("📡 ICE 状态（被叫）:", pc2.iceConnectionState);
    };

    const remoteOffer = JSON.parse(offerStr);
    await pc2.setRemoteDescription(new RTCSessionDescription(remoteOffer));

    // ✅ remote description 设置完毕 → flush 缓冲候选
    remoteDescReadyRef.current = true;
    await flushPendingCandidates(pc2);

    const answer = await pc2.createAnswer();
    await pc2.setLocalDescription(answer);

    socket.send({
      action: "call_answer",
      receiveId: from,
      callId,
      accept: true,
      content: JSON.stringify(answer),
    });
  }, [incomingCall, wsRef, cleanupCall, flushPendingCandidates]);

  const rejectIncomingCall = useCallback(() => {
    const pending = incomingCall;
    if (!pending) return;
    wsRef.current?.send({
      action: "call_answer",
      receiveId: pending.from,
      callId: pending.callId,
      accept: false,
    });
    setIncomingCall(null);
  }, [incomingCall, wsRef]);

  const handleCallSignal = useCallback(
    async (msg: any) => {
      // handleCallSignal 是整个 Hook 的“事件分发器”。
      // Chat 页面收到所有 call_* WebSocket 消息后，都会交给这里处理。
      console.log("📨 收到信令:", msg);
      const { action, from, callType, callId, accept, content } = msg;
      const pc = peerRef.current;
      const socket = wsRef.current;
      const me = userId;

      switch (action) {
        case "call_invite": {
          if (from === me) return;
          if (callStateRef.current.status !== "idle") {
            socket?.send({ action: "call_answer", receiveId: from, callId, accept: false });
            return;
          }
          setIncomingCall({ callId, from, callType, offer: content || "" });
          break;
        }

        case "call_answer": {
          if (from === me) return;
          if (accept === false) {
            cleanupCall();
            alert("🚫 对方拒绝通话");
            return;
          }
          setCallState((prev) =>
            prev.status === "in-call" ? prev : { ...prev, callId, peerId: from, status: "in-call" }
          );
          if (!pc || !content) return;
          if (pc.signalingState !== "stable") {
            const remoteAnswer = JSON.parse(content);
            await pc.setRemoteDescription(new RTCSessionDescription(remoteAnswer));
            // ✅ remote description 设置完毕 → flush 缓冲候选
            remoteDescReadyRef.current = true;
            await flushPendingCandidates(pc);
          }
          break;
        }

        case "call_candidate": {
          if (from === me || !content) return;
          const ice = JSON.parse(content);

          // 如果此时 PeerConnection 还没准备好，先缓冲，避免因为时序问题丢候选。
          if (!pc || !remoteDescReadyRef.current) {
            console.log("📦 缓冲 ICE 候选（PC 或 remoteDesc 未就绪）");
            pendingCandidatesRef.current.push(ice);
            return;
          }

          try {
            await pc.addIceCandidate(new RTCIceCandidate(ice));
          } catch (err) {
            console.warn("⚠️ addIceCandidate 失败:", err);
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
    [wsRef, userId, cleanupCall, flushPendingCandidates]
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
