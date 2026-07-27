package analyzer

import (
	"math"
	"sort"
	"strings"
	"sync"
	"unicode"

	"layerone-scout/pkg/utils"
)

type CorpusMetrics struct {
	TokenCount        int
	UniqueTokenCount   int
	LexicalDiversity   float64
	ShannonEntropy     float64
	ShannonNormalized  float64
	ZipfSlope          float64
	ZipfFit            float64
}

var (
	normalizedLexiconOnce sync.Once
	normalizedLexicon     map[string]map[string]float64
)

func AnalyzeCorpus(texts []string) (CorpusMetrics, map[string]int) {
	freq := buildTokenFrequency(texts)
	metrics := CorpusMetrics{TokenCount: sumFreq(freq), UniqueTokenCount: len(freq)}
	if metrics.TokenCount == 0 {
		return metrics, freq
	}
	metrics.LexicalDiversity = float64(metrics.UniqueTokenCount) / float64(metrics.TokenCount)
	metrics.ShannonEntropy = shannonEntropy(freq)
	if metrics.UniqueTokenCount > 1 {
		metrics.ShannonNormalized = metrics.ShannonEntropy / math.Log2(float64(metrics.UniqueTokenCount))
	}
	if metrics.ShannonNormalized < 0 {
		metrics.ShannonNormalized = 0
	}
	if metrics.ShannonNormalized > 1 {
		metrics.ShannonNormalized = 1
	}
	metrics.ZipfSlope = estimateZipfSlope(freq)
	metrics.ZipfFit = 1 / (1 + math.Abs(metrics.ZipfSlope+1))
	return metrics, freq
}

func buildTokenFrequency(texts []string) map[string]int {
	freq := make(map[string]int)
	for _, text := range texts {
		for _, token := range utils.Tokenize(text) {
			token = normalizeToken(token)
			if token == "" || !containsLetter(token) {
				continue
			}
			freq[token]++
		}
	}
	return freq
}

func sumFreq(freq map[string]int) int {
	total := 0
	for _, n := range freq {
		total += n
	}
	return total
}

func shannonEntropy(freq map[string]int) float64 {
	total := sumFreq(freq)
	if total == 0 {
		return 0
	}
	entropy := 0.0
	for _, n := range freq {
		p := float64(n) / float64(total)
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}
	return entropy
}

func estimateZipfSlope(freq map[string]int) float64 {
	if len(freq) < 2 {
		return 0
	}
	type pair struct{ token string; count int }
	pairs := make([]pair, 0, len(freq))
	for token, count := range freq {
		pairs = append(pairs, pair{token: token, count: count})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].count == pairs[j].count {
			return pairs[i].token < pairs[j].token
		}
		return pairs[i].count > pairs[j].count
	})

	n := float64(len(pairs))
	if n < 2 {
		return 0
	}
	var sumX, sumY, sumXY, sumXX float64
	for i, p := range pairs {
		x := math.Log(float64(i + 1))
		y := math.Log(float64(p.count))
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}
	nf := float64(len(pairs))
	denom := nf*sumXX - sumX*sumX
	if denom == 0 {
		return 0
	}
	return (nf*sumXY - sumX*sumY) / denom
}

func normalizedLexiconMap() map[string]map[string]float64 {
	normalizedLexiconOnce.Do(func() {
		normalizedLexicon = make(map[string]map[string]float64, len(lexicon))
		for trait, dict := range lexicon {
			normalizedLexicon[trait] = make(map[string]float64, len(dict))
			for term, weight := range dict {
				normalizedLexicon[trait][normalizeToken(term)] = weight
			}
		}
	})
	return normalizedLexicon
}

func normalizeToken(token string) string {
	return strings.TrimSpace(utils.NormalizeText(strings.ToLower(token)))
}

func containsLetter(token string) bool {
	for _, r := range token {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func logit(p float64) float64 {
	if p <= 0 {
		return math.Inf(-1)
	}
	if p >= 1 {
		return math.Inf(1)
	}
	return math.Log(p / (1 - p))
}

func logistic(x float64) float64 {
	if x >= 0 {
		z := math.Exp(-x)
		return 1 / (1 + z)
	}
	z := math.Exp(x)
	return z / (1 + z)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
