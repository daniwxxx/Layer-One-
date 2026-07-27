package fetcher

import (
	"strings"
	"testing"
)

func TestParseCountLoose(t *testing.T) {
	cases := map[string]int{
		"987":   987,
		"1.2K":  1200,
		"1,234": 1234,
		"2.5M":  2500000,
	}
	for input, want := range cases {
		if got := parseCountLoose(input); got != want {
			t.Fatalf("parseCountLoose(%q) = %d, want %d", input, got, want)
		}
	}
}

func TestParseProfileCounts(t *testing.T) {
	text := "241019074 followers · 1375 following. Se unió el Jun 2009."
	followers, following, posts := parseProfileCounts(text)
	if followers != 241019074 {
		t.Fatalf("followers = %d, want %d", followers, 241019074)
	}
	if following != 1375 {
		t.Fatalf("following = %d, want %d", following, 1375)
	}
	if posts != 0 {
		t.Fatalf("posts = %d, want 0", posts)
	}
}

func TestParsePostCandidate(t *testing.T) {
	html := `
	<article data-testid="tweet">
	  <div data-testid="tweetText"><span>Hello #go @openai https://example.com</span></div>
	  <time datetime="2024-01-02T15:04:05Z"></time>
	  <div aria-label="12 Likes"></div>
	  <div aria-label="3 Reposts"></div>
	  <div aria-label="1 Reply"></div>
	</article>`

	blocks := extractTweetBlocks(html)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 tweet block, got %d", len(blocks))
	}
	cand, ok := parsePostCandidate(blocks[0], "user", 0)
	if !ok {
		t.Fatal("expected candidate to parse")
	}
	if !strings.Contains(cand.Text, "Hello") || !strings.Contains(cand.Text, "#go") {
		t.Fatalf("unexpected text: %q", cand.Text)
	}
	if cand.Likes != 12 || cand.Reposts != 3 || cand.Replies != 1 {
		t.Fatalf("unexpected counts: likes=%d reposts=%d replies=%d", cand.Likes, cand.Reposts, cand.Replies)
	}
	if len(cand.Hashtags) == 0 || cand.Hashtags[0] != "go" {
		t.Fatalf("hashtags not extracted: %#v", cand.Hashtags)
	}
	if len(cand.Mentions) == 0 || cand.Mentions[0] != "openai" {
		t.Fatalf("mentions not extracted: %#v", cand.Mentions)
	}
	if len(cand.Links) == 0 || cand.Links[0] != "https://example.com" {
		t.Fatalf("links not extracted: %#v", cand.Links)
	}
}
