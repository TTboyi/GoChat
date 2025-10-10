import React, { useState } from "react";
import { useForm } from "react-hook-form";
import { Link } from 'react-router-dom';

type CaptchaLoginForm = {
  phone: string;
  code: string;
};

const CaptchaLogin: React.FC = () => {
  const { register, handleSubmit, formState: { errors } } = useForm<CaptchaLoginForm>();
  const [countdown, setCountdown] = useState(0);

  const onSubmit = (data: CaptchaLoginForm) => {
    console.log("验证码登录信息：", data);
  };

  const sendCode = () => {
    if (countdown > 0) return; // 倒计时期间禁止点击
    console.log("发送验证码到手机号");
    setCountdown(60);
    const timer = setInterval(() => {
      setCountdown(prev => {
        if (prev <= 1) {
          clearInterval(timer);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
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
            {/* 手机号 */}
            <div>
              <input
                {...register("phone", { 
                  required: "请输入手机号",
                  pattern: { value: /^1[3-9]\d{9}$/, message: "手机号格式不正确" }
                })}
                placeholder="手机号"
                className="w-full rounded-lg border placeholder-black/40 text-black border-gray-300 bg-white/50 p-3 focus:outline-none focus:ring-2 focus:ring-blue-400"
              />
              {errors.phone && <p className="text-red-500 text-sm mt-1">{errors.phone.message}</p>}
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
                disabled={countdown > 0}
                className={`w-20 rounded-lg font-semibold py-3 ${
                  countdown > 0
                    ? "bg-gray-400 text-white cursor-not-allowed"
                    : "bg-blue-500 hover:bg-blue-600 text-white"
                }`}
              >
                {countdown > 0 ? `${countdown}s` : "发送"}
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
