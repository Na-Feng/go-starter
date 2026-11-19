package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"go-starter/internal/kernel/errs"
)

type FileInfo struct {
	Key      string `json:"key"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	URL      string `json:"url"`
}

// Store 抽象文件存储接口，支持常规上传、分片上传与 Range 读取
type Store interface {
	Save(ctx context.Context, filename string, r io.Reader) (FileInfo, error)
	OpenSeeker(ctx context.Context, key string) (io.ReadSeekCloser, time.Time, string, error)
	InitMultipart(ctx context.Context, filename string, totalSize int64, totalChunks int, sha256Hex string) (MultipartSession, error)
	SaveChunk(ctx context.Context, uploadID string, index int, r io.Reader) error
	CompleteMultipart(ctx context.Context, uploadID string) (FileInfo, error)
}

type LocalStore struct {
	baseDir string
	imgOpt  ImageOptions
}

func NewLocalStore(baseDir string, imgOpt ImageOptions) (*LocalStore, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &LocalStore{baseDir: baseDir, imgOpt: imgOpt}, nil
}

func (s *LocalStore) Save(_ context.Context, filename string, r io.Reader) (FileInfo, error) {
	ext := filepath.Ext(filepath.Base(filename))
	src := r
	if isCompressibleImage(filename) {
		compressed, newExt, err := compressImage(r, filename, s.imgOpt)
		if err != nil {
			return FileInfo{}, err
		}
		src = compressed
		ext = newExt
	}

	key := newFileKey() + ext
	dstPath := filepath.Join(s.baseDir, key)

	f, err := os.Create(dstPath)
	if err != nil {
		return FileInfo{}, errs.Internal(err)
	}
	defer f.Close()

	n, err := io.Copy(f, src)
	if err != nil {
		_ = os.Remove(dstPath)
		return FileInfo{}, errs.Internal(err)
	}

	return FileInfo{
		Key:      key,
		Filename: filepath.Base(filename),
		Size:     n,
		URL:      "/api/v1/files/" + key,
	}, nil
}

func (s *LocalStore) OpenSeeker(_ context.Context, key string) (io.ReadSeekCloser, time.Time, string, error) {
	clean := filepath.Base(key)
	if clean == "." || clean == ".." || clean != key {
		return nil, time.Time{}, "", errs.BadRequest("invalid file key")
	}
	f, err := os.Open(filepath.Join(s.baseDir, clean))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, time.Time{}, "", errs.NotFound("file not found")
		}
		return nil, time.Time{}, "", errs.Internal(err)
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, time.Time{}, "", errs.Internal(err)
	}
	return f, info.ModTime(), clean, nil
}

func newFileKey() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
