package analyzer

import (
	"math"
)

func ComputeBigFive(texts []string) (map[string]float64, map[string]float64) {
	metrics, freq := AnalyzeCorpus(texts)
	result := map[string]float64{
		"openness": 0.5, "conscientiousness": 0.5, "extraversion": 0.5,
		"agreeableness": 0.5, "neuroticism": 0.5,
	}
	conf := map[string]float64{
		"openness": 0, "conscientiousness": 0, "extraversion": 0,
		"agreeableness": 0, "neuroticism": 0,
	}
	if metrics.TokenCount == 0 {
		return result, conf
	}

	lex := normalizedLexiconMap()
	for trait, dict := range lex {
		var evidence float64
		matchedTokens := 0
		for token, count := range freq {
			weight, ok := dict[token]
			if !ok {
				continue
			}
			matchedTokens += count
			freqBoost := 1 + 0.35*math.Log1p(float64(count))
			rankBoost := 1 + 0.20*(1-metrics.ZipfFit)
			evidence += weight * freqBoost * rankBoost
		}

		posterior := logistic(logit(0.5) + evidence/(1.4+0.8*(1-metrics.LexicalDiversity)))
		result[trait] = clamp01(posterior)

		coverage := float64(matchedTokens) / float64(metrics.TokenCount)
		signalStrength := math.Tanh(math.Abs(evidence) / 2.2)
		entropySignal := 1 - metrics.ShannonNormalized
		zipfSignal := metrics.ZipfFit
		conf[trait] = clamp01(0.32*coverage + 0.33*signalStrength + 0.20*entropySignal + 0.15*zipfSignal)
	}

	return result, conf
}
