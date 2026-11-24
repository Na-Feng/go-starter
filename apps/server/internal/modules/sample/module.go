package sample

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"go-starter/internal/kernel/auth"
	"go-starter/internal/kernel/errs"
	"go-starter/internal/kernel/httpx"
	"go-starter/internal/kernel/worker"
)

type Item struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateReq struct {
	Title string `json:"title" validate:"required,min=2,max=100"`
}

type Repository interface {
	Create(ctx context.Context, title string) (Item, error)
	GetByID(ctx context.Context, id string) (Item, error)
	List(ctx context.Context, limit, offset int) ([]Item, int64, error)
}

type TxRunner interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type memoryRepo struct {
	mu    sync.RWMutex
	seq   int
	order []string
	items map[string]Item
}

func NewMemoryRepo() Repository {
	return &memoryRepo{items: make(map[string]Item)}
}

func (m *memoryRepo) Create(_ context.Context, title string) (Item, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seq++
	id := fmt.Sprintf("smp_%d", m.seq)
	item := Item{ID: id, Title: title, CreatedAt: time.Now().UTC()}
	m.items[id] = item
	m.order = append(m.order, id)
	return item, nil
}

func (m *memoryRepo) GetByID(_ context.Context, id string) (Item, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	item, ok := m.items[id]
	if !ok {
		return Item{}, errs.NotFound("sample item not found")
	}
	return item, nil
}

func (m *memoryRepo) List(_ context.Context, limit, offset int) ([]Item, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	total := int64(len(m.order))
	if offset >= len(m.order) {
		return []Item{}, total, nil
	}
	end := offset + limit
	if end > len(m.order) {
		end = len(m.order)
	}
	out := make([]Item, 0, end-offset)
	for _, id := range m.order[offset:end] {
		out = append(out, m.items[id])
	}
	return out, total, nil
}

type Service struct {
	repo Repository
	tx   TxRunner
}

func NewService(repo Repository, tx TxRunner) *Service {
	return &Service{repo: repo, tx: tx}
}

func (s *Service) Create(ctx context.Context, req CreateReq) (Item, error) {
	var created Item
	err := s.runInTx(ctx, func(txCtx context.Context) error {
		var err error
		created, err = s.repo.Create(txCtx, req.Title)
		return err
	})
	if err != nil {
		return Item{}, err
	}
	return created, nil
}

func (s *Service) Get(ctx context.Context, id string) (Item, error) {
	if id == "" {
		return Item{}, errs.BadRequest("id is required")
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context, pq httpx.PageQuery) (httpx.PageResult[Item], error) {
	items, total, err := s.repo.List(ctx, pq.Limit(), pq.Offset())
	if err != nil {
		return httpx.PageResult[Item]{}, err
	}
	return httpx.PageResult[Item]{
		Items:    items,
		Total:    total,
		Page:     pq.Page,
		PageSize: pq.PageSize,
	}, nil
}

func (s *Service) runInTx(ctx context.Context, fn func(context.Context) error) error {
	if s.tx == nil {
		return fn(ctx)
	}
	return s.tx.WithinTx(ctx, fn)
}

type Handler struct {
	svc    *Service
	worker *worker.Client
	log    *slog.Logger
}

func NewHandler(svc *Service, w *worker.Client, log *slog.Logger) *Handler {
	return &Handler{svc: svc, worker: w, log: log}
}

func (h *Handler) Register(r *auth.Router) {
	r.Public("GET /api/v1/samples", h.list)
	r.Public("GET /api/v1/samples/{id}", h.get)
	r.Protected("POST /api/v1/samples", "sample:write", h.create)
	r.Authenticated("POST /api/v1/samples/export", h.exportAsync)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req CreateReq
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	item, err := h.svc.Create(r.Context(), req)
	if err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	httpx.Created(w, r, item)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	item, err := h.svc.Get(r.Context(), id)
	if err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	httpx.OK(w, r, item)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	pq := httpx.ParsePageQuery(r)
	res, err := h.svc.List(r.Context(), pq)
	if err != nil {
		httpx.Fail(w, r, h.log, err)
		return
	}
	httpx.OK(w, r, res)
}

func (h *Handler) exportAsync(w http.ResponseWriter, r *http.Request) {
	if h.worker == nil {
		httpx.Fail(w, r, h.log, errs.Unavailable("worker not configured", nil))
		return
	}
	claims, _ := auth.UserFrom(r.Context())
	taskID, err := h.worker.EnqueueSampleExport(r.Context(), claims.UserID)
	if err != nil {
		httpx.Fail(w, r, h.log, errs.Internal(err))
		return
	}
	httpx.Accepted(w, r, map[string]string{"task_id": taskID})
}
