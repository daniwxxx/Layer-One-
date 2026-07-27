package analyzer

import (
	"math"

	"layerone-scout/pkg/utils"
)

// Simple TF-IDF para extraer palabras clave de un conjunto de textos.
// No se usa directamente en el análisis, pero puede ser útil para enriquecer intereses.
// Esta es una implementación básica para demostración.
func TFIDFKeywords(texts []string, topN int) []string {
	// Document frequency
	docFreq := make(map[string]int)
	for _, text := range texts {
		tokens := utils.Tokenize(text)
		seen := make(map[string]bool)
		for _, t := range tokens {
			if !seen[t] {
				seen[t] = true
				docFreq[t]++
			}
		}
	}
	N := len(texts)
	// Score each token
	scores := make(map[string]float64)
	for _, text := range texts {
		tokens := utils.Tokenize(text)
		tf := make(map[string]int)
		for _, t := range tokens {
			tf[t]++
		}
		for t, count := range tf {
			idf := math.Log(float64(N) / float64(docFreq[t]+1))
			scores[t] += float64(count) * idf
		}
	}
	// Sort by score
	type pair struct {
		word  string
		score float64
	}
	var pairs []pair
	for w, s := range scores {
		pairs = append(pairs, pair{w, s})
	}
	for i := 0; i < len(pairs)-1; i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[i].score < pairs[j].score {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}
	if len(pairs) > topN {
		pairs = pairs[:topN]
	}
	var result []string
	for _, p := range pairs {
		result = append(result, p.word)
	}
	return result
}
