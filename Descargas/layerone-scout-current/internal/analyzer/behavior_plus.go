package analyzer

import (
	"math"
	"sort"
	"strings"

	"layerone-scout/internal/model"
	"layerone-scout/pkg/utils"
)

var firstPersonTokens = map[string]struct{}{
	"yo": {}, "me": {}, "mi": {}, "mio": {}, "mía": {}, "nos": {}, "nosotros": {}, "nuestra": {}, "nuestro": {},
}

var secondPersonTokens = map[string]struct{}{
	"tu": {}, "tú": {}, "te": {}, "ti": {}, "usted": {}, "ustedes": {}, "vos": {}, "vosotros": {}, "les": {},
}

var ctaPhrases = []string{
	"comenta", "comentá", "opina", "seguime", "sígueme", "sigue", "compartí", "comparte", "reenvía", "manda dm",
	"envía dm", "link in bio", "link en bio", "haz clic", "hacé clic", "join", "subscribe", "suscríbete", "únete",
}

type BehaviorMetricsPlus struct {
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
	SentenceCount     int
	MediaRate         float64
	RepostRate        float64
	ReplyRate         float64
	FirstPersonRate   float64
	SecondPersonRate  float64
	CallToActionRate  float64
	TemporalEntropy   float64
	RepetitionScore   float64
}

func AnalyzeBehaviorPlus(posts []model.Post, texts []string) BehaviorMetricsPlus {
	if len(posts) == 0 && len(texts) == 0 {
		return BehaviorMetricsPlus{}
	}
	m := BehaviorMetricsPlus{}
	m.AvgPostLength = averagePostLengthPlus(texts)
	m.QuestionRate, m.ExclamationRate = punctuationRatesPlus(texts)
	m.HashtagRate, m.MentionRate, m.LinkRate = entityRatesPlus(posts)
	m.ActiveSpanDays, m.PostsPerActiveDay = activitySpanPlus(posts)
	m.Burstiness, m.Regularity = postingBurstinessPlus(posts)
	m.SentenceCount = sentenceCount(texts)
	m.MediaRate = mediaRate(posts)
	m.RepostRate, m.ReplyRate = activityActionRates(posts)
	m.FirstPersonRate, m.SecondPersonRate = pronounRates(texts)
	m.CallToActionRate = callToActionRate(texts)
	m.TemporalEntropy = temporalEntropy(posts)
	m.RepetitionScore = repetitionScore(texts)
	return m
}

func averagePostLengthPlus(texts []string) float64 {
	if len(texts) == 0 {
		return 0
	}
	total := 0
	for _, text := range texts {
		total += len(utils.Tokenize(text))
	}
	return float64(total) / float64(len(texts))
}

func punctuationRatesPlus(texts []string) (questionRate, exclamationRate float64) {
	if len(texts) == 0 {
		return 0, 0
	}
	questions, exclaims := 0, 0
	for _, text := range texts {
		if strings.Contains(text, "?") {
			questions++
		}
		if strings.Contains(text, "!") {
			exclaims++
		}
	}
	den := float64(len(texts))
	return float64(questions) / den, float64(exclaims) / den
}

func entityRatesPlus(posts []model.Post) (hashtagRate, mentionRate, linkRate float64) {
	if len(posts) == 0 {
		return 0, 0, 0
	}
	h, m, l := 0, 0, 0
	for _, post := range posts {
		h += len(post.Hashtags)
		m += len(post.Mentions)
		l += len(post.Links)
	}
	den := float64(len(posts))
	return float64(h) / den, float64(m) / den, float64(l) / den
}

func activitySpanPlus(posts []model.Post) (spanDays, postsPerDay float64) {
	if len(posts) == 0 {
		return 0, 0
	}
	ordered := append([]model.Post(nil), posts...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].CreatedAt.Before(ordered[j].CreatedAt) })
	spanDays = ordered[len(ordered)-1].CreatedAt.Sub(ordered[0].CreatedAt).Hours() / 24
	if spanDays <= 0 {
		return 0, float64(len(posts))
	}
	return spanDays, float64(len(posts)) / spanDays
}

