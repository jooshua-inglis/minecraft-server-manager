package tui

import (
	"reflect"
	"testing"
)

func TestLineChanWriter(t *testing.T) {
	ch := make(chan string, 8)
	w := &lineChanWriter{ch: ch}

	for _, chunk := range []string{"hel", "lo\r\nwor", "ld\npartial"} {
		if _, err := w.Write([]byte(chunk)); err != nil {
			t.Fatal(err)
		}
	}
	close(ch)

	var got []string
	for l := range ch {
		got = append(got, l)
	}
	if want := []string{"hello", "world"}; !reflect.DeepEqual(got, want) {
		t.Errorf("lines = %#v, want %#v", got, want)
	}
	if string(w.buf) != "partial" {
		t.Errorf("buffered %q, want %q", w.buf, "partial")
	}
}
