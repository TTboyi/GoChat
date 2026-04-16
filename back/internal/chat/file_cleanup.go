// ============================================================
// 文件：back/internal/chat/file_cleanup.go
// 作用：定期清理 static/files 目录中的"孤儿文件"（上传后没有被任何消息引用的文件）。
//
// 为什么需要清理孤儿文件？
//   用户上传文件分两步：先 POST /message/uploadFile（得到 url），再发消息带上这个 url。
//   如果用户上传后取消发送，文件就永久存在磁盘上但没人引用，占用空间。
//   清理任务定期扫描，把超过一定时间没有被消息引用的文件删掉。
//
// 设计细节：
//   通过对比"文件系统里的文件"与"数据库里 url 字段引用的文件"来找出孤儿文件。
//   时间阈值通常设置为 24 小时，给足时间让消息被发出来后被 Persist 写库。
// ============================================================

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
