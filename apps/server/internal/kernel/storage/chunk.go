package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go-starter/internal/kernel/errs"
)

const maxChunkCount = 2000

type MultipartSession struct {
	UploadID    string `json:"upload_id"`
	Filename    string `json:"filename"`
	TotalSize   int64  `json:"total_size"`
	TotalChunks int    `json:"total_chunks"`
	SHA256      string `json:"sha256,omitempty"`
}

func (s *LocalStore) chunkDir(uploadID string) (string, error) {
	clean := filepath.Base(uploadID)
	if clean == "." || clean == ".." || clean != uploadID {
		return "", errs.BadRequest("invalid upload_id")
	}
	return filepath.Join(s.baseDir, ".chunks", clean), nil
}

func (s *LocalStore) InitMultipart(_ context.Context, filename string, totalSize int64, totalChunks int, sha256Hex string) (MultipartSession, error) {
	if totalChunks <= 0 || totalChunks > maxChunkCount {
		return MultipartSession{}, errs.BadRequest("invalid total_chunks")
	}
	uploadID := newFileKey()
	dir, err := s.chunkDir(uploadID)
	if err != nil {
		return MultipartSession{}, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return MultipartSession{}, errs.Internal(err)
	}

	sess := MultipartSession{
		UploadID:    uploadID,
		Filename:    filepath.Base(filename),
		TotalSize:   totalSize,
		TotalChunks: totalChunks,
		SHA256:      sha256Hex,
	}
	raw, _ := json.Marshal(sess)
	if err := os.WriteFile(filepath.Join(dir, "meta.json"), raw, 0o644); err != nil {
		return MultipartSession{}, errs.Internal(err)
	}
	return sess, nil
}

func (s *LocalStore) SaveChunk(_ context.Context, uploadID string, index int, r io.Reader) error {
	dir, err := s.chunkDir(uploadID)
	if err != nil {
		return err
	}
	sess, err := s.readMeta(dir)
	if err != nil {
		return err
	}
	if index < 0 || index >= sess.TotalChunks {
		return errs.BadRequest("chunk index out of range")
	}

	partPath := filepath.Join(dir, fmt.Sprintf("%06d.part", index))
	f, err := os.Create(partPath)
	if err != nil {
		return errs.Internal(err)
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		_ = os.Remove(partPath)
		return errs.Internal(err)
	}
	return nil
}

func (s *LocalStore) CompleteMultipart(_ context.Context, uploadID string) (FileInfo, error) {
	dir, err := s.chunkDir(uploadID)
	if err != nil {
		return FileInfo{}, err
	}
	sess, err := s.readMeta(dir)
	if err != nil {
		return FileInfo{}, err
	}

	ext := filepath.Ext(sess.Filename)
	key := newFileKey() + ext
	dstPath := filepath.Join(s.baseDir, key)
	dst, err := os.Create(dstPath)
	if err != nil {
		return FileInfo{}, errs.Internal(err)
	}

	hasher := sha256.New()
	multi := io.MultiWriter(dst, hasher)
	var written int64

	for i := 0; i < sess.TotalChunks; i++ {
		partPath := filepath.Join(dir, fmt.Sprintf("%06d.part", i))
		pf, err := os.Open(partPath)
		if err != nil {
			_ = dst.Close()
			_ = os.Remove(dstPath)
			if os.IsNotExist(err) {
				return FileInfo{}, errs.BadRequest(fmt.Sprintf("missing chunk %d", i))
			}
			return FileInfo{}, errs.Internal(err)
		}
		n, copyErr := io.Copy(multi, pf)
		_ = pf.Close()
		if copyErr != nil {
			_ = dst.Close()
			_ = os.Remove(dstPath)
			return FileInfo{}, errs.Internal(copyErr)
		}
		written += n
	}
	_ = dst.Close()

	if sess.SHA256 != "" {
		got := hex.EncodeToString(hasher.Sum(nil))
		if got != sess.SHA256 {
			_ = os.Remove(dstPath)
			return FileInfo{}, errs.BadRequest("sha256 checksum mismatch")
		}
	}

	_ = os.RemoveAll(dir)
	return FileInfo{
		Key:      key,
		Filename: sess.Filename,
		Size:     written,
		URL:      "/api/v1/files/" + key,
	}, nil
}

func (s *LocalStore) readMeta(dir string) (MultipartSession, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "meta.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return MultipartSession{}, errs.NotFound("upload session not found")
		}
		return MultipartSession{}, errs.Internal(err)
	}
	var sess MultipartSession
	if err := json.Unmarshal(raw, &sess); err != nil {
		return MultipartSession{}, errs.Internal(err)
	}
	return sess, nil
}
