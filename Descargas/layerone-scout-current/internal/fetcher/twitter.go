package fetcher

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"layerone-scout/internal/model"
)

type TwitterFetcher struct {
	cfg Config
}

func NewTwitterFetcher(cfg Config) *TwitterFetcher {
	return &TwitterFetcher{cfg: cfg}
}

func (f *TwitterFetcher) Platform() string { return "x" }

func (f *TwitterFetcher) Fetch(ctx context.Context, username string) (model.Person, error) {
	url := fmt.Sprintf("https://twitter.com/%s", username)
	client := &http.Client{Timeout: f.cfg.Timeout}
	var lastErr error

	for attempt := 0; attempt <= f.cfg.MaxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return model.Person{}, err
		}
		req.Header.Set("User-Agent", f.cfg.UserAgent)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		req.Header.Set("Accept-Language", "es-ES,es;q=0.9,en;q=0.8")
		req.Header.Set("Cache-Control", "no-cache")
		req.Header.Set("Pragma", "no-cache")
		req.Header.Set("Referer", "https://x.com/")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			sleepBackoff(f.cfg.BackoffBase, f.cfg.Jitter, attempt)
			continue
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, f.cfg.MaxBodyBytes))
		_ = resp.Body.Close()
		if err != nil {
			lastErr = err
			sleepBackoff(f.cfg.BackoffBase, f.cfg.Jitter, attempt)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
			sleepBackoff(f.cfg.BackoffBase, f.cfg.Jitter, attempt)
			continue
		}

		html := string(body)
		title := firstNonEmpty(metaContent(html, "twitter:title"), metaContent(html, "og:title"), titleText(html))
		desc := cleanProfileDescription(firstNonEmpty(metaContent(html, "twitter:description"), metaContent(html, "og:description"), metaContent(html, "description")))
		displayName := profileDisplayNameFromTitleOrMeta(metaContent(html, "twitter:title"), title)
		if displayName == "" {
			displayName = username
		}

		followers, following, _ := parseProfileCounts(desc)
		p := model.NewPerson(displayName, username, "x", desc, followers, following)
		p.SourceURL = url
		p.SourceID = username

		blocks := extractTweetBlocks(html)
		for i, block := range blocks {
			cand, ok := parsePostCandidate(block, username, i)
			if !ok {
				continue
			}
			post := model.Post{
				ID:        cand.ID,
				Text:      cand.Text,
				Likes:     cand.Likes,
				Reposts:   cand.Reposts,
				Replies:   cand.Replies,
				CreatedAt: cand.CreatedAt,
				Hashtags:  cand.Hashtags,
				Mentions:  cand.Mentions,
				Links:     cand.Links,
				Media:     cand.Media,
				Language:  cand.Language,
			}
			p.Posts = append(p.Posts, post)
		}

		p.RawPostsCount = len(p.Posts)
		p.LastFetched = time.Now()
		if p.Name == "" {
			p.Name = username
		}
		return p, nil
	}

	return model.Person{}, fmt.Errorf("falló después de %d intentos: %w", f.cfg.MaxRetries, lastErr)
}

func sleepBackoff(base, jitter time.Duration, attempt int) {
	delay := base + time.Duration(attempt)*time.Second + time.Duration(attempt)*jitter
	if delay > 0 {
		time.Sleep(delay)
	}
}
