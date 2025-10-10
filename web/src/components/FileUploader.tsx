import React, { useRef, useState } from "react";
import api from "../api/api";

interface FileUploaderProps {
  onUploadSuccess: (url: string, type: string) => void;
}

const FileUploader: React.FC<FileUploaderProps> = ({ onUploadSuccess }) => {
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const [uploading, setUploading] = useState(false);

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    const formData = new FormData();
    formData.append("file", file);
    setUploading(true);

    try {
      const res = await api.uploadFile(formData);

      // ✅ 根据你的后端返回结构提取完整 URL
      const relativeUrl = res.data?.url;
      const fileUrl = relativeUrl
        ? `http://localhost:8000${relativeUrl}`
        : "";

      if (fileUrl) {
        const type = file.type.startsWith("image/") ? "image" : "file";
        onUploadSuccess(fileUrl, type);
      } else {
        alert("上传失败：未返回文件URL");
      }
    } catch (err) {
      console.error("文件上传失败:", err);
      alert("上传失败，请重试");
    } finally {
      setUploading(false);
    }
  };

  return (
    <>
      <input
        ref={fileInputRef}
        type="file"
        hidden
        onChange={handleFileChange}
      />
      <button
        onClick={() => fileInputRef.current?.click()}
        disabled={uploading}
        className="bg-gray-200 hover:bg-gray-300 text-gray-800 font-semibold px-3 py-2 rounded-lg"
      >
        {uploading ? "上传中..." : "📎 文件"}
      </button>
    </>
  );
};

export default FileUploader;
