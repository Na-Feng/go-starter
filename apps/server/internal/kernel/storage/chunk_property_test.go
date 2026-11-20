package storage_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"math/rand"
	"os"
	"strings"
	"testing"
	"testing/quick"

	"go-starter/internal/kernel/storage"
)

func TestMultipartStateMachineInvariantsProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	prop := func(payload []byte, rawChunks uint8) bool {
		if len(payload) == 0 {
			payload = []byte("non-empty-seed")
		}
		numChunks := int(rawChunks%8) + 1
		if numChunks > len(payload) {
			numChunks = len(payload)
		}

		baseDir := t.TempDir()
		store, err := storage.NewLocalStore(baseDir, storage.ImageOptions{})
		if err != nil {
			return false
		}

		sum := sha256.Sum256(payload)
		sess, err := store.InitMultipart(
			context.Background(),
			"../nested/archive.bin",
			int64(len(payload)),
			numChunks,
			hex.EncodeToString(sum[:]),
		)
		if err != nil || sess.Filename != "archive.bin" {
			return false
		}

		parts := splitIntoChunks(payload, numChunks)
		order := rng.Perm(numChunks)
		for _, idx := range order {
			if err := store.SaveChunk(context.Background(), sess.UploadID, idx, bytes.NewReader([]byte("stale"))); err != nil {
				return false
			}
			if err := store.SaveChunk(context.Background(), sess.UploadID, idx, bytes.NewReader(parts[idx])); err != nil {
				return false
			}
		}

		info, err := store.CompleteMultipart(context.Background(), sess.UploadID)
		if err != nil || info.Size != int64(len(payload)) {
			return false
		}

		rc, _, _, err := store.OpenSeeker(context.Background(), info.Key)
		if err != nil {
			return false
		}
		defer rc.Close()
		got, err := io.ReadAll(rc)
		return err == nil && bytes.Equal(got, payload)
	}

	if err := quick.Check(prop, &quick.Config{MaxCount: 60}); err != nil {
		t.Fatalf("multipart upload invariant violated: %v", err)
	}
}

func TestMultipartFailureLeavesZeroOrphanedFiles(t *testing.T) {
	baseDir := t.TempDir()
	store, err := storage.NewLocalStore(baseDir, storage.ImageOptions{})
	if err != nil {
		t.Fatalf("init store: %v", err)
	}

	sessMissing, err := store.InitMultipart(context.Background(), "video.mp4", 10, 2, "")
	if err != nil {
		t.Fatalf("init missing test: %v", err)
	}
	_ = store.SaveChunk(context.Background(), sessMissing.UploadID, 0, bytes.NewReader([]byte("12345")))
	if _, err := store.CompleteMultipart(context.Background(), sessMissing.UploadID); err == nil {
		t.Fatalf("expected missing chunk error")
	}

	badSum := strings.Repeat("a", 64)
	sessTampered, err := store.InitMultipart(context.Background(), "archive.tar", 5, 1, badSum)
	if err != nil {
		t.Fatalf("init checksum test: %v", err)
	}
	_ = store.SaveChunk(context.Background(), sessTampered.UploadID, 0, bytes.NewReader([]byte("hello")))
	if _, err := store.CompleteMultipart(context.Background(), sessTampered.UploadID); err == nil {
		t.Fatalf("expected sha256 mismatch error")
	}

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, ent := range entries {
		if ent.Name() != ".chunks" {
			t.Fatalf("orphaned merged file leaked into baseDir on failed complete: %s", ent.Name())
		}
	}
}

func splitIntoChunks(data []byte, n int) [][]byte {
	out := make([][]byte, 0, n)
	base := len(data) / n
	rem := len(data) % n
	offset := 0
	for i := 0; i < n; i++ {
		sz := base
		if i < rem {
			sz++
		}
		out = append(out, data[offset:offset+sz])
		offset += sz
	}
	return out
}
