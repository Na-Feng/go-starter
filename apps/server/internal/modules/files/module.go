package files

import (
	"log/slog"
	"net/http"
	"strconv"

	"go-starter/internal/kernel/auth"
	"go-starter/internal/kernel/errs"
	"go-starter/internal/kernel/httpx"
	"go-starter/internal/kernel/storage"
)

type InitMultipartReq struct {
	Filename    string `json:"filename" validate:"required,min=1,max=255"`
	TotalSize   int64  `json:"total_size" validate:"required,gt=0"`
	TotalChunks int    `json:"total_chunks" validate:"required,gt=0"`
	SHA256      string `json:"sha256" validate:"omitempty,len=64"`
}

type Handler struct {
	store            storage.Store
	maxUploadMB      int64
	maxLargeUploadMB int64
	log              *slog.Logger
}

func NewHandler(store storage.Store, maxUploadMB, maxLargeUploadMB int64, log *slog.Logger) *Handler {
	return &Handler{
		store:            store,
		maxUploadMB:      maxUploadMB,
		maxLargeUploadMB: maxLargeUploadMB,
		log:              log,
	}
}

func (h *Handler) Register(r *auth.Router) {
	r.Authenticated("POST /api/v1/files/upload", h.upload)
	r.Authenticated("POST /api/v1/files/multipart/init", h.initMultipart)
	r.Authenticated("PUT /api/v1/files/multipart/{upload_id}/chunks/{index}", h.uploadChunk)
	r.Authenticated("POST /api/v1/files/multipart/{upload_id}/complete", h.completeMultipart)
	r.Public("GET /api/v1/files/{key}", h.download)
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	maxBytes := h.maxUploadMB << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	if err := r.ParseMultipartForm(maxBytes); err != nil {
		httpx.Fail(w, r, h.log, errs.BadRequest("file too large or invalid multipart form"))
		return
	}
	f, header, err := r.FormFile("file")
	if err != nil {
		httpx.Fail(w, r, h.log, errs.BadRequest("missing 'file' form field"))
		return
	}
	defer f.Close()

	info, err := h.store.Save(r.Context(), header.Filename, f)
	if err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	httpx.Created(w, r, info)
}

func (h *Handler) initMultipart(w http.ResponseWriter, r *http.Request) {
	var req InitMultipartReq
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	if h.maxLargeUploadMB > 0 && req.TotalSize > (h.maxLargeUploadMB<<20) {
		httpx.Fail(w, r, h.log, errs.BadRequest("file exceeds max large upload limit"))
		return
	}
	sess, err := h.store.InitMultipart(r.Context(), req.Filename, req.TotalSize, req.TotalChunks, req.SHA256)
	if err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	httpx.Created(w, r, sess)
}

func (h *Handler) uploadChunk(w http.ResponseWriter, r *http.Request) {
	idx, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		httpx.Fail(w, r, h.log, errs.BadRequest("invalid chunk index"))
		return
	}
	maxChunkBytes := h.maxUploadMB << 20
	reader := http.MaxBytesReader(w, r.Body, maxChunkBytes)
	if err := h.store.SaveChunk(r.Context(), r.PathValue("upload_id"), idx, reader); err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	httpx.OK(w, r, map[string]any{"chunk_index": idx, "status": "uploaded"})
}

func (h *Handler) completeMultipart(w http.ResponseWriter, r *http.Request) {
	info, err := h.store.CompleteMultipart(r.Context(), r.PathValue("upload_id"))
	if err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	httpx.Created(w, r, info)
}

func (h *Handler) download(w http.ResponseWriter, r *http.Request) {
	rs, modTime, name, err := h.store.OpenSeeker(r.Context(), r.PathValue("key"))
	if err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	defer rs.Close()
	// ServeContent 原生支持 HTTP Range 断点续传与视频进度条拖动
	http.ServeContent(w, r, name, modTime, rs)
}
