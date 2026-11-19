package storage

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"

	"go-starter/internal/kernel/errs"
)

const maxImageDimension = 8192

type ImageOptions struct {
	MaxWidth    int
	JPEGQuality int
}

func isCompressibleImage(filename string) bool {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".jpg", ".jpeg", ".png":
		return true
	default:
		return false
	}
}

// compressImage 进行防像素炸弹尺寸校验、EXIF 方向纠偏与等比压缩（纯 Go 无 CGO）
func compressImage(r io.Reader, filename string, opt ImageOptions) (io.Reader, string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, "", errs.Internal(err)
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, "", errs.BadRequest("invalid image file: " + err.Error())
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > maxImageDimension || cfg.Height > maxImageDimension {
		return nil, "", errs.BadRequest(fmt.Sprintf("image dimensions %dx%d exceed limit (%d)", cfg.Width, cfg.Height, maxImageDimension))
	}

	img, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return nil, "", errs.BadRequest("decode image failed: " + err.Error())
	}

	maxW := opt.MaxWidth
	if maxW <= 0 {
		maxW = 1920
	}
	if img.Bounds().Dx() > maxW {
		img = imaging.Resize(img, maxW, 0, imaging.Lanczos)
	}

	quality := opt.JPEGQuality
	if quality <= 0 || quality > 100 {
		quality = 80
	}

	var buf bytes.Buffer
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".png" {
		if err := imaging.Encode(&buf, img, imaging.PNG); err != nil {
			return nil, "", errs.Internal(err)
		}
		return &buf, ".png", nil
	}
	if err := imaging.Encode(&buf, img, imaging.JPEG, imaging.JPEGQuality(quality)); err != nil {
		return nil, "", errs.Internal(err)
	}
	return &buf, ".jpg", nil
}
