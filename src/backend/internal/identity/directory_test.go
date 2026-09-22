package identity

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestLookupReadsUnifiedProfileAndCaches(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		if r.URL.Path != "/api/public/subject-1" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"displayName":"统一昵称","avatarUrl":"https://id.isoumao.cn/profile/avatar/subject-1"}`))
	}))
	defer server.Close()

	directory := NewDirectory(server.URL)
	name, avatar := directory.Lookup(context.Background(), "subject-1")
	if name != "统一昵称" || avatar != "https://id.isoumao.cn/profile/avatar/subject-1" {
		t.Fatalf("lookup = %q / %q", name, avatar)
	}

	// 第二次查询命中进程内缓存，不再回源。
	directory.Lookup(context.Background(), "subject-1")
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("upstream calls = %d, want 1", got)
	}
}

func TestLookupDisabledWithoutBaseURL(t *testing.T) {
	directory := NewDirectory("")
	if directory.Enabled() {
		t.Fatal("empty base url must disable sync")
	}
	if name, avatar := directory.Lookup(context.Background(), "subject-1"); name != "" || avatar != "" {
		t.Fatalf("disabled lookup returned %q / %q", name, avatar)
	}
}
