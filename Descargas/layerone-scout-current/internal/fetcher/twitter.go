package fetcher

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"layerone-scout/internal/model"
	"layerone-scout/pkg/utils"
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
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return model.Person{}, err
		}
		req.Header.Set("User-Agent", f.cfg.UserAgent)
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(f.cfg.BackoffBase + time.Duration(attempt)*time.Second + time.Duration(attempt)*f.cfg.Jitter)
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
			time.Sleep(f.cfg.BackoffBase + time.Duration(attempt)*time.Second + time.Duration(attempt)*f.cfg.Jitter)
			continue
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, f.cfg.MaxBodyBytes))
		if err != nil {
			lastErr = err
			continue
		}
		html := string(body)
		name := regexp.MustCompile(`<title>([^<]*)<\/title>`).FindStringSubmatch(html)
		displayName := username
		if len(name) > 1 {
			displayName = strings.TrimSuffix(strings.TrimSpace(name[1]), " / X")
		}
		bio := regexp.MustCompile(`<div[^>]*data-testid="UserDescription"[^>]*>(.*?)<\/div>`).FindStringSubmatch(html)
		bioText := ""
		if len(bio) > 1 {
			bioText = stripTags(bio[1])
		}
		followers := 0
		if m := regexp.MustCompile(`data-testid="followers"[^>]*>([^<]*)`).FindStringSubmatch(html); len(m) > 1 {
			followers = utils.ParseAbbreviated(m[1])
		}
		following := 0
		if m := regexp.MustCompile(`data-testid="following"[^>]*>([^<]*)`).FindStringSubmatch(html); len(m) > 1 {
			following = utils.ParseAbbreviated(m[1])
		}
		p := model.NewPerson(displayName, username, "x", bioText, followers, following)
		p.SourceURL = fmt.Sprintf("https://twitter.com/%s", username)
		p.SourceID = username
		tweets := regexp.MustCompile(`<div[^>]*data-testid="tweet"[^>]*>(.*?)<\/div>`).FindAllStringSubmatch(html, -1)
		for i, match := range tweets {
			if i >= 20 {
				break
			}
			tweetText := stripTags(match[1])
			likes := 0
			reposts := 0
			if m := regexp.MustCompile(`data-testid="like"[^>]*>([^<]*)`).FindStringSubmatch(match[1]); len(m) > 1 {
				likes = utils.ParseAbbreviated(m[1])
			}
			if m := regexp.MustCompile(`data-testid="retweet"[^>]*>([^<]*)`).FindStringSubmatch(match[1]); len(m) > 1 {
				reposts = utils.ParseAbbreviated(m[1])
			}
			post := model.Post{
				ID:        fmt.Sprintf("%s-%d", username, time.Now().UnixNano()+int64(i)),
				Text:      tweetText,
				Likes:     likes,
				Reposts:   reposts,
				CreatedAt: time.Now().Add(-time.Duration(i) * time.Hour),
			}
			post.Hashtags, post.Mentions = extractMetadata(post.Text)
			p.Posts = append(p.Posts, post)
		}
		p.RawPostsCount = len(p.Posts)
		p.LastFetched = time.Now()
		return p, nil
	}
	return model.Person{}, fmt.Errorf("falló después de %d intentos: %w", f.cfg.MaxRetries, lastErr)
}

func stripTags(html string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	return strings.TrimSpace(re.ReplaceAllString(html, " "))
}
