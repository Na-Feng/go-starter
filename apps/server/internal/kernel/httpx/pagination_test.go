package httpx_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"testing/quick"

	"go-starter/internal/kernel/httpx"
)

func TestPageQueryInvariantsProperty(t *testing.T) {
	prop := func(rawPage, rawSize int) bool {
		q := url.Values{}
		q.Set("page", fmt.Sprintf("%d", rawPage))
		q.Set("page_size", fmt.Sprintf("%d", rawSize))
		req := httptest.NewRequest(http.MethodGet, "/items?"+q.Encode(), nil)

		pq := httpx.ParsePageQuery(req)
		if pq.Page < 1 || pq.Page > httpx.MaxPage {
			return false
		}
		if pq.PageSize < 1 || pq.PageSize > httpx.MaxPageSize {
			return false
		}
		if pq.Limit() != pq.PageSize {
			return false
		}
		if pq.Offset() < 0 || pq.Offset() != (pq.Page-1)*pq.PageSize {
			return false
		}

		direct := httpx.PageQuery{Page: rawPage, PageSize: rawSize}
		return direct.Limit() >= 1 && direct.Limit() <= httpx.MaxPageSize && direct.Offset() >= 0
	}

	if err := quick.Check(prop, &quick.Config{MaxCount: 500}); err != nil {
		t.Fatalf("pagination invariant violated: %v", err)
	}
}

func FuzzParsePageQuery(f *testing.F) {
	f.Add("1", "20")
	f.Add("0", "0")
	f.Add("-999999", "-1")
	f.Add("9223372036854775807", "999999999")
	f.Add("abc", "NaN")

	f.Fuzz(func(t *testing.T, pageStr, sizeStr string) {
		q := url.Values{}
		q.Set("page", pageStr)
		q.Set("page_size", sizeStr)
		req := httptest.NewRequest(http.MethodGet, "/items?"+q.Encode(), nil)
		pq := httpx.ParsePageQuery(req)

		if pq.Page < 1 || pq.Page > httpx.MaxPage {
			t.Fatalf("Page invariant broken: %d", pq.Page)
		}
		if pq.PageSize < 1 || pq.PageSize > httpx.MaxPageSize {
			t.Fatalf("PageSize invariant broken: %d", pq.PageSize)
		}
		if pq.Offset() < 0 {
			t.Fatalf("Offset overflow/negative invariant broken: %d", pq.Offset())
		}
	})
}
