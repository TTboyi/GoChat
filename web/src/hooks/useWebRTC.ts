// ============================================================
// 文件：web/src/hooks/useWebRTC.ts
// 作用：封装整个音视频通话的状态机与信令逻辑，是前端最复杂的 Hook。
//
// 关键概念介绍：
//
// WebRTC（Web Real-Time Communication）：
//   浏览器内置的点对点音视频通信协议。一旦建立连接，媒体数据在
//   两端浏览器之间直接流动，不经过服务器（带宽零消耗）。
//
// SDP（Session Description Protocol，会话描述协议）：
//   描述"我支持什么编解码器、分辨率、码率"的文本协议。
//   Offer = 主叫方发起的 SDP；Answer = 被叫方回应的 SDP。
//   双方交换 Offer/Answer 后就知道"要用哪种编解码器通话"了。
//
// ICE Candidate（网络候选地址）：
//   因为用户的网络地址可能在 NAT/防火墙后面，WebRTC 通过
//   STUN 服务发现公网 IP，通过 TURN 服务转发（NAT 穿越失败时）。
//   每一个"我可以通过这个 IP+端口接收数据"就是一个 ICE Candidate。
//
// ICE Candidate 缓冲机制（本 Hook 的关键设计）：
//   问题：主叫方在 setLocalDescription 之后就立刻开始发 ICE Candidate，
//         而被叫方可能还没 setRemoteDescription（未知"Offer"是什么），
//         此时调用 addIceCandidate 会报错。
//   解决：pendingCandidatesRef 缓冲所有在 remoteDescReadyRef=false 期间
//         收到的 ICE Candidate，等 setRemoteDescription 完成后批量 flush。
//
// RTCPeerConnection 参数选择：
//   bundlePolicy="max-bundle"：把音频和视频复用同一个传输通道，
//     减少 ICE 协商次数（只需要找一对地址，而不是两对）。
//   iceCandidatePoolSize=4：提前预收集候选地址，减少建立连接的等待时间。
//
// VP9 优先的原因：
//   VP9 相比 VP8 在相同码率下画质更好（约省 50% 带宽）。
//   Chrome 和 Firefox 均支持，通过 setCodecPreferences 重排顺序即可。
//
// 码率控制（applyBitrate）：
//   浏览器默认码率较保守，通过 RTCRtpSender.setParameters 手动设置：
//   视频 2 Mbps + 音频 128 kbps，在局域网/宽带下能保证较高质量。
//   必须在连接 connected 后再设置，此时 encodings 才已初始化完毕。
//
// callStateRef + callState 双保险：
//   React state 的更新是异步的，在回调函数里直接读 callState 可能读到旧值。
//   callStateRef 始终是最新值的同步副本，用于在 WebSocket 回调里做判断。
// ============================================================
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
  offer: string;
}

// 视频通话媒体约束：720p / 30fps，音频开启回声消除和降噪
const VIDEO_CONSTRAINTS: MediaStreamConstraints = {
  video: {
    facingMode: "user",
    width: { ideal: 1280 },
    height: { ideal: 720 },
    frameRate: { ideal: 30 },
  },
  audio: {
    echoCancellation: true,
    noiseSuppression: true,
    autoGainControl: true,
  },
};

const AUDIO_CONSTRAINTS: MediaStreamConstraints = {
  audio: {
    echoCancellation: true,
    noiseSuppression: true,
    autoGainControl: true,
  },
};

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
    console.warn("⚠️ 无法获取 TURN 凭证，仅使用 STUN:", e);
  }
  return stunServers;
}

// createPeerConnection 统一创建 RTCPeerConnection，加上
// bundlePolicy 和 iceCandidatePoolSize 减少协商延迟。
function createPeerConnection(iceServers: RTCIceServer[]): RTCPeerConnection {
  return new RTCPeerConnection({
    iceServers,
    bundlePolicy: "max-bundle",    // 所有媒体共用一条传输，减少 ICE 协商轮次
    iceCandidatePoolSize: 4,       // 预先收集候选，缩短首次连接时间
  });
}

// preferCodec 将指定编解码器（如 VP9、H264）排到优先位置。
// 在 createOffer/createAnswer 前调用，对所有视频 transceiver 生效。
function preferCodec(pc: RTCPeerConnection, codecMime: string) {
  if (typeof RTCRtpSender.getCapabilities !== "function") return;
  pc.getTransceivers().forEach((transceiver) => {
    if (transceiver.sender.track?.kind !== "video") return;
    const caps = RTCRtpSender.getCapabilities("video");
    if (!caps) return;
    const preferred = caps.codecs.filter((c) =>
      c.mimeType.toLowerCase().includes(codecMime.toLowerCase())
    );
    const rest = caps.codecs.filter(
      (c) => !c.mimeType.toLowerCase().includes(codecMime.toLowerCase())
    );
    try {
      transceiver.setCodecPreferences([...preferred, ...rest]);
    } catch (e) {
      console.warn("setCodecPreferences 不支持:", e);
    }
  });
}

