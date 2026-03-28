package utils

import (
	"fmt"
	"mime/multipart"
	"strings"
)

//计算机存储单位：Byte、KB、MB、GB、TB、PB、EB
//int64最大支持EB
const (
	B int64 = 1 << (10 * iota)
	KB
	MB
	GB
	TB
	PB
	EB
)

//Human 友好的显示
func Human(size int64) string {
	h := "EB"
	switch {
	case size < KB:
		h = "B"
	case size < MB:
		h = "KB"
	case size < GB:
		h = "MB"
	case size < TB:
		h = "GB"
	case size < PB:
		h = "TB"
	case size < EB:
		h = "PB"
	}
	return fmt.Sprintf("%d%s", size, h)
}

//ValidateImage 验证图片
func ValidateImage(f *multipart.FileHeader, maxSize int64) (msg string, ok bool) {
	if f == nil {
		return "上传图片不能为空", false
	}

	//验证配置参数
	if maxSize == 0 || maxSize > 4*GB {
		return "配置参数错误,单个文件大小: 1B-4GB", false
	}

	//验证文件大小
	if f.Size > maxSize {
		return fmt.Sprintf("单个文件大小不能超过: %s, 当前文件[%s]: %s, ", Human(maxSize), f.Filename, Human(f.Size)), false
	}

	//判断文件类型
	fileName := strings.ToLower(f.Filename)
	if !strings.Contains(fileName, ".") {
		return fmt.Sprintf("无法识别的文件格式: %s", fileName), false
	}

	fileNameArr := strings.Split(fileName, ".")
	fileType := strings.ToLower(fileNameArr[len(fileNameArr)-1])
	if !ValidateFileType(fileType) {
		return fmt.Sprintf("错误的文件格式: %s", fileName), false
	}

	//contentType := f.Header.Get("Content-Type")
	//if !strings.Contains(strings.ToLower(contentType), "image") {
	//	return fmt.Sprintf("无法识别的文件格式: %s", contentType), false
	//}
	return "", true
}

//ValidateFileType 验证文件类型
func ValidateFileType(fileType string) bool {
	fileTypes := make([]string, 0)
	//
	fileTypes = append(fileTypes, "jpg", "gif", "png", "jpeg", "bmp", "jpe", "psd")
	for _, v := range fileTypes {
		if fileType == v {
			return true
		}
	}
	return false
}