func postingBurstinessPlus(posts []model.Post) (burstiness, regularity float64) {
	if len(posts) < 2 {
		return 0, 1
	}
	ordered := append([]model.Post(nil), posts...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].CreatedAt.Before(ordered[j].CreatedAt) })
	intervals := make([]float64, 0, len(ordered)-1)
	for i := 1; i < len(ordered); i++ {
		if h := ordered[i].CreatedAt.Sub(ordered[i-1].CreatedAt).Hours(); h > 0 {
			intervals = append(intervals, h)
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
	sd := math.Sqrt(variance / float64(len(intervals)))
	cv := 0.0
	if mean > 0 {
		cv = sd / mean
	}
	burstiness = clamp01(cv / (1 + cv))
	regularity = clamp01(1 - burstiness)
	return
}

func sentenceCount(texts []string) int {
	count := 0
	for _, text := range texts {
		for _, r := range text {
			if r == '.' || r == '!' || r == '?' || r == '…' {
				count++
			}
		}
	}
	return count
}

func mediaRate(posts []model.Post) float64 {
	if len(posts) == 0 {
		return 0
	}
	media := 0
	for _, post := range posts {
		if post.Media {
			media++
		}
	}
	return float64(media) / float64(len(posts))
}

func activityActionRates(posts []model.Post) (repostRate, replyRate float64) {
	if len(posts) == 0 {
		return 0, 0
	}
	reposts, replies := 0, 0
	for _, post := range posts {
		reposts += post.Reposts
		replies += post.Replies
	}
	den := float64(len(posts))
	return float64(reposts) / den, float64(replies) / den
}

func pronounRates(texts []string) (firstPersonRate, secondPersonRate float64) {
	totalTokens := 0
	first, second := 0, 0
	for _, text := range texts {
		tokens := utils.Tokenize(text)
		totalTokens += len(tokens)
		for _, token := range tokens {
			t := normalizeToken(token)
			if _, ok := firstPersonTokens[t]; ok {
				first++
			}
			if _, ok := secondPersonTokens[t]; ok {
				second++
			}
		}
	}
	if totalTokens == 0 {
		return 0, 0
	}
	den := float64(totalTokens)
	return float64(first) / den, float64(second) / den
}

func callToActionRate(texts []string) float64 {
	if len(texts) == 0 {
		return 0
	}
	match := 0
	for _, text := range texts {
		low := strings.ToLower(utils.NormalizeText(text))
		for _, phrase := range ctaPhrases {
			if strings.Contains(low, phrase) {
				match++
				break
			}
		}
	}
	return float64(match) / float64(len(texts))
}

func temporalEntropy(posts []model.Post) float64 {
	if len(posts) < 2 {
		return 0
	}
	buckets := make(map[int]int)
	for _, post := range posts {
		if !post.CreatedAt.IsZero() {
			buckets[post.CreatedAt.UTC().Hour()]++
		}
	}
	if len(buckets) < 2 {
		return 0
	}
	total := 0
	for _, n := range buckets {
		total += n
	}
	entropy := 0.0
	for _, n := range buckets {
		p := float64(n) / float64(total)
		entropy -= p * math.Log2(p)
	}
	return clamp01(entropy / math.Log2(24))
}

func repetitionScore(texts []string) float64 {
	if len(texts) < 2 {
		return 0
	}
	sum, pairs := 0.0, 0
	for i := 1; i < len(texts); i++ {
		a := tokenSet(texts[i-1])
		b := tokenSet(texts[i])
		if len(a) == 0 || len(b) == 0 {
			continue
		}
		sum += jaccard(a, b)
		pairs++
	}
	if pairs == 0 {
		return 0
	}
	return clamp01(sum / float64(pairs))
}

func tokenSet(text string) map[string]struct{} {
	set := make(map[string]struct{})
	for _, token := range utils.Tokenize(text) {
		if norm := normalizeToken(token); norm != "" {
			set[norm] = struct{}{}
		}
	}
	return set
}

func jaccard(a, b map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	inter := 0
	for k := range a {
		if _, ok := b[k]; ok {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}
