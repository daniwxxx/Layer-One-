package fetcher

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	reWhitespace     = regexp.MustCompile(`\s+`)
	reTitle          = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	reMeta1          = regexp.MustCompile(`(?is)<meta[^>]+(?:property|name)=["']([^"']+)["'][^>]*content=["']([^"']*)["'][^>]*>`)
	reMeta2          = regexp.MustCompile(`(?is)<meta[^>]+content=["']([^"']*)["'][^>]*(?:property|name)=["']([^"']+)["'][^>]*>`)
	reTweetArticle   = regexp.MustCompile(`(?is)<article[^>]*data-testid=["']tweet["'][^>]*>(.*?)</article>`)
	reTweetDiv       = regexp.MustCompile(`(?is)<div[^>]*data-testid=["']tweet["'][^>]*>(.*?)</div>`)
	reTweetTextBlock = regexp.MustCompile(`(?is)data-testid=["']tweetText["'][^>]*>(.*?)</div>`)
	reTime           = regexp.MustCompile(`(?i)<time[^>]*datetime=["']([^"']+)["']`)
	reHashtag        = regexp.MustCompile(`(?i)(?:^|\s)#([\p{L}\p{N}_]+)`)
	reMention        = regexp.MustCompile(`(?i)(?:^|\s)@([\p{L}\p{N}_]+)`)
	reURL            = regexp.MustCompile(`(?i)https?://[^\s<>"]+`)
	reCountLoose     = regexp.MustCompile(`([0-9][0-9.,]*\s*[km]?)`)
)

func collapseWhitespace(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.TrimSpace(reWhitespace.ReplaceAllString(s, " "))
}

func stripTags(html string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	return collapseWhitespace(re.ReplaceAllString(html, " "))
}

func metaContent(html, key string) string {
	for _, re := range []*regexp.Regexp{reMeta1, reMeta2} {
		matches := re.FindAllStringSubmatch(html, -1)
		for _, m := range matches {
			if len(m) < 3 {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(m[1]), key) || strings.EqualFold(strings.TrimSpace(m[2]), key) {
				if strings.EqualFold(strings.TrimSpace(m[1]), key) {
					return strings.TrimSpace(m[2])
				}
				return strings.TrimSpace(m[1])
			}
		}
	}
	return ""
}

func titleText(html string) string {
	if m := reTitle.FindStringSubmatch(html); len(m) > 1 {
		return stripTags(m[1])
	}
	return ""
}

func extractTweetBlocks(html string) []string {
	blocks := reTweetArticle.FindAllStringSubmatch(html, -1)
	if len(blocks) == 0 {
		blocks = reTweetDiv.FindAllStringSubmatch(html, -1)
	}
	out := make([]string, 0, len(blocks))
	for _, b := range blocks {
		if len(b) > 1 {
			out = append(out, b[1])
		}
	}
	return out
}

func tweetTextFromBlock(block string) string {
	if m := reTweetTextBlock.FindStringSubmatch(block); len(m) > 1 {
		return stripTags(m[1])
	}
	if m := regexp.MustCompile(`(?is)<div[^>]*lang=["'][^"']+["'][^>]*>(.*?)</div>`).FindStringSubmatch(block); len(m) > 1 {
		return stripTags(m[1])
	}
	return stripTags(block)
}

func tweetLanguageFromBlock(block string) string {
	if m := regexp.MustCompile(`(?i)lang=["']([^"']+)["']`).FindStringSubmatch(block); len(m) > 1 {
		return strings.ToLower(strings.TrimSpace(m[1]))
	}
	return ""
}

func tweetCreatedAtFromBlock(block string) time.Time {
	if m := reTime.FindStringSubmatch(block); len(m) > 1 {
		if t, err := time.Parse(time.RFC3339, m[1]); err == nil {
			return t
		}
		if t, err := time.Parse("2006-01-02T15:04:05.000Z", m[1]); err == nil {
			return t
		}
	}
	return time.Time{}
}

func tweetCountsFromBlock(block string) (likes, reposts, replies int) {
	candidates := append([]string{}, attributeValues(block, "aria-label", "title", "data-label")...)
	candidates = append(candidates, collapseWhitespace(stripTags(block)))
	for _, candidate := range candidates {
		if likes == 0 {
			likes = countNearLabels(candidate, "like", "likes", "me gusta", "favs")
		}
		if reposts == 0 {
			reposts = countNearLabels(candidate, "repost", "reposts", "retweet", "retweets", "rt")
		}
		if replies == 0 {
			replies = countNearLabels(candidate, "reply", "replies", "respuesta", "respuestas")
		}
	}
	return
}

func attributeValues(block string, attrs ...string) []string {
	vals := make([]string, 0, len(attrs))
	for _, attr := range attrs {
		pattern := fmt.Sprintf(`(?i)%s\s*=\s*(["'])(.*?)\1`, regexp.QuoteMeta(attr))
		re := regexp.MustCompile(pattern)
		for _, m := range re.FindAllStringSubmatch(block, -1) {
			if len(m) > 2 {
				vals = append(vals, collapseWhitespace(m[2]))
			}
		}
	}
	return uniqueStrings(vals)
}

