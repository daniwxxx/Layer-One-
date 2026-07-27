package analyzer

import (
	"math"
	"sort"
	"strings"
	"time"

	"layerone-scout/internal/model"
	"layerone-scout/pkg/utils"
)

type BehaviorMetrics struct {
	Burstiness        float64
	Regularity        float64
	QuestionRate      float64
	ExclamationRate   float64
	HashtagRate       float64
	MentionRate       float64
	LinkRate          float64
	AvgPostLength     float64
	ActiveSpanDays    float64
	PostsPerActiveDay float64
}

func AnalyzeBehavior(posts []model.Post, texts []string) BehaviorMetrics {
	m := BehaviorMetrics{}
	if len(posts) == 0 {
		return m
	}

	m.AvgPostLength = averagePostLength(texts)
	m.QuestionRate, m.ExclamationRate = punctuationRates(texts)
	m.HashtagRate, m.MentionRate, m.LinkRate = entityRates(posts)
	m.ActiveSpanDays, m.PostsPerActiveDay = activitySpan(posts)
	m.Burstiness, m.Regularity = postingBurstiness(posts)
	return m
}

func averagePostLength(texts []string) float64 {
	if len(texts) == 0 {
		return 0
	}
	total := 0
	for _, text := range texts {
		total += len(utils.Tokenize(text))
	}
	return float64(total) / float64(len(texts))
}

func punctuationRates(texts []string) (questionRate, exclamationRate float64) {
	if len(texts) == 0 {
		return 0, 0
	}
	questions := 0
	exclaims := 0
	for _, text := range texts {
		if strings.Contains(text, "?") {
			questions++
		}
		if strings.Contains(text, "!") {
			exclaims++
		}
	}
	return float64(questions) / float64(len(texts)), float64(exclaims) / float64(len(texts))
}

func entityRates(posts []model.Post) (hashtagRate, mentionRate, linkRate float64) {
	if len(posts) == 0 {
		return 0, 0, 0
	}
	h := 0
	m := 0
	l := 0
	for _, post := range posts {
		h += len(post.Hashtags)
		m += len(post.Mentions)
		l += len(post.Links)
	}
	den := float64(len(posts))
	return float64(h) / den, float64(m) / den, float64(l) / den
}

func activitySpan(posts []model.Post) (spanDays, postsPerDay float64) {
	if len(posts) == 0 {
		return 0, 0
	}
	ordered := make([]model.Post, len(posts))
	copy(ordered, posts)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].CreatedAt.Before(ordered[j].CreatedAt) })
	first := ordered[0].CreatedAt
	last := ordered[len(ordered)-1].CreatedAt
	spanDays = last.Sub(first).Hours() / 24
	if spanDays <= 0 {
		spanDays = 0
		postsPerDay = float64(len(posts))
		return
	}
	postsPerDay = float64(len(posts)) / spanDays
	return
}

func postingBurstiness(posts []model.Post) (burstiness, regularity float64) {
	if len(posts) < 2 {
		return 0, 1
	}
	ordered := make([]model.Post, len(posts))
	copy(ordered, posts)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].CreatedAt.Before(ordered[j].CreatedAt) })

	intervals := make([]float64, 0, len(ordered)-1)
	for i := 1; i < len(ordered); i++ {
		hours := ordered[i].CreatedAt.Sub(ordered[i-1].CreatedAt).Hours()
		if hours > 0 {
			intervals = append(intervals, hours)
		}
	}
	if len(intervals) == 0 {
		return 0, 1
	}
	mean := 0.0
	for _, v := range intervals {
		mean += v
	}
	mean /= float64(len(intervals))
	variance := 0.0
	for _, v := range intervals {
		d := v - mean
		variance += d * d
	}
	variance /= float64(len(intervals))
	sd := math.Sqrt(variance)
	cv := 0.0
	if mean > 0 {
		cv = sd / mean
	}
	burstiness = clamp01(cv / (1 + cv))
	regularity = clamp01(1 - burstiness)
	return
}
