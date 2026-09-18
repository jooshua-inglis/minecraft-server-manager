package webapi

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestSplitLines(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"\n", nil},
		{"one", []string{"one"}},
		{"one\ntwo\n", []string{"one", "two"}},
		{"one\ntwo", []string{"one", "two"}},
	}
	for _, c := range cases {
		if got := splitLines(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("splitLines(%q) = %#v, want %#v", c.in, got, c.want)
		}
	}
}

func TestSSELineWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	sw := &sseLineWriter{w: rec, flusher: rec}

	if _, err := sw.Write([]byte("hel")); err != nil {
		t.Fatal(err)
	}
	if _, err := sw.Write([]byte("lo\r\nworld\n")); err != nil {
		t.Fatal(err)
	}
	if _, err := sw.Write([]byte("partial")); err != nil {
		t.Fatal(err)
	}

	want := "data: hello\n\ndata: world\n\n"
	if got := rec.Body.String(); got != want {
		t.Errorf("sseLineWriter output = %q, want %q", got, want)
	}
	if string(sw.buf) != "partial" {
		t.Errorf("sseLineWriter buffered %q, want %q", sw.buf, "partial")
	}
}

func TestRequireAuth(t *testing.T) {
	h := &Handler{token: "s3cret"}
	called := false
	protected := h.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	cases := []struct {
		name       string
		authHeader string
		wantStatus int
		wantCalled bool
	}{
		{"no header", "", http.StatusUnauthorized, false},
		{"wrong token", "Bearer nope", http.StatusUnauthorized, false},
		{"empty bearer", "Bearer ", http.StatusUnauthorized, false},
		{"correct token", "Bearer s3cret", http.StatusOK, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			called = false
			req := httptest.NewRequest(http.MethodPost, "/api/servers/foo/start", nil)
			if c.authHeader != "" {
				req.Header.Set("Authorization", c.authHeader)
			}
			rec := httptest.NewRecorder()
			protected(rec, req)

			if rec.Code != c.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, c.wantStatus)
			}
			if called != c.wantCalled {
				t.Errorf("handler called = %v, want %v", called, c.wantCalled)
			}
		})
	}
}
