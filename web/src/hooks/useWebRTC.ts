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

export function useWebRTC(
  wsRef: RefObject<ChatWebSocket | null>,
  userId: string | undefined
) {
  const [callState, setCallState] = useState<CallState>(IDLE_CALL_STATE);
  const callStateRef = useRef<CallState>(IDLE_CALL_STATE);

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

  const startCall = useCallback(
    async (callType: "audio" | "video", active: SessionItem) => {
      const ws = wsRef.current;
      if (!ws || !active?.id) return;

      const callId = Date.now().toString();
      setCallState({
        callId,
        peerId: active.id,
        status: "ringing",
        callType,
        isCaller: true,
      });

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
      if (localVideoRef.current && callType === "video") {
        localVideoRef.current.srcObject = stream;
      }

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

      alert(
        `📞 正在呼叫 ${active.name}（${
          callType === "video" ? "视频" : "语音"
        }通话）`
      );
    },
    [wsRef]
  );

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
            socket?.send({
              action: "call_answer",
              receiveId: from,
              callId,
              accept: false,
            });
            return;
          }
          const ok = window.confirm(
            `📞 ${from} 发起${
              callType === "video" ? "视频" : "语音"
            }通话，是否接听？`
          );
          if (!ok) {
            socket?.send({
              action: "call_answer",
              receiveId: from,
              callId,
              accept: false,
            });
            return;
          }

          setCallState({
            callId,
            peerId: from,
            status: "in-call",
            callType,
            isCaller: false,
          });

          const constraints =
            callType === "video"
              ? { video: true, audio: true }
              : { audio: true };
          const stream = await navigator.mediaDevices.getUserMedia(constraints);
          localStreamRef.current = stream;
          if (localVideoRef.current) localVideoRef.current.srcObject = stream;

          const pc2 = new RTCPeerConnection({
            iceServers: [{ urls: "stun:stun.l.google.com:19302" }],
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
              socket?.send({
                action: "call_candidate",
                receiveId: from,
                callId,
                content: JSON.stringify(event.candidate),
              });
            }
          };

          if (!content) {
            console.warn("⚠️ call_invite 没带 offer content");
            return;
          }
          const remoteOffer = JSON.parse(content);
          await pc2.setRemoteDescription(
            new RTCSessionDescription(remoteOffer)
          );
          const answer = await pc2.createAnswer();
          await pc2.setLocalDescription(answer);

          socket?.send({
            action: "call_answer",
            receiveId: from,
            callId,
            accept: true,
            content: JSON.stringify(answer),
          });
          break;
        }

        case "call_answer": {
          if (from === me) return;
          if (accept === false) {
            alert("🚫 对方拒绝通话");
            cleanupCall();
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
          alert("📴 通话结束");
          cleanupCall();
          break;
        }
      }
    },
    [wsRef, userId, cleanupCall]
  );

  const endCall = useCallback(() => {
    const ws = wsRef.current;
    const { peerId, callId } = callStateRef.current;
    if (ws && peerId && callId) {
      callEnd(ws, peerId, callId);
    }
    cleanupCall();
  }, [wsRef, cleanupCall]);

  return {
    callState,
    localVideoRef,
    remoteVideoRef,
    startCall,
    handleCallSignal,
    endCall,
  };
}
