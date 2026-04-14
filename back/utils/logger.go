package utils

import (
	"io"
	"log/slog"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

// AppLogger 是全局结构化日志器，项目所有包统一使用它。
// 初始化前调用 InitLogger；在 main.go 里 LoadConfig 之后立即调用。
var AppLogger *slog.Logger

// InitLogger 根据配置初始化 AppLogger。
// logPath 为空时只写 stdout；非空时同时写 stdout（文本格式）和日志文件（JSON 格式）。
func InitLogger(logPath string, maxSizeMB, maxAgeDays, maxBackups int) {
	var writer io.Writer

	if logPath != "" {
		// 确保日志目录存在
		if err := os.MkdirAll(dirOf(logPath), 0755); err != nil {
			slog.Warn("无法创建日志目录，回退到 stdout", "err", err)
			writer = os.Stdout
		} else {
			rotator := &lumberjack.Logger{
				Filename:   logPath,
				MaxSize:    maxSizeMB,
				MaxAge:     maxAgeDays,
				MaxBackups: maxBackups,
				Compress:   true,
			}
			// 双输出：文件 JSON（易检索）+ stdout 文本（易阅读）
			writer = io.MultiWriter(rotator, os.Stdout)
		}
	} else {
		writer = os.Stdout
	}

	AppLogger = slog.New(slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		// 在 JSON 日志中输出完整来源文件，方便定位问题
		AddSource: false,
	}))

	slog.SetDefault(AppLogger)
}

// dirOf 返回路径的目录部分（不依赖 filepath，避免多余 import）。
func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}
