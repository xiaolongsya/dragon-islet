package handler

import (
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	_ "image/gif"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dragon-islet/internal/global"

	"github.com/gin-gonic/gin"
)

type UploadHandler struct{}

func (h *UploadHandler) UploadImage(c *gin.Context) {
	// 限制 5MB
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 5<<20)

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "上传失败或文件超过5MB"})
		return
	}
	defer file.Close()

	// 解码图片 (支持 jpg, png, gif)
	img, _, err := image.Decode(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的图片格式"})
		return
	}

	// 读取配置
	saveDir := global.CONFIG.GetString("upload.save_path")
	if saveDir == "" {
		saveDir = "./uploads"
	}
	baseURL := global.CONFIG.GetString("upload.base_url")
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://%s/uploads", c.Request.Host)
	}

	// 确保上传目录存在
	err = os.MkdirAll(saveDir, 0755)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建目录失败"})
		return
	}

	// 生成文件名
	filename := fmt.Sprintf("%d.jpg", time.Now().UnixNano())
	savePath := filepath.Join(saveDir, filename)

	out, err := os.Create(savePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "保存图片失败"})
		return
	}
	defer out.Close()

	// 压缩并以 JPEG 格式保存 (质量设为 60)
	var opt jpeg.Options
	opt.Quality = 60
	err = jpeg.Encode(out, img, &opt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "图片压缩失败"})
		return
	}

	// 返回访问 URL
	url := fmt.Sprintf("%s/%s", strings.TrimRight(baseURL, "/"), filename)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"url": url}})
}
