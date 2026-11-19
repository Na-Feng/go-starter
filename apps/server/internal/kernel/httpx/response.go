package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"

	"go-starter/internal/kernel/errs"
	"go-starter/internal/kernel/logger"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

type Envelope struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func OK(w http.ResponseWriter, r *http.Request, data any) {
	WriteJSON(w, http.StatusOK, Envelope{
		Code:      0,
		Message:   "ok",
		Data:      data,
		RequestID: logger.RequestIDFrom(r.Context()),
	})
}

func Created(w http.ResponseWriter, r *http.Request, data any) {
	WriteJSON(w, http.StatusCreated, Envelope{
		Code:      0,
		Message:   "created",
		Data:      data,
		RequestID: logger.RequestIDFrom(r.Context()),
	})
}

func Accepted(w http.ResponseWriter, r *http.Request, data any) {
	WriteJSON(w, http.StatusAccepted, Envelope{
		Code:      0,
		Message:   "accepted",
		Data:      data,
		RequestID: logger.RequestIDFrom(r.Context()),
	})
}

func Fail(w http.ResponseWriter, r *http.Request, log *slog.Logger, err error) {
	appErr := errs.From(err)
	reqID := logger.RequestIDFrom(r.Context())

	if appErr.HTTPStatus >= 500 && log != nil {
		log.Error("request failed",
			"request_id", reqID,
			"method", r.Method,
			"path", r.URL.Path,
			"error", appErr.Error(),
		)
	}

	WriteJSON(w, appErr.HTTPStatus, Envelope{
		Code:      appErr.Code,
		Message:   appErr.Message,
		RequestID: reqID,
	})
}

func DecodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return errs.BadRequest("invalid JSON payload: " + err.Error())
	}
	return nil
}

func DecodeAndValidate(r *http.Request, dst any) error {
	if err := DecodeJSON(r, dst); err != nil {
		return err
	}
	if err := validate.Struct(dst); err != nil {
		var verRs validator.ValidationErrors
		if errors.As(err, &verRs) {
			msgs := make([]string, 0, len(verRs))
			for _, fe := range verRs {
				msgs = append(msgs, fmt.Sprintf("%s failed on '%s'", fe.Field(), fe.Tag()))
			}
			return errs.BadRequest("validation failed: " + strings.Join(msgs, "; "))
		}
		return errs.BadRequest("validation failed: " + err.Error())
	}
	return nil
}
