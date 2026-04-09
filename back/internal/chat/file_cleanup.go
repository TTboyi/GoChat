package chat

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const fileExpireDuration = time.Hour

// StartFileCleanup 定期清理超过1小时的上传文件
func StartFileCleanup(staticFilePath string) {
	go func() {
		// 启动时先清理一次历史遗留
		cleanExpiredFiles(staticFilePath)

		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			cleanExpiredFiles(staticFilePath)
		}
	}()
}

func cleanExpiredFiles(dir string) {
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		// 只清理 file_ 前缀的上传文件，不清理其他静态资源
		if !strings.HasPrefix(name, "file_") {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if time.Since(info.ModTime()) > fileExpireDuration {
			if err := os.Remove(path); err == nil {
				log.Printf("🗑️ 已删除过期文件: %s", name)
			}
		}
		return nil
	})
}
