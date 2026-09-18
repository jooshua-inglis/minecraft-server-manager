package cliapp

import "testing"

func TestIsLoopback(t *testing.T) {
	cases := []struct {
		host string
		want bool
	}{
		{"127.0.0.1", true},
		{"localhost", true},
		{"::1", true},
		{"0.0.0.0", false},
		{"192.168.1.5", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isLoopback(c.host); got != c.want {
			t.Errorf("isLoopback(%q) = %v, want %v", c.host, got, c.want)
		}
	}
}
