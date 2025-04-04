//go:build linux

package internal

import (
	_ "embed"
	utils "github.com/mzky/utils/common"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

//go:embed linux/MP4Box
var mp4Box []byte

func GetMP4Box() string {
	mp4boxPath := filepath.Join(os.TempDir(), "MP4Box") // 指定ffmpeg路径
	if !utils.IsExist(mp4boxPath) {
		logrus.Info("第一次运行,自动释放MP4Box")
		if err := os.WriteFile(mp4boxPath, mp4Box, os.ModePerm); err != nil {
			logrus.Error(err)
			logrus.Fatal("释放MP4Box失败,查看文件权限是否正常")
		}
	}
	return mp4boxPath
}
