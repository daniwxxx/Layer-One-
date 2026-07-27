package fetcher

import "testing"

func TestParseProfileCountsSmart(t *testing.T) {
	text := "241019074 followers · 1375 following. Se unió el Jun 2009."
	followers, following, posts := parseProfileCountsSmart(text)
	if followers != 241019074 {
		t.Fatalf("followers=%d, want %d", followers, 241019074)
	}
	if following != 1375 {
		t.Fatalf("following=%d, want %d", following, 1375)
	}
	if posts != 0 {
		t.Fatalf("posts=%d, want 0", posts)
	}
}
