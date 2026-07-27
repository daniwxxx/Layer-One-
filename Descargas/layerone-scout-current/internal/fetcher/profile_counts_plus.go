package fetcher

import (
	"regexp"
	"strings"
)

var countWindowRE = regexp.MustCompile(`[0-9][0-9.,]*\s*[kKmM]?`)

func parseProfileCountsSmart(text string) (followers, following, posts int) {
	norm := collapseWhitespace(strings.ToLower(text))
	followers = countNearLabelSmart(norm, []string{"followers", "seguidores"})
	following = countNearLabelSmart(norm, []string{"following", "siguiendo"})
	posts = countNearLabelSmart(norm, []string{"posts", "publicaciones"})
	return
}

func countNearLabelSmart(text string, labels []string) int {
	for _, label := range labels {
		if v := countAroundLabel(text, label); v > 0 {
			return v
		}
	}
	return 0
}

func countAroundLabel(text, label string) int {
	text = collapseWhitespace(strings.ToLower(text))
	label = strings.ToLower(strings.TrimSpace(label))
	idx := strings.Index(text, label)
	if idx < 0 {
		return 0
	}

	before := strings.TrimSpace(text[max(0, idx-32):idx])
	after := strings.TrimSpace(text[min(len(text), idx+len(label)):min(len(text), idx+len(label)+32)])

	if v := countWindowBefore(before); v > 0 {
		return v
	}
	if v := countWindowAfter(after); v > 0 {
		return v
	}
	return 0
}

func countWindowBefore(s string) int {
	matches := countWindowRE.FindAllString(s, -1)
	if len(matches) == 0 {
		return 0
	}
	return parseCountLoose(matches[len(matches)-1])
}

func countWindowAfter(s string) int {
	matches := countWindowRE.FindAllString(s, -1)
	if len(matches) == 0 {
		return 0
	}
	return parseCountLoose(matches[0])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
