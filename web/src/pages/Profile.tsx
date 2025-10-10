import React, { useEffect, useState } from "react";
import api from "../api/api";

interface UserInfo {
  uuid: string;
  nickname: string;
  avatar: string;
  signature?: string;
  telephone?: string;
}

const Profile: React.FC = () => {
  const [userInfo, setUserInfo] = useState<UserInfo | null>(null);
  const [nickname, setNickname] = useState("");
  const [signature, setSignature] = useState("");
  const [loading, setLoading] = useState(false);
  const [avatarUploading, setAvatarUploading] = useState(false);

  // 获取用户信息
  const loadUserInfo = async () => {
    try {
      const res = await api.getUserInfo();
      const data = res.data?.data || res.data;
      if (data) {
        setUserInfo(data);
        setNickname(data.nickname);
        setSignature(data.signature || "");
      }
    } catch (err) {
      console.error("获取用户信息失败:", err);
      alert("获取用户信息失败，请重新登录");
    }
  };

  useEffect(() => {
    loadUserInfo();
  }, []);

  // 上传头像
  const handleAvatarUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    const formData = new FormData();
    formData.append("file", file);
    setAvatarUploading(true);

    try {
      const { avatarUrl } = await api.uploadAvatar(formData);
      if (avatarUrl) {
        setUserInfo((prev) => (prev ? { ...prev, avatar: avatarUrl } : null));
        alert("头像上传成功！");
      }
    } catch (err) {
      console.error("头像上传失败:", err);
      alert("头像上传失败，请重试");
    } finally {
      setAvatarUploading(false);
    }
  };

  // 保存资料
  const handleSave = async () => {
    if (!userInfo) return;
    setLoading(true);
    try {
      const payload = {
        uuid: userInfo.uuid,
        nickname,
        signature,
      };
      const res = await api.updateUser(payload);
      if (res.data?.code === 0 || res.status === 200) {
        alert("资料已更新");
      } else {
        alert("更新失败");
      }
    } catch (err) {
      console.error("更新资料失败:", err);
      alert("更新失败，请稍后再试");
    } finally {
      setLoading(false);
    }
  };

  if (!userInfo)
    return (
      <div className="flex justify-center items-center h-screen">
        <p className="text-gray-500">加载中...</p>
      </div>
    );

  return (
    <div className="h-screen flex justify-center items-center bg-gradient-to-br from-blue-100 to-gray-100">
      <div className="bg-white/70 backdrop-blur-md rounded-xl shadow-xl p-8 w-96">
        <h2 className="text-center text-2xl font-bold text-gray-800 mb-6">
          个人资料
        </h2>

        {/* 头像部分 */}
        <div className="flex flex-col items-center mb-6">
          <img
            src={
              userInfo.avatar
                ? `http://localhost:8080${userInfo.avatar}`
                : "https://via.placeholder.com/100"
            }
            alt="头像"
            className="w-24 h-24 rounded-full border object-cover mb-3"
          />
          <label className="cursor-pointer text-blue-600 hover:underline text-sm">
            {avatarUploading ? "上传中..." : "更换头像"}
            <input
              type="file"
              accept="image/*"
              onChange={handleAvatarUpload}
              hidden
            />
          </label>
        </div>

        {/* 昵称 */}
        <div className="mb-4">
          <label className="block text-sm font-medium text-gray-700 mb-1">
            昵称
          </label>
          <input
            value={nickname}
            onChange={(e) => setNickname(e.target.value)}
            className="w-full border border-gray-300 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-400 bg-white/80 text-black"
          />
        </div>

        {/* 个性签名 */}
        <div className="mb-6">
          <label className="block text-sm font-medium text-gray-700 mb-1">
            个性签名
          </label>
          <textarea
            value={signature}
            onChange={(e) => setSignature(e.target.value)}
            rows={2}
            placeholder="写点什么吧..."
            className="w-full border border-gray-300 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-400 bg-white/80 text-black"
          />
        </div>

        {/* 保存按钮 */}
        <button
          onClick={handleSave}
          disabled={loading}
          className="w-full bg-blue-500 hover:bg-blue-600 text-white font-semibold py-3 rounded-lg"
        >
          {loading ? "保存中..." : "保存修改"}
        </button>
      </div>
    </div>
  );
};

export default Profile;
