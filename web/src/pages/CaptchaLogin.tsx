import React, { useState } from "react";
import { useForm } from "react-hook-form";
import { Link, useNavigate } from 'react-router-dom';
import api from "../api/api";
import { setToken, setRefreshToken } from "../utils/session";

type CaptchaLoginForm = {
  email: string;
  code: string;
};

const CaptchaLogin: React.FC = () => {
  const { register, handleSubmit, watch, formState: { errors } } = useForm<CaptchaLoginForm>();
  const [countdown, setCountdown] = useState(0);
  const [sending, setSending] = useState(false);
  const navigate = useNavigate();

  const onSubmit = async (data: CaptchaLoginForm) => {
    try {
      const res = await api.emailCaptchaLogin({ email: data.email, code: data.code });
      const token = res.data?.token || res.data?.data?.token;
      const refresh = res.data?.refresh || res.data?.data?.refresh;
      if (token) {
        setToken(token);
        if (refresh) setRefreshToken(refresh);
        navigate("/chat");
      } else {
        alert("登录失败：未返回 Token");
      }
    } catch (err: any) {
      alert(err?.response?.data?.error || "验证码登录失败");
    }
  };





  const sendCode = async () => {
    const email = watch("email");
    if (!email) { alert("请先输入邮箱"); return; }
    if (countdown > 0) return;
    setSending(true);
    try {
      await api.sendEmailCaptcha({ email });
      setCountdown(60);
      const timer = setInterval(() => {
        setCountdown(prev => {
          if (prev <= 1) { clearInterval(timer); return 0; }
          return prev - 1;
        });
      }, 1000);
    } catch (err: any) {
      alert(err?.response?.data?.error || "发送失败，请稍后再试");
    } finally {
      setSending(false);
    }
  };

  return (
    <div
      className="h-screen w-screen bg-cover bg-center relative"
      style={{ backgroundImage: "url('/apex.png')" }}
    >
      <div className="absolute inset-0 bg-black/5 z-0"></div>

      <div className="relative z-10 flex items-center justify-center h-full">
        <div className="bg-white/40 rounded-xl shadow-xl p-8 w-96">
          <h2 className="text-center text-3xl font-bold text-gray-800 mb-6">验证码登录</h2>

          <form onSubmit={handleSubmit(onSubmit)} className="space-y-5">
            {/* 邮箱 */}
            <div>
              <input
                {...register("email", {
                  required: "请输入邮箱",
                  pattern: { value: /^[^\s@]+@[^\s@]+\.[^\s@]+$/, message: "邮箱格式不正确" }
                })}
                placeholder="邮箱"
                className="w-full rounded-lg border placeholder-black/40 text-black border-gray-300 bg-white/50 p-3 focus:outline-none focus:ring-2 focus:ring-blue-400"
              />
              {errors.email && <p className="text-red-500 text-sm mt-1">{errors.email.message}</p>}
            </div>

            {/* 验证码 + 发送按钮 */}
            <div className="flex space-x-3">
              <input
                {...register("code", { required: "请输入验证码" })}
                placeholder="验证码"
                className="flex-1 rounded-lg border placeholder-black/40 text-black border-gray-300 bg-white/50 p-3 focus:outline-none focus:ring-2 focus:ring-blue-400"
              />
              <button
                type="button"
                onClick={sendCode}
                disabled={countdown > 0 || sending}
                className={`w-20 rounded-lg font-semibold py-3 ${
                  countdown > 0 || sending
                    ? "bg-gray-400 text-white cursor-not-allowed"
                    : "bg-blue-500 hover:bg-blue-600 text-white"
                }`}
              >
                {countdown > 0 ? `${countdown}s` : sending ? "..." : "发送"}
              </button>
            </div>
            {errors.code && <p className="text-red-500 text-sm mt-1">{errors.code.message}</p>}

            {/* 登录按钮 */}
            <button
              type="submit"
              className="w-full bg-green-500 hover:bg-green-600 text-white font-semibold py-3 rounded-lg"
            >
              登录
            </button>
          </form>

          {/* 底部链接 */}
          <div className="mt-6 flex justify-end space-x-4">
            <Link to="/" className="text-blue-600 text-sm hover:underline">密码登录</Link>
            <Link to="/register" className="text-blue-600 text-sm hover:underline">注册</Link>
          </div>
        </div>
      </div>
    </div>
  );
};

export default CaptchaLogin;
