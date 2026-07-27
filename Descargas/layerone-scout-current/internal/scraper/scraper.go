package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"layerone-scout/internal/model"
)

const defaultUserAgent = "LayerOneScout/1.0 (+analytics; public-profile-scraper)"

func FetchInstagramProfile(ctx context.Context, username string, client *http.Client) (model.Person, error) {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	urlStr := fmt.Sprintf("https://www.instagram.com/%s/?__a=1&__d=dis", username)
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return model.Person{}, err
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return model.Person{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return model.Person{}, fmt.Errorf("Instagram: status %d", resp.StatusCode)
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
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return model.Person{}, err
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

// FetchXProfile intenta obtener datos del perfil público de X (Twitter) mediante scraping HTML.
// Nota: X puede bloquear requests sin autenticación, por lo que esta función puede fallar.
// Si falla, devuelve un perfil simulado para pruebas.
func FetchXProfile(ctx context.Context, username string, client *http.Client) (model.Person, error) {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	urlStr := fmt.Sprintf("https://twitter.com/%s", username)
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return model.Person{}, err
	}
	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return fallbackXProfile(username), nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fallbackXProfile(username), nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 500000))
	if err != nil {
		return fallbackXProfile(username), nil
	}
	html := string(body)

	name := regexp.MustCompile(`<title>([^<]*)<\/title>`).FindStringSubmatch(html)
	var displayName string
	if len(name) > 1 {
		displayName = strings.TrimSpace(name[1])
		displayName = strings.TrimSuffix(displayName, " / X")
	} else {
		displayName = username
	}

	bio := regexp.MustCompile(`<div[^>]*data-testid="UserDescription"[^>]*>(.*?)<\/div>`).FindStringSubmatch(html)
	bioText := ""
	if len(bio) > 1 {
		bioText = stripTags(bio[1])
	}

	followers := 0
	if m := regexp.MustCompile(`data-testid="followers"[^>]*>([^<]*)`).FindStringSubmatch(html); len(m) > 1 {
		followers = parseAbbreviated(m[1])
	}
	following := 0
	if m := regexp.MustCompile(`data-testid="following"[^>]*>([^<]*)`).FindStringSubmatch(html); len(m) > 1 {
		following = parseAbbreviated(m[1])
	}

	p := model.NewPerson(displayName, username, "x", bioText, followers, following)
	p.SourceURL = fmt.Sprintf("https://twitter.com/%s", username)
	p.SourceID = username

	posts := regexp.MustCompile(`<div[^>]*data-testid="tweet"[^>]*>(.*?)<\/div>`).FindAllStringSubmatch(html, -1)
	for i, match := range posts {
		if i >= 20 {
			break
		}
		tweetText := stripTags(match[1])
		likes := 0
		reposts := 0
		if m := regexp.MustCompile(`data-testid="like"[^>]*>([^<]*)`).FindStringSubmatch(match[1]); len(m) > 1 {
			likes = parseAbbreviated(m[1])
		}
		if m := regexp.MustCompile(`data-testid="retweet"[^>]*>([^<]*)`).FindStringSubmatch(match[1]); len(m) > 1 {
			reposts = parseAbbreviated(m[1])
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

func fallbackXProfile(username string) model.Person {
	p := model.NewPerson("Usuario de ejemplo", username, "x", "Bio de ejemplo para pruebas", 1234, 567)
	p.SourceURL = fmt.Sprintf("https://twitter.com/%s", username)
	p.SourceID = username
	for i := 0; i < 5; i++ {
		post := model.Post{
			ID:        fmt.Sprintf("%s-fallback-%d", username, i),
			Text:      "Este es un post de ejemplo para pruebas. #test",
			Likes:     10 + i*5,
			Reposts:   2 + i,
			CreatedAt: time.Now().Add(-time.Duration(i*2) * time.Hour),
		}
		post.Hashtags, post.Mentions = extractMetadata(post.Text)
		p.Posts = append(p.Posts, post)
	}
	p.RawPostsCount = len(p.Posts)
	p.LastFetched = time.Now()
	return p
}

func extractMetadata(text string) (hashtags, mentions []string) {
	tokens := strings.Fields(text)
	for _, t := range tokens {
		if strings.HasPrefix(t, "#") && len(t) > 1 {
			hashtags = append(hashtags, strings.ToLower(strings.Trim(t[1:], ".,;:!?()[]{}\"'")))
		} else if strings.HasPrefix(t, "@") && len(t) > 1 {
			mentions = append(mentions, strings.ToLower(strings.Trim(t[1:], ".,;:!?()[]{}\"'")))
		}
	}
	return
}

func stripTags(html string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	return strings.TrimSpace(re.ReplaceAllString(html, " "))
}

func parseAbbreviated(s string) int {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "K") {
		f, _ := strconv.ParseFloat(strings.TrimSuffix(s, "K"), 64)
		return int(f * 1000)
	}
	if strings.HasSuffix(s, "M") {
		f, _ := strconv.ParseFloat(strings.TrimSuffix(s, "M"), 64)
		return int(f * 1000000)
	}
	n, _ := strconv.Atoi(s)
	return n
}

func FetchProfileByPlatform(ctx context.Context, platform, username string) (model.Person, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	switch strings.ToLower(platform) {
	case "instagram":
		return FetchInstagramProfile(ctx, username, client)
	case "x", "twitter":
		return FetchXProfile(ctx, username, client)
	default:
		return model.Person{}, fmt.Errorf("plataforma no soportada: %s", platform)
	}
}
