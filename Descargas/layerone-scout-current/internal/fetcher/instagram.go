package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
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
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return model.Person{}, err
		}
		req.Header.Set("User-Agent", f.cfg.UserAgent)
		req.Header.Set("Accept", "application/json,text/plain,*/*")
		req.Header.Set("Accept-Language", "es-ES,es;q=0.9,en;q=0.8")
		req.Header.Set("Referer", "https://www.instagram.com/")

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

		if p, ok := parseInstagramJSON(body, username); ok {
			return p, nil
		}
		if p, ok := parseInstagramHTML(string(body), username); ok {
			return p, nil
		}

		lastErr = fmt.Errorf("instagram: no se pudo extraer el perfil")
		sleepBackoff(f.cfg.BackoffBase, f.cfg.Jitter, attempt)
	}

	return model.Person{}, fmt.Errorf("falló después de %d intentos: %w", f.cfg.MaxRetries, lastErr)
}

func parseInstagramJSON(body []byte, username string) (model.Person, bool) {
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
		return model.Person{}, false
	}
	u := data.GraphQL.User
	if u.Username == "" && u.FullName == "" && u.Biography == "" && len(u.Posts.Edges) == 0 {
		return model.Person{}, false
	}

	p := model.NewPerson(u.FullName, firstNonEmpty(u.Username, username), "instagram", u.Biography, u.Follower.Count, u.Following.Count)
	p.SourceURL = fmt.Sprintf("https://www.instagram.com/%s/", username)
	p.SourceID = firstNonEmpty(u.Username, username)
	for _, edge := range u.Posts.Edges {
		ts, _ := strconv.ParseInt(edge.Node.Timestamp, 10, 64)
		t := time.Unix(ts, 0)
		post := model.Post{
			ID:        fmt.Sprintf("%s-%d", p.SourceID, t.Unix()),
			Text:      edge.Node.Text,
			Likes:     edge.Node.Likes.Count,
			Reposts:   0,
			Replies:   edge.Node.Comments.Count,
			CreatedAt: t,
		}
		post.Hashtags, post.Mentions, post.Links = textEntities(post.Text)
		p.Posts = append(p.Posts, post)
	}
	p.RawPostsCount = len(p.Posts)
	p.LastFetched = time.Now()
	return p, true
}

func parseInstagramHTML(html, username string) (model.Person, bool) {
	title := titleText(html)
	metaTitle := metaContent(html, "og:title")
	name := profileDisplayNameFromTitleOrMeta(metaTitle, title)
	rawDesc := firstNonEmpty(metaContent(html, "og:description"), metaContent(html, "description"), metaContent(html, "twitter:description"))
	followers, following, _ := parseProfileCounts(rawDesc)
	desc := cleanProfileDescription(rawDesc)

	if name == "" && desc == "" && followers == 0 && following == 0 {
		return model.Person{}, false
	}
	if name == "" {
		name = username
	}

	p := model.NewPerson(name, username, "instagram", desc, followers, following)
	p.SourceURL = fmt.Sprintf("https://www.instagram.com/%s/", username)
	p.SourceID = username
	p.RawPostsCount = 0
	p.LastFetched = time.Now()
	return p, true
}