func countNearLabels(s string, labels ...string) int {
	cleaned := collapseWhitespace(strings.ToLower(s))
	for _, label := range labels {
		label = regexp.QuoteMeta(strings.ToLower(label))
		patterns := []string{
			fmt.Sprintf(`(?i)%s[^0-9]{0,20}(%s)`, label, reCountLoose.String()),
			fmt.Sprintf(`(?i)(%s)[^a-z0-9]{0,20}%s`, reCountLoose.String(), label),
		}
		for _, pattern := range patterns {
			re := regexp.MustCompile(pattern)
			if m := re.FindStringSubmatch(cleaned); len(m) > 1 {
				return parseCountLoose(m[1])
			}
		}
	}
	return 0
}

func parseCountLoose(s string) int {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0
	}
	mult := 1.0
	if strings.HasSuffix(s, "k") {
		mult = 1_000
		s = strings.TrimSuffix(s, "k")
	} else if strings.HasSuffix(s, "m") {
		mult = 1_000_000
		s = strings.TrimSuffix(s, "m")
	}
	s = strings.ReplaceAll(s, " ", "")
	if strings.Contains(s, ",") && !strings.Contains(s, ".") {
		s = strings.ReplaceAll(s, ",", "")
	} else {
		s = strings.ReplaceAll(s, ",", ".")
	}
	if strings.Count(s, ".") == 1 {
		parts := strings.SplitN(s, ".", 2)
		if len(parts[1]) == 3 {
			s = parts[0] + parts[1]
		}
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return int(f * mult)
	}
	return 0
}

func textEntities(text string) (hashtags, mentions, links []string) {
	for _, m := range reHashtag.FindAllStringSubmatch(text, -1) {
		if len(m) > 1 {
			hashtags = append(hashtags, strings.ToLower(m[1]))
		}
	}
	for _, m := range reMention.FindAllStringSubmatch(text, -1) {
		if len(m) > 1 {
			mentions = append(mentions, strings.ToLower(m[1]))
		}
	}
	for _, m := range reURL.FindAllString(text, -1) {
		links = append(links, strings.TrimRight(m, `.,;:!?)\"'`))
	}
	return uniqueStrings(hashtags), uniqueStrings(mentions), uniqueStrings(links)
}

func uniqueStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func profileDisplayNameFromTitleOrMeta(title, metaTitle string) string {
	candidate := firstNonEmpty(metaTitle, title)
	candidate = collapseWhitespace(stripTags(candidate))
	if candidate == "" {
		return ""
	}
	if i := strings.Index(candidate, " (@"); i > 0 {
		return strings.TrimSpace(candidate[:i])
	}
	if i := strings.Index(candidate, " / X"); i > 0 {
		return strings.TrimSpace(candidate[:i])
	}
	if i := strings.Index(candidate, " • Instagram"); i > 0 {
		return strings.TrimSpace(candidate[:i])
	}
	return candidate
}

func cleanProfileDescription(desc string) string {
	desc = collapseWhitespace(stripTags(desc))
	if desc == "" {
		return ""
	}
	patterns := []string{
		`(?i)^\s*[0-9][0-9.,]*\s*[km]?[\s,·•|-]+followers?[\s,·•|-]+[0-9][0-9.,]*\s*[km]?[\s,·•|-]+following[\s,·•|-]+[0-9][0-9.,]*\s*[km]?[\s,·•|-]+posts?.*$`,
		`(?i)^\s*[0-9][0-9.,]*\s*[km]?[\s,·•|-]+followers?[\s,·•|-]+[0-9][0-9.,]*\s*[km]?[\s,·•|-]+following.*$`,
		`(?i)\s*See the latest conversations with @[^\s]+.*$`,
		`(?i)\s*See .*?profile.*$`,
	}
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		desc = re.ReplaceAllString(desc, "")
	}
	return collapseWhitespace(desc)
}

func parseProfileCounts(text string) (followers, following, posts int) {
	followers = countNearLabels(text, "followers", "seguidores")
	following = countNearLabels(text, "following", "siguiendo")
	posts = countNearLabels(text, "posts", "publicaciones")
	return
}

func parsePostCandidate(block string, username string, fallbackIndex int) (tweetCandidate, bool) {
	text := tweetTextFromBlock(block)
	if text == "" {
		return tweetCandidate{}, false
	}
	likes, reposts, replies := tweetCountsFromBlock(block)
	createdAt := tweetCreatedAtFromBlock(block)
	lang := tweetLanguageFromBlock(block)
	hashtags, mentions, links := textEntities(text)
	media := strings.Contains(strings.ToLower(block), "tweetphoto") || strings.Contains(strings.ToLower(block), "video") || strings.Contains(strings.ToLower(block), "img")
	if createdAt.IsZero() {
		createdAt = time.Now().Add(-time.Duration(fallbackIndex) * time.Hour)
	}
	return tweetCandidate{
		ID:        fmt.Sprintf("%s-%d", username, time.Now().UnixNano()+int64(fallbackIndex)),
		Text:      text,
		Likes:     likes,
		Reposts:   reposts,
		Replies:   replies,
		CreatedAt: createdAt,
		Hashtags:  hashtags,
		Mentions:  mentions,
		Links:     links,
		Media:     media,
		Language:  lang,
	}, true
}

type tweetCandidate struct {
	ID        string
	Text      string
	Likes     int
	Reposts   int
	Replies   int
	CreatedAt time.Time
	Hashtags  []string
	Mentions  []string
	Links     []string
	Media     bool
	Language  string
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
