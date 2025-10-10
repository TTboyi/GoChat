import React from "react";
import { useForm } from "react-hook-form";
import { Link } from 'react-router-dom';
import { useNavigate } from "react-router-dom";
import api from "../api/api";
import { setToken } from "../utils/session";


type LoginForm = {
  nickname: string;
  password: string;
};

const Login: React.FC = () => {
  const { register, handleSubmit, formState: { errors } } = useForm<LoginForm>();

  const navigate = useNavigate();

  const onSubmit = async (data: LoginForm) => {
    try {
      const res = await api.login(data);
      const token = res.data?.token || res.data?.data?.token;
    if (token) {
      setToken(token);
      alert("登录成功！");
      navigate("/chat");
    } else {
      alert("登录失败：未返回 Token");
    }
      } catch (err) {
      alert("登录失败，请检查账号或密码");
    }
  };

  return (
    <div
      className="h-screen w-screen bg-cover bg-center relative"
      style={{ backgroundImage: "url('/apex.png')" }}
    >
      {/* 可选的深色背景蒙层 */}
      <div className="absolute inset-0 bg-black/5  z-0"></div>

      {/* 登录卡片容器 */}
      <div className="relative z-10 flex items-center justify-center h-full">
        <div className=" bg-white/40  rounded-xl shadow-xl p-8">
          <h2 className="text-center text-3xl font-bold text-gray-800 mb-6">登录</h2>

          <form onSubmit={handleSubmit(onSubmit)} className="space-y-5">
            <div>
              <input
                {...register("nickname", { required: "请输入账号" })}
                placeholder="账号"
                className="w-full rounded-lg border placeholder-black/40 text-black border-gray-300 bg-white/50 p-3 focus:outline-none focus:ring-2 focus:ring-blue-400"
              />
              {errors.nickname && <p className="text-red-500 text-sm mt-1">{errors.nickname.message}</p>}
            </div>

            <div>
              <input
                type="password"
                {...register("password", { required: "请输入密码" })}
                placeholder="密码"
                className="w-full rounded-lg border  placeholder-black/40 text-black border-gray-300 bg-white/50 p-3 focus:outline-none focus:ring-2 focus:ring-blue-400"
              />
              {errors.password && <p className="text-red-500 text-sm mt-1">{errors.password.message}</p>}
            </div>

            <button
              type="submit"
              className="w-full bg-blue-500 hover:bg-blue-600 text-white font-semibold py-3 rounded-lg"
            >
              登录
            </button>
          </form>

          {/* 右下角超链接 */}
          <div className="mt-6 flex justify-end  space-x-4">
            <Link to="/captcha-login" className="text-blue-600 text-sm hover:underline">验证码登录</Link>
            <Link to="/register" className="text-blue-600 hover:underline text-sm">注册</Link>

          </div>
        </div>
      </div>
      
    </div>
    
  );
};

export default Login;