// applyBitrate 通过 RTCRtpSender.setParameters 设置码率上限。
// 视频 2Mbps / 音频 128kbps，比浏览器默认值高得多。
// 需在 setLocalDescription 之后调用，否则 encodings 可能还未初始化。
async function applyBitrate(pc: RTCPeerConnection) {
  for (const sender of pc.getSenders()) {
    const params = sender.getParameters();
    if (!params.encodings || params.encodings.length === 0) {
      params.encodings = [{}];
    }
    if (sender.track?.kind === "video") {
      params.encodings[0].maxBitrate = 2_000_000; // 2 Mbps
    } else if (sender.track?.kind === "audio") {
      params.encodings[0].maxBitrate = 128_000;   // 128 kbps
    }
    try {
      await sender.setParameters(params);
    } catch (e) {
      console.warn("setParameters 失败（部分浏览器在 offer 前不支持）:", e);
    }
  }
}

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

  const pendingCandidatesRef = useRef<RTCIceCandidateInit[]>([]);
  const remoteDescReadyRef = useRef(false);

  useEffect(() => {
    callStateRef.current = callState;
  }, [callState]);

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

  const startCall = useCallback(
    async (callType: "audio" | "video", active: SessionItem) => {
      const ws = wsRef.current;
      if (!ws || !active?.id) return;
      if (callStateRef.current.status !== "idle") return;

      const callId = Date.now().toString();

      let stream: MediaStream;
      try {
        stream = await navigator.mediaDevices.getUserMedia(
          callType === "video" ? VIDEO_CONSTRAINTS : AUDIO_CONSTRAINTS
        );
      } catch (err: any) {
        console.error("🚫 无法访问媒体设备:", err);
        alert("无法访问麦克风或摄像头，请检查浏览器权限。");
        return;
      }

      localStreamRef.current = stream;
      if (callType === "video") attachStream(localVideoRef, stream);

      setCallState({ callId, peerId: active.id, status: "ringing", callType, isCaller: true });

      const iceServers = await fetchIceServers();
      const pc = createPeerConnection(iceServers);
      peerRef.current = pc;
      remoteDescReadyRef.current = false;
      pendingCandidatesRef.current = [];

      stream.getTracks().forEach((t) => pc.addTrack(t, stream));

      // 优先使用 VP9（相同码率下画质优于 VP8）
      if (callType === "video") preferCodec(pc, "VP9");

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
          console.log(`🌐 ICE候选 [caller] type=${event.candidate.type}`);
          callCandidate(ws, active.id, callId, event.candidate);
        }
      };

      pc.oniceconnectionstatechange = () => {
        console.log("📡 ICE 状态:", pc.iceConnectionState);
        if (pc.iceConnectionState === "connected") {
          // 连接建立后再设置码率，此时 encodings 已就绪
          applyBitrate(pc);
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

  const endCall = useCallback(() => {
    const ws = wsRef.current;
    const { peerId, callId } = callStateRef.current;
    if (ws && peerId && callId) callEnd(ws, peerId, callId);
    cleanupCall();
  }, [wsRef, cleanupCall]);

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
      stream = await navigator.mediaDevices.getUserMedia(
        callType === "video" ? VIDEO_CONSTRAINTS : AUDIO_CONSTRAINTS
      );
    } catch {
      alert("无法访问摄像头/麦克风");
      socket.send({ action: "call_answer", receiveId: from, callId, accept: false });
      cleanupCall();
      return;
    }

    localStreamRef.current = stream;
    if (callType === "video") attachStream(localVideoRef, stream);

    const iceServers = await fetchIceServers();
    const pc2 = createPeerConnection(iceServers);
    peerRef.current = pc2;
    remoteDescReadyRef.current = false;

    stream.getTracks().forEach((t) => pc2.addTrack(t, stream));

    if (callType === "video") preferCodec(pc2, "VP9");

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
        console.log(`🌐 ICE候选 [callee] type=${event.candidate.type}`);
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
      if (pc2.iceConnectionState === "connected") {
        applyBitrate(pc2);
      }
    };

    const remoteOffer = JSON.parse(offerStr);
    await pc2.setRemoteDescription(new RTCSessionDescription(remoteOffer));

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
            remoteDescReadyRef.current = true;
            await flushPendingCandidates(pc);
          }
          break;
        }

        case "call_candidate": {
          if (from === me || !content) return;
          const ice = JSON.parse(content);
          if (!pc || !remoteDescReadyRef.current) {
            console.log("📦 缓冲 ICE 候选");
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
