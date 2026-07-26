package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"layerone-scout/internal/model"
)

type InstagramFetcher struct {
	cfg Config
}

func NewInstagramFetcher(cfg Config) *InstagramFetcher {
	return &InstagramFetcher{cfg: cfg}
}

func (f *InstagramFetcher) Platform() string { return "instagram" }

func (f *InstagramFetcher) Fetch(ctx context.Context, username string) (model.Person, error) {
	url := fmt.Sprintf("https://www.instagram.com/%s/?__a=1&__d=dis", username)
	client := &http.Client{Timeout: f.cfg.Timeout}
	var lastErr error
	for attempt := 0; attempt <= f.cfg.MaxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return model.Person{}, err
		}
		req.Header.Set("User-Agent", f.cfg.UserAgent)
		req.Header.Set("Accept", "application/json")
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
		var data struct {
			GraphQL struct {
				User struct {
					FullName  string `json:"full_name"`
					Username  string `json:"username"`
					Biography string `json:"biography"`
					Follower  struct {
						Count int `json:"count"`
					} `json:"edge_followed_by"`
					Following struct {
						Count int `json:"count"`
					} `json:"edge_follow"`
					Posts struct {
						Edges []struct {
							Node struct {
								Text     string `json:"text"`
								Likes    struct {
									Count int `json:"count"`
								} `json:"edge_liked_by"`
								Comments struct {
									Count int `json:"count"`
								} `json:"edge_media_to_comment"`
								Timestamp string `json:"taken_at_timestamp"`
							} `json:"node"`
						} `json:"edges"`
					} `json:"edge_owner_to_timeline_media"`
				} `json:"user"`
			} `json:"graphql"`
		}
		if err := json.Unmarshal(body, &data); err != nil {
			lastErr = err
			continue
		}
		u := data.GraphQL.User
		p := model.NewPerson(u.FullName, u.Username, "instagram", u.Biography, u.Follower.Count, u.Following.Count)
		p.SourceURL = fmt.Sprintf("https://www.instagram.com/%s/", username)
		p.SourceID = u.Username
		for _, edge := range u.Posts.Edges {
			ts, _ := strconv.ParseInt(edge.Node.Timestamp, 10, 64)
			t := time.Unix(ts, 0)
			post := model.Post{
				ID:        fmt.Sprintf("%s-%d", u.Username, t.Unix()),
				Text:      edge.Node.Text,
				Likes:     edge.Node.Likes.Count,
				Reposts:   0,
				Replies:   edge.Node.Comments.Count,
				CreatedAt: t,
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

func extractMetadata(text string) (hashtags, mentions []string) {
	tokens := strings.Fields(text)
	for _, t := range tokens {
		if strings.HasPrefix(t, "#") && len(t) > 1 {
			hashtags = append(hashtags, strings.ToLower(t[1:]))
		} else if strings.HasPrefix(t, "@") && len(t) > 1 {
			mentions = append(mentions, strings.ToLower(t[1:]))
		}
	}
	return
}
