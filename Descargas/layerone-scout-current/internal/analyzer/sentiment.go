package analyzer

import (
	"layerone-scout/pkg/utils"
)

var sentimentLexicon = map[string]float64{
	"feliz": 0.8, "alegre": 0.7, "contento": 0.6, "genial": 0.7,
	"maravilloso": 0.8, "excelente": 0.7, "bueno": 0.5,
	"triste": -0.7, "deprimido": -0.8, "enojado": -0.6, "frustrado": -0.7,
	"horrible": -0.8, "malo": -0.5, "terrible": -0.8,
}

func AnalyzeSentiment(texts []string) float64 {
	total := 0.0
	count := 0
	for _, text := range texts {
		tokens := utils.Tokenize(text)
		for _, token := range tokens {
			token = normalizeToken(token)
			if val, ok := sentimentLexicon[token]; ok {
				total += val
				count++
			}
		}
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}
