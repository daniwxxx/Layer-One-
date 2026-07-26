package analyzer

import (
	"layerone-scout/pkg/utils"
)

var lexicon = map[string]map[string]float64{
	"openness": {
		"creativo": 0.8, "imaginativo": 0.7, "curioso": 0.6, "aventurero": 0.7,
		"artístico": 0.8, "innovador": 0.7, "abstracto": 0.5, "complejo": 0.4,
		"tradicional": -0.6, "convencional": -0.7, "rutinario": -0.5,
		"explorador": 0.6, "original": 0.7, "visionario": 0.8,
		"cerrado": -0.6, "conservador": -0.5, "filosófico": 0.5,
	},
	"conscientiousness": {
		"ordenado": 0.8, "responsable": 0.7, "disciplinado": 0.7, "eficiente": 0.6,
		"organizado": 0.8, "planificador": 0.6, "persistente": 0.7,
		"desordenado": -0.6, "irresponsable": -0.7, "impulsivo": -0.5,
		"cumplidor": 0.6, "meticuloso": 0.7, "formal": 0.5, "productivo": 0.6,
	},
	"extraversion": {
		"social": 0.7, "extrovertido": 0.8, "energético": 0.6, "animado": 0.7,
		"divertido": 0.6, "sociable": 0.8, "hablador": 0.5,
		"tímido": -0.6, "reservado": -0.7, "introvertido": -0.8,
		"fiesta": 0.7, "carismático": 0.7, "vida de la fiesta": 0.8,
	},
	"agreeableness": {
		"amable": 0.7, "empático": 0.8, "compasivo": 0.7, "cooperativo": 0.6,
		"generoso": 0.6, "confiable": 0.5, "respetuoso": 0.5,
		"hostil": -0.7, "egoísta": -0.6, "competitivo": -0.4,
		"colaborador": 0.6, "solidario": 0.7, "cordial": 0.5, "amigable": 0.7,
	},
	"neuroticism": {
		"ansioso": 0.8, "inseguro": 0.7, "preocupado": 0.6, "estresado": 0.7,
		"deprimido": 0.8, "volátil": 0.5,
		"tranquilo": -0.6, "sereno": -0.7, "equilibrado": -0.8,
		"nervioso": 0.7, "sensible": 0.5, "estable": -0.6, "temeroso": 0.7,
	},
}

func ComputeBigFive(texts []string) (map[string]float64, map[string]float64) {
	scores := map[string]float64{
		"openness": 0, "conscientiousness": 0, "extraversion": 0,
		"agreeableness": 0, "neuroticism": 0,
	}
	counts := map[string]int{
		"openness": 0, "conscientiousness": 0, "extraversion": 0,
		"agreeableness": 0, "neuroticism": 0,
	}
	for _, text := range texts {
		tokens := utils.Tokenize(text)
		for _, token := range tokens {
			for trait, dict := range lexicon {
				if weight, ok := dict[token]; ok {
					scores[trait] += weight
					counts[trait]++
				}
			}
		}
	}
	result := map[string]float64{}
	conf := map[string]float64{}
	for trait := range scores {
		if counts[trait] > 0 {
			avg := scores[trait] / float64(counts[trait])
			val := (avg + 1) / 2
			if val < 0 {
				val = 0
			} else if val > 1 {
				val = 1
			}
			result[trait] = val
			c := float64(counts[trait]) / 10.0
			if c > 1 {
				c = 1
			}
			conf[trait] = c
		} else {
			result[trait] = 0.5
			conf[trait] = 0.0
		}
	}
	return result, conf
}
