package fleet

import "testing"

func TestParsePlayerCount(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"There are 0 of a max of 20 players online: ", "0/20"},
		{"There are 3 of a max of 20 players online: bob, alice, carl", "3/20"},
		{"", "-"},
		{"some unrelated response", "-"},
	}
	for _, c := range cases {
		if got := parsePlayerCount(c.in); got != c.want {
			t.Errorf("parsePlayerCount(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
