package httpx_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-starter/internal/kernel/errs"
	"go-starter/internal/kernel/httpx"
)

type fuzzPayload struct {
	Title string `json:"title" validate:"required,min=2,max=50"`
	Count int    `json:"count" validate:"gte=0,lte=100"`
}

func FuzzDecodeAndValidate(f *testing.F) {
	f.Add([]byte(`{"title":"ok","count":1}`))
	f.Add([]byte(`{`))
	f.Add([]byte(`{"title":"","unknown":true}`))
	f.Add([]byte(`null`))

	f.Fuzz(func(t *testing.T, data []byte) {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(data))
		var dst fuzzPayload
		err := httpx.DecodeAndValidate(req, &dst)
		if err != nil {
			var appErr *errs.AppError
			if !errors.As(err, &appErr) {
				t.Fatalf("expected *errs.AppError, got %T: %v", err, err)
			}
		}
	})
}
