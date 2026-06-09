package upload

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Uploader struct {
	BasePath string // filesystem path, e.g. ./static/upload
	BaseURL  string // URL prefix, e.g. /static/upload
	MaxSize  int64  // bytes
}

func NewUploader(basePath, baseURL string, maxSizeMB int64) *Uploader {
	return &Uploader{
		BasePath: basePath,
		BaseURL:  baseURL,
		MaxSize:  maxSizeMB * 1024 * 1024,
	}
}

func (u *Uploader) SaveImage(file *multipart.FileHeader) (string, error) {
	if file.Size > u.MaxSize {
		return "", fmt.Errorf("file too large, max %d MB", u.MaxSize/(1024*1024))
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" && ext != ".webp" {
		return "", fmt.Errorf("unsupported image format: %s", ext)
	}

	dateDir := time.Now().Format("2006/01/02")
	dir := filepath.Join(u.BasePath, dateDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), randomStr(8), ext)
	fullPath := filepath.Join(dir, filename)

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/%s/%s", u.BaseURL, dateDir, filename)
	return url, nil
}

func (u *Uploader) SaveVideo(file *multipart.FileHeader) (string, error) {
	if file.Size > u.MaxSize*5 { // 5x image max for video
		return "", fmt.Errorf("file too large")
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".mp4" && ext != ".webm" && ext != ".mov" {
		return "", fmt.Errorf("unsupported video format: %s (use mp4/webm/mov)", ext)
	}

	dateDir := time.Now().Format("2006/01/02")
	dir := filepath.Join(u.BasePath, dateDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("vid_%d_%s%s", time.Now().UnixNano(), randomStr(8), ext)
	fullPath := filepath.Join(dir, filename)

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/%s/%s", u.BaseURL, dateDir, filename)
	return url, nil
}

func randomStr(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
