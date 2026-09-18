package webapi

import (
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
