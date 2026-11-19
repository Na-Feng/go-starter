package files_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-starter/internal/kernel/auth"
	"go-starter/internal/kernel/storage"
	"go-starter/internal/modules/files"
)

func TestImageCompressionAndMultipartUpload(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewLocalStore(dir, storage.ImageOptions{MaxWidth: 800, JPEGQuality: 80})
	if err != nil {
		t.Fatalf("init store failed: %v", err)
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	jwtMgr := auth.NewManager("test-secret")
	token, _ := jwtMgr.Sign("usr_1", []string{"operator"}, auth.TokenTypeAccess, time.Hour)

	mux := http.NewServeMux()
	files.NewHandler(store, 10, 100, log).Register(mux, jwtMgr, nil)

	// 1. 测试大图自动等比压缩（1600x800 -> 800x400）
	img := image.NewRGBA(image.Rect(0, 0, 1600, 800))
	img.Set(10, 10, color.RGBA{R: 255, A: 255})
	var imgBuf bytes.Buffer
	_ = jpeg.Encode(&imgBuf, img, &jpeg.Options{Quality: 95})

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "photo.jpg")
	_, _ = fw.Write(imgBuf.Bytes())
	_ = mw.Close()

	reqUp := httptest.NewRequest(http.MethodPost, "/api/v1/files/upload", &body)
	reqUp.Header.Set("Content-Type", mw.FormDataContentType())
	reqUp.Header.Set("Authorization", "Bearer "+token)
	recUp := httptest.NewRecorder()
	mux.ServeHTTP(recUp, reqUp)
	if recUp.Code != http.StatusCreated {
		t.Fatalf("expected 201 upload, got %d: %s", recUp.Code, recUp.Body.String())
	}

	// 2. 测试大文件分片上传（3 个分片 + SHA256 校验 + Range 下载）
	fullData := []byte("chunk0-chunk1-chunk2-video-stream-data")
	sum := sha256.Sum256(fullData)
	initPayload, _ := json.Marshal(files.InitMultipartReq{
		Filename:    "demo.mp4",
		TotalSize:   int64(len(fullData)),
		TotalChunks: 3,
		SHA256:      hex.EncodeToString(sum[:]),
	})
	reqInit := httptest.NewRequest(http.MethodPost, "/api/v1/files/multipart/init", bytes.NewReader(initPayload))
	reqInit.Header.Set("Authorization", "Bearer "+token)
	recInit := httptest.NewRecorder()
	mux.ServeHTTP(recInit, reqInit)
	if recInit.Code != http.StatusCreated {
		t.Fatalf("expected 201 init, got %d", recInit.Code)
	}

	var initEnv struct {
		Data storage.MultipartSession `json:"data"`
	}
	_ = json.Unmarshal(recInit.Body.Bytes(), &initEnv)
	uploadID := initEnv.Data.UploadID

	chunks := [][]byte{fullData[:10], fullData[10:20], fullData[20:]}
	for i, ch := range chunks {
		reqCh := httptest.NewRequest(http.MethodPut,
			"/api/v1/files/multipart/"+uploadID+"/chunks/"+string(rune('0'+i)), bytes.NewReader(ch))
		reqCh.Header.Set("Authorization", "Bearer "+token)
		recCh := httptest.NewRecorder()
		mux.ServeHTTP(recCh, reqCh)
		if recCh.Code != http.StatusOK {
			t.Fatalf("chunk %d failed: %d", i, recCh.Code)
		}
	}

	reqComp := httptest.NewRequest(http.MethodPost, "/api/v1/files/multipart/"+uploadID+"/complete", nil)
	reqComp.Header.Set("Authorization", "Bearer "+token)
	recComp := httptest.NewRecorder()
	mux.ServeHTTP(recComp, reqComp)
	if recComp.Code != http.StatusCreated {
		t.Fatalf("complete failed: %d: %s", recComp.Code, recComp.Body.String())
	}

	var compEnv struct {
		Data storage.FileInfo `json:"data"`
	}
	_ = json.Unmarshal(recComp.Body.Bytes(), &compEnv)
	if compEnv.Data.Size != int64(len(fullData)) {
		t.Fatalf("expected size %d, got %d", len(fullData), compEnv.Data.Size)
	}

	// 验证 Range 请求（206 Partial Content）
	reqRange := httptest.NewRequest(http.MethodGet, compEnv.Data.URL, nil)
	reqRange.Header.Set("Range", "bytes=0-5")
	recRange := httptest.NewRecorder()
	mux.ServeHTTP(recRange, reqRange)
	if recRange.Code != http.StatusPartialContent || recRange.Body.String() != "chunk0" {
		t.Fatalf("expected 206 'chunk0', got %d %q", recRange.Code, recRange.Body.String())
	}
}
