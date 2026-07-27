package analyzer

import (
	"strings"

	"layerone-scout/pkg/utils"
)

var sentimentPhraseLexicon = map[string]float64{
	"me encanta": 0.9,
	"me fascina": 0.9,
	"me alegra": 0.7,
	"me gusta": 0.6,
	"me preocupa": -0.6,
	"no me gusta": -0.8,
	"no puedo": -0.5,
	"odio": -0.9,
	"estoy feliz": 0.8,
	"estoy triste": -0.8,
	"muy bueno": 0.7,
	"muy malo": -0.8,
	"buen trabajo": 0.7,
	"mal trabajo": -0.7,
	"que bien": 0.6,
	"que mal": -0.6,
}

var sentimentTokenLexicon = map[string]float64{
	"feliz": 0.8, "alegre": 0.7, "contento": 0.6, "genial": 0.7,
	"maravilloso": 0.8, "excelente": 0.7, "bueno": 0.5,
	"triste": -0.7, "deprimido": -0.8, "enojado": -0.6, "frustrado": -0.7,
	"horrible": -0.8, "malo": -0.5, "terrible": -0.8,
	"ansioso": -0.6, "preocupado": -0.5, "tranquilo": 0.4, "sereno": 0.5,
	"amor": 0.8, "odio": -0.9, "gracias": 0.4, "perdon": -0.2,
}

var negators = map[string]struct{}{
	"no": {}, "nunca": {}, "jamas": {}, "sin": {}, "ni": {},
}

func AnalyzeSentimentPlus(texts []string) float64 {
	total := 0.0
	count := 0
	for _, text := range texts {
		low := strings.ToLower(utils.NormalizeText(text))
		for phrase, value := range sentimentPhraseLexicon {
			if strings.Contains(low, phrase) {
				total += value
				count++
			}
		}
		tokens := utils.Tokenize(text)
		pendingNegation := 0
		for _, token := range tokens {
			t := normalizeToken(token)
			if _, ok := negators[t]; ok {
				pendingNegation = 3
				continue
			}
			value, ok := sentimentTokenLexicon[t]
			if !ok {
				if pendingNegation > 0 {
					pendingNegation--
				}
				continue
			}
			if pendingNegation > 0 {
				value = -value * 0.75
				pendingNegation--
			}
			total += value
			count++
		}
	}
	if count == 0 {
		return 0
	}
	result := total / float64(count)
	if result > 1 {
		return 1
	}
	if result < -1 {
		return -1
	}
	return result
}
